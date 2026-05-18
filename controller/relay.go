package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/middleware"
	dbmodel "github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/monitor"
	relaybilling "github.com/pagoda-inference/one-api/relay/billing"
	billingratio "github.com/pagoda-inference/one-api/relay/billing/ratio"
	"github.com/pagoda-inference/one-api/relay/controller"
	relaymeta "github.com/pagoda-inference/one-api/relay/meta"
	"github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
)

// AnthropicRequest is the incoming Anthropic /v1/messages format
type AnthropicRequest struct {
	Model         string                  `json:"model"`
	Messages      []AnthropicMessage      `json:"messages"`
	System        any                     `json:"system,omitempty"` // string or []SystemBlock
	MaxTokens     int                     `json:"max_tokens"`
	Stream        bool                    `json:"stream,omitempty"`
	StreamOptions *AnthropicStreamOptions `json:"stream_options,omitempty"`
	Temperature   *float64                `json:"temperature,omitempty"`
	TopP          *float64                `json:"top_p,omitempty"`
	TopK          *int                    `json:"top_k,omitempty"`
	Tools         []AnthropicTool         `json:"tools,omitempty"`
	ToolChoice    any                     `json:"tool_choice,omitempty"` // string or ToolChoiceBlock
	Metadata      map[string]any          `json:"metadata,omitempty"`
	StopSequences []string                `json:"stop_sequences,omitempty"`
	Thinking      *AnthropicThinking      `json:"thinking,omitempty"`
	Betas         []string                `json:"betas,omitempty"`
	ExtraFields   map[string]any          `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve unknown fields
func (r *AnthropicRequest) UnmarshalJSON(data []byte) error {
	type Alias AnthropicRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Collect unknown fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	knownKeys := map[string]struct{}{
		"model": {}, "messages": {}, "system": {}, "max_tokens": {},
		"stream": {}, "stream_options": {}, "temperature": {}, "top_p": {},
		"top_k": {}, "tools": {}, "tool_choice": {}, "metadata": {},
		"stop_sequences": {}, "thinking": {}, "betas": {},
	}
	for key := range knownKeys {
		delete(raw, key)
	}
	if len(raw) > 0 {
		r.ExtraFields = raw
	}
	return nil
}

type AnthropicStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type AnthropicThinking struct {
	Type         string `json:"type,omitempty"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
	Display      string `json:"display,omitempty"` // "summarized" | "omitted"
}

type AnthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"` // object or map
}

type ToolChoiceBlock struct {
	Type string `json:"type"` // "auto", "any", "tool"
	Name string `json:"name,omitempty"`
}

type SystemBlock struct {
	Type string `json:"type"` // always "text"
	Text string `json:"text"`
}

var hopByHopResponseHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
	"proxy-connection":    {},
	"content-length":      {},
}

func copySafeResponseHeaders(c *gin.Context, src http.Header) {
	for k, vv := range src {
		lk := strings.ToLower(strings.TrimSpace(k))
		if _, skip := hopByHopResponseHeaders[lk]; skip {
			continue
		}
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
}

func stringifyToolArguments(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func mergeToolArguments(existing string, incoming any) string {
	switch x := incoming.(type) {
	case nil:
		return existing
	case string:
		part := strings.TrimSpace(x)
		if part == "" {
			return existing
		}
		// Some backends emit full JSON snapshots per chunk instead of deltas.
		// When we detect a complete JSON value, replace instead of append.
		if (strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")) ||
			(strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]")) {
			return part
		}
		// Streaming OpenAI-compatible backends usually send JSON fragments as strings.
		return existing + x
	default:
		// If backend sends object/array directly, treat it as the full argument payload.
		b, err := json.Marshal(x)
		if err != nil {
			return existing
		}
		return string(b)
	}
}

func parseToolInputJSON(args string) (any, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return nil, false
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return nil, false
	}
	return v, true
}

var toolCallXMLRe = regexp.MustCompile(`(?is)<tool_call>\s*(\{.*?\})\s*</tool_call>`)
var kvLineRe = regexp.MustCompile(`(?m)^\s*[-*]?\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.+?)\s*$`)

func trimQuotedValue(v string) string {
	s := strings.TrimSpace(v)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

func tryExtractToolUseFromYamlLike(input string) (name string, toolInput any, ok bool) {
	if strings.TrimSpace(input) == "" {
		return "", nil, false
	}
	matches := kvLineRe.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return "", nil, false
	}
	kv := make(map[string]string)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(m[1]))
		v := trimQuotedValue(m[2])
		if k == "" || v == "" {
			continue
		}
		kv[k] = v
	}
	if len(kv) == 0 {
		return "", nil, false
	}

	toolName := strings.TrimSpace(kv["tool_name"])
	if toolName == "" {
		toolName = strings.TrimSpace(kv["name"])
	}
	if toolName == "" {
		toolName = strings.TrimSpace(kv["tool"])
	}
	// Heuristic: if common write args are present, infer Write.
	if toolName == "" {
		if kv["file_path"] != "" && kv["content"] != "" {
			toolName = "Write"
		}
	}
	if toolName == "" {
		return "", nil, false
	}

	inputMap := make(map[string]any)
	for k, v := range kv {
		switch k {
		case "tool_name", "name", "tool", "function":
			continue
		default:
			inputMap[k] = v
		}
	}
	if len(inputMap) == 0 {
		return "", nil, false
	}
	return toolName, inputMap, true
}

func tryExtractToolUseFromText(input string) (name string, toolInput any, ok bool) {
	if strings.TrimSpace(input) == "" {
		return "", nil, false
	}
	matches := toolCallXMLRe.FindAllStringSubmatch(input, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			continue
		}
		n, _ := payload["name"].(string)
		n = strings.TrimSpace(n)
		if n == "" {
			// Some variants use function.name
			if f, ok := payload["function"].(map[string]any); ok {
				if fn, ok := f["name"].(string); ok {
					n = strings.TrimSpace(fn)
				}
				if n == "" {
					if fn, ok := f["tool_name"].(string); ok {
						n = strings.TrimSpace(fn)
					}
				}
			}
		}
		if n == "" {
			continue
		}
		if args, exists := payload["arguments"]; exists {
			switch v := args.(type) {
			case map[string]any:
				return n, v, true
			case string:
				if parsed, ok := parseToolInputJSON(v); ok {
					return n, parsed, true
				}
			}
		}
		if in, exists := payload["input"]; exists {
			switch v := in.(type) {
			case map[string]any:
				return n, v, true
			case string:
				if parsed, ok := parseToolInputJSON(v); ok {
					return n, parsed, true
				}
			}
		}
	}
	if n, in, ok := tryExtractToolUseFromYamlLike(input); ok {
		return n, in, true
	}
	return "", nil, false
}

var thinkBlockRe = regexp.MustCompile(`(?is)<think>(.*?)</think>`)

func splitThinkBlocksFromText(input string) (cleanText string, thinkingText string) {
	if input == "" {
		return "", ""
	}
	var thinkParts []string
	matches := thinkBlockRe.FindAllStringSubmatch(input, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			part := strings.TrimSpace(m[1])
			if part != "" {
				thinkParts = append(thinkParts, part)
			}
		}
	}
	clean := thinkBlockRe.ReplaceAllString(input, "")

	// Handle malformed streams that emit orphan "</think>" without a matching "<think>".
	// In this case, treat the prefix before each orphan close tag as leaked thinking content.
	rest := clean
	var cleanedParts []string
	for {
		idx := strings.Index(strings.ToLower(rest), "</think>")
		if idx < 0 {
			break
		}
		part := strings.TrimSpace(rest[:idx])
		if part != "" {
			thinkParts = append(thinkParts, part)
		}
		rest = rest[idx+len("</think>"):]
	}
	cleanedParts = append(cleanedParts, rest)
	clean = strings.Join(cleanedParts, "")
	clean = strings.ReplaceAll(clean, "<think>", "")
	clean = strings.ReplaceAll(clean, "</think>", "")
	clean = strings.TrimSpace(clean)
	return clean, strings.Join(thinkParts, "\n")
}

func longestSuffixPrefix(s, pattern string) int {
	max := len(pattern) - 1
	if max > len(s) {
		max = len(s)
	}
	for k := max; k > 0; k-- {
		if strings.HasSuffix(s, pattern[:k]) {
			return k
		}
	}
	return 0
}

func splitThinkTaggedChunk(chunk string, inThink *bool, carry *string) (textOut string, thinkingOut string) {
	input := *carry + chunk
	*carry = ""
	for len(input) > 0 {
		if *inThink {
			if idx := strings.Index(strings.ToLower(input), "</think>"); idx >= 0 {
				thinkingOut += input[:idx]
				input = input[idx+len("</think>"):]
				*inThink = false
			} else {
				k := longestSuffixPrefix(strings.ToLower(input), "</think>")
				thinkingOut += input[:len(input)-k]
				*carry = input[len(input)-k:]
				input = ""
			}
			continue
		}

		// Malformed case: orphan closing tag appears before any opening tag.
		// Treat preceding text as leaked thinking instead of user-visible text.
		if closeIdx := strings.Index(strings.ToLower(input), "</think>"); closeIdx >= 0 {
			if openIdx := strings.Index(strings.ToLower(input), "<think>"); openIdx < 0 || closeIdx < openIdx {
				thinkingOut += input[:closeIdx]
				input = input[closeIdx+len("</think>"):]
				continue
			}
		}

		if idx := strings.Index(strings.ToLower(input), "<think>"); idx >= 0 {
			textOut += input[:idx]
			input = input[idx+len("<think>"):]
			*inThink = true
			continue
		}
		k := longestSuffixPrefix(strings.ToLower(input), "<think>")
		textOut += input[:len(input)-k]
		*carry = input[len(input)-k:]
		input = ""
	}
	return textOut, thinkingOut
}

func mapFinishReasonToAnthropic(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "stop_sequence":
		return "stop_sequence"
	case "pause_turn":
		return "pause_turn"
	case "refusal":
		return "refusal"
	default:
		return "end_turn"
	}
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []AnthropicContent
}

type AnthropicContent struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type,omitempty"`
		Data      string `json:"data,omitempty"`
		Url       string `json:"url,omitempty"`
	} `json:"source,omitempty"`
	CacheControl *struct {
		Type string `json:"type"` // "ephemeral"
	} `json:"cache_control,omitempty"`
	// For tool_result blocks
	ToolUseId string `json:"tool_use_id,omitempty"`
}

func isAnthropicToolResultType(t string) bool {
	return t == "tool_result" || strings.HasSuffix(t, "_tool_result")
}

func isEmptyAnthropicUserContentBlocks(blocks []AnthropicContent) bool {
	if len(blocks) == 0 {
		return true
	}
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				return false
			}
		case "image":
			// image content is meaningful even without text
			if b.Source != nil && (strings.TrimSpace(b.Source.Url) != "" || strings.TrimSpace(b.Source.Data) != "") {
				return false
			}
		default:
			// Unknown content block types should be treated as meaningful to avoid dropping data silently.
			return false
		}
	}
	return true
}

// parseAnthropicContent handles both string and array content formats
func parseAnthropicContent(content any) []AnthropicContent {
	if content == nil {
		return nil
	}

	// If it's a string, convert to single text content block
	if str, ok := content.(string); ok {
		return []AnthropicContent{{Type: "text", Text: str}}
	}

	// If it's an array, parse each content block
	if arr, ok := content.([]any); ok {
		result := make([]AnthropicContent, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				c := AnthropicContent{Type: fmt.Sprintf("%v", m["type"])}
				if text, ok := m["text"].(string); ok {
					c.Text = text
				}
				// Anthropic tool_result uses `content`, not `text`.
				// Preserve textual content for conversion into OpenAI tool message.
				if isAnthropicToolResultType(c.Type) && c.Text == "" {
					switch v := m["content"].(type) {
					case string:
						c.Text = v
					case []any:
						var sb strings.Builder
						for _, item := range v {
							itemMap, ok := item.(map[string]any)
							if !ok {
								continue
							}
							if t, ok := itemMap["text"].(string); ok && t != "" {
								if sb.Len() > 0 {
									sb.WriteString("\n")
								}
								sb.WriteString(t)
							}
						}
						c.Text = sb.String()
					}
				}
				if toolUseId, ok := m["tool_use_id"].(string); ok {
					c.ToolUseId = toolUseId
				}
				if source, ok := m["source"].(map[string]any); ok {
					c.Source = &struct {
						Type      string `json:"type"`
						MediaType string `json:"media_type,omitempty"`
						Data      string `json:"data,omitempty"`
						Url       string `json:"url,omitempty"`
					}{
						Type: fmt.Sprintf("%v", source["type"]),
					}
					if mediatype, ok := source["media_type"].(string); ok {
						c.Source.MediaType = mediatype
					}
					if data, ok := source["data"].(string); ok {
						c.Source.Data = data
					}
					if url, ok := source["url"].(string); ok {
						c.Source.Url = url
					}
				}
				result = append(result, c)
			}
		}
		return result
	}

	return nil
}

// ConvertAnthropicToOpenAI converts Anthropic /v1/messages request to OpenAI /v1/chat/completions format
func ConvertAnthropicToOpenAI(req *AnthropicRequest) *model.GeneralOpenAIRequest {
	messages := make([]model.Message, 0, len(req.Messages)+1)

	// Handle system prompt - support both string and array formats
	if req.System != nil {
		switch s := req.System.(type) {
		case string:
			if s != "" {
				messages = append(messages, model.Message{
					Role:    "system",
					Content: s,
				})
			}
		case []any:
			// Array of {type: "text", text: "..."} or similar
			var sb strings.Builder
			for _, item := range s {
				if m, ok := item.(map[string]any); ok {
					if text, ok := m["text"].(string); ok {
						sb.WriteString(text)
						sb.WriteString("\n\n")
					}
				}
			}
			systemContent := strings.TrimSpace(sb.String())
			if systemContent != "" {
				messages = append(messages, model.Message{
					Role:    "system",
					Content: systemContent,
				})
			}
		}
	}

	for _, msg := range req.Messages {
		contentBlocks := parseAnthropicContent(msg.Content)
		// VSCode/SDK can emit empty user turns when user only presses Enter.
		// Do not forward empty user messages to upstream, otherwise the model may
		// repeatedly reason about "(no content)" and derail tool execution flow.
		if msg.Role == "user" && isEmptyAnthropicUserContentBlocks(contentBlocks) {
			continue
		}

		// Fast path: single text block stays as plain string content.
		if len(contentBlocks) == 1 && contentBlocks[0].Type == "text" && contentBlocks[0].ToolUseId == "" {
			messages = append(messages, model.Message{
				Role:    msg.Role,
				Content: contentBlocks[0].Text,
			})
			continue
		}

		// General path: build normal content parts and extract tool_result into role=tool messages.
		contentList := make([]any, 0, len(contentBlocks))
		toolMessages := make([]model.Message, 0)
		for _, c := range contentBlocks {
			if c.Type == "text" {
				contentList = append(contentList, map[string]string{"type": "text", "text": c.Text})
				continue
			}

			if isAnthropicToolResultType(c.Type) {
				if c.ToolUseId == "" {
					// Invalid tool_result block (missing tool_use_id) should not produce
					// malformed OpenAI tool messages; keep as plain user text fallback.
					if c.Text != "" {
						contentList = append(contentList, map[string]string{"type": "text", "text": c.Text})
					}
					continue
				}
				toolMessages = append(toolMessages, model.Message{
					Role:       "tool",
					ToolCallId: c.ToolUseId,
					Content:    c.Text,
				})
				continue
			}

			if c.Type == "image" && c.Source != nil {
				if c.Source.Url != "" {
					contentList = append(contentList, map[string]any{
						"type":      "image_url",
						"image_url": map[string]string{"url": c.Source.Url},
					})
				} else if c.Source.Data != "" {
					contentList = append(contentList, map[string]any{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", c.Source.MediaType, c.Source.Data),
						},
					})
				}
			}
		}

		// Keep the original user/assistant message when there is non-tool content.
		if len(contentList) > 0 {
			messages = append(messages, model.Message{
				Role:    msg.Role,
				Content: contentList,
			})
		}
		// Append tool_result as separate tool messages for OpenAI-compatible schema.
		messages = append(messages, toolMessages...)
	}

	openaiReq := &model.GeneralOpenAIRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Handle stream options for usage in stream
	if req.Stream {
		openaiReq.StreamOptions = &model.StreamOptions{IncludeUsage: true}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = convertAnthropicTools(req.Tools)
	}

	// Convert tool_choice
	if req.ToolChoice != nil {
		openaiReq.ToolChoice = convertAnthropicToolChoice(req.ToolChoice)
	} else if len(req.Tools) > 0 {
		// Anthropic default is auto: tools are available but the model may answer normally.
		openaiReq.ToolChoice = "auto"
	}

	// Preserve explicit thinking intent before compatibility policy is applied.
	if req.Thinking != nil {
		if openaiReq.ExtraFields == nil {
			openaiReq.ExtraFields = make(map[string]any)
		}
		thinkingType := strings.TrimSpace(strings.ToLower(req.Thinking.Type))
		thinkingEnabled := thinkingType == "" || thinkingType == "enabled" || thinkingType == "thinking"
		openaiReq.ExtraFields["enable_thinking"] = thinkingEnabled
		if req.Thinking.BudgetTokens > 0 {
			openaiReq.ExtraFields["thinking_budget_tokens"] = req.Thinking.BudgetTokens
		}
		thinkingPayload := map[string]any{
			"type":          "enabled",
			"budget_tokens": req.Thinking.BudgetTokens,
		}
		if req.Thinking.Display != "" {
			thinkingPayload["display"] = req.Thinking.Display
		}
		if !thinkingEnabled {
			thinkingPayload["type"] = "disabled"
		}
		openaiReq.ExtraFields["thinking"] = thinkingPayload
	}

	// Preserve any extra fields from original request
	if len(req.ExtraFields) > 0 {
		if openaiReq.ExtraFields == nil {
			openaiReq.ExtraFields = make(map[string]any)
		}
		for k, v := range req.ExtraFields {
			switch k {
			case "thinking_budget_tokens":
				// Keep converted value if present.
				continue
			case "reasoning", "reasoning_effort", "reasoning_content", "thinking_budget":
				continue
			case "chat_template_kwargs":
				if extraKwargs, ok := v.(map[string]any); ok {
					for kk, vv := range extraKwargs {
						openaiReq.ChatTemplateKwargs[kk] = vv
					}
				}
			default:
				openaiReq.ExtraFields[k] = v
			}
		}
	}

	applyAnthropicThinkingPolicy(openaiReq, req)

	return openaiReq
}

func allowAnthropicThinking(modelName string) bool {
	policy := strings.ToLower(strings.TrimSpace(config.AnthropicThinkingPolicy))
	switch policy {
	case "pass_thinking":
		return true
	case "whitelist":
		if modelName == "" {
			return false
		}
		target := strings.ToLower(strings.TrimSpace(modelName))
		for _, raw := range strings.Split(config.AnthropicThinkingWhitelist, ",") {
			item := strings.ToLower(strings.TrimSpace(raw))
			if item == "" {
				continue
			}
			if target == item {
				return true
			}
		}
		return false
	default:
		// strict_compat by default
		return false
	}
}

func applyAnthropicThinkingPolicy(openaiReq *model.GeneralOpenAIRequest, req *AnthropicRequest) {
	if openaiReq.ExtraFields == nil {
		openaiReq.ExtraFields = make(map[string]any)
	}
	if openaiReq.ChatTemplateKwargs == nil {
		openaiReq.ChatTemplateKwargs = make(map[string]any)
	}

	if allowAnthropicThinking(req.Model) {
		return
	}

	// strict compatibility guard:
	// many OpenAI-compatible upstreams may leak thought text or stall tool loop when thinking is on.
	openaiReq.ReasoningEffort = nil
	openaiReq.ChatTemplateKwargs["enable_thinking"] = false
	openaiReq.ExtraFields["enable_thinking"] = false
	openaiReq.ExtraFields["thinking"] = map[string]any{
		"type": "disabled",
	}
}

func convertAnthropicTools(anthropicTools []AnthropicTool) []model.Tool {
	tools := make([]model.Tool, 0, len(anthropicTools))
	for _, t := range anthropicTools {
		tools = append(tools, model.Tool{
			Type: "function",
			Function: model.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return tools
}

func convertAnthropicToolChoice(toolChoice any) any {
	switch tc := toolChoice.(type) {
	case string:
		if tc == "auto" {
			return "auto"
		} else if tc == "any" {
			return "required" // OpenAI uses "required" instead of "any"
		}
	case map[string]any:
		if toolType, ok := tc["type"].(string); ok && toolType == "tool" {
			if name, ok := tc["name"].(string); ok {
				return map[string]any{
					"type": "function",
					"function": map[string]any{
						"name": name,
					},
				}
			}
		}
	}
	return "auto"
}

// ConvertOpenAIResponseToAnthropic converts OpenAI /v1/chat/completions response to Anthropic format
// originModel: the model name sent to user (for HideUpstreamModel)
// hideUpstreamModel: if true, replace upstream model name with originModel
func ConvertOpenAIResponseToAnthropic(respBody []byte, isStream bool, originModel string, hideUpstreamModel bool) ([]byte, error) {
	if isStream {
		return convertOpenAIStreamToAnthropic(respBody, originModel, hideUpstreamModel)
	}
	return convertOpenAITextToAnthropic(respBody, originModel, hideUpstreamModel)
}

func convertOpenAITextToAnthropic(respBody []byte, originModel string, hideUpstreamModel bool) ([]byte, error) {
	var openaiResp struct {
		Id      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				Reasoning        string `json:"reasoning"`
				Thinking         string `json:"thinking"`
				RedactedThinking string `json:"redacted_thinking"`
				FunctionCall     *struct {
					Name      string `json:"name"`
					Arguments any    `json:"arguments"`
				} `json:"function_call"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments any    `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, err
	}

	stopReason := "end_turn"
	if len(openaiResp.Choices) > 0 {
		stopReason = mapFinishReasonToAnthropic(openaiResp.Choices[0].FinishReason)
	}

	var contentBlocks []map[string]any
	if len(openaiResp.Choices) > 0 {
		msg := openaiResp.Choices[0].Message
		thinkingText := msg.ReasoningContent
		if thinkingText == "" {
			thinkingText = msg.Reasoning
		}
		if thinkingText == "" {
			thinkingText = msg.Thinking
		}
		if thinkingText == "" {
			thinkingText = msg.RedactedThinking
		}
		cleanContent, taggedThinking := splitThinkBlocksFromText(msg.Content)
		if thinkingText == "" && taggedThinking != "" {
			thinkingText = taggedThinking
		}
		if thinkingText != "" {
			contentBlocks = append(contentBlocks, map[string]any{
				"type":      "thinking",
				"thinking":  thinkingText,
				"signature": "",
			})
		}
		if cleanContent != "" {
			contentBlocks = append(contentBlocks, map[string]any{
				"type": "text",
				"text": cleanContent,
			})
		}
		toolCalls := msg.ToolCalls
		if len(toolCalls) == 0 && msg.FunctionCall != nil && strings.TrimSpace(msg.FunctionCall.Name) != "" {
			toolCalls = append(toolCalls, struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments any    `json:"arguments"`
				} `json:"function"`
			}{
				ID:   "",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments any    `json:"arguments"`
				}{
					Name:      msg.FunctionCall.Name,
					Arguments: msg.FunctionCall.Arguments,
				},
			})
		}

		for _, tc := range toolCalls {
			var toolInput any
			argsText := stringifyToolArguments(tc.Function.Arguments)
			if argsText != "" {
				_ = json.Unmarshal([]byte(argsText), &toolInput)
			}
			toolUseId := tc.ID
			if toolUseId == "" {
				toolUseId = fmt.Sprintf("toolu_%d", len(contentBlocks))
			}
			toolUseType := "tool_use"
			if strings.TrimSpace(tc.Type) == "server_tool_use" {
				toolUseType = "server_tool_use"
			}
			contentBlocks = append(contentBlocks, map[string]any{
				"type":  toolUseType,
				"id":    toolUseId,
				"name":  tc.Function.Name,
				"input": toolInput,
			})
		}
	}

	if len(contentBlocks) == 0 {
		contentBlocks = append(contentBlocks, map[string]any{
			"type": "text",
			"text": "",
		})
	}
	// Guard: tool_use stop reason must have at least one tool_use content block.
	if stopReason == "tool_use" {
		hasToolUse := false
		for _, b := range contentBlocks {
			if t, ok := b["type"].(string); ok && (t == "tool_use" || t == "server_tool_use") {
				hasToolUse = true
				break
			}
		}
		if !hasToolUse {
			stopReason = "end_turn"
		}
	}

	anthropicResp := map[string]any{
		"id":          openaiResp.Id,
		"type":        "message",
		"role":        "assistant",
		"content":     contentBlocks,
		"model":       originModel,
		"stop_reason": stopReason,
		"usage": map[string]int{
			"input_tokens":  openaiResp.Usage.PromptTokens,
			"output_tokens": openaiResp.Usage.CompletionTokens,
		},
	}
	if !hideUpstreamModel {
		anthropicResp["model"] = openaiResp.Model
	}

	return json.Marshal(anthropicResp)
}

func convertOpenAIStreamToAnthropic(respBody []byte, originModel string, hideUpstreamModel bool) ([]byte, error) {
	lines := strings.Split(string(respBody), "\n")
	var anthropicLines []string
	var messageId, modelName string
	var stopReasonStr string
	var inputTokens, outputTokens int
	messageStarted := false
	thinkingIndex := -1
	redactedThinkingIndex := -1
	textIndex := -1
	nextIndex := 0
	seenToolKeys := make([]string, 0)
	seenToolKeySet := make(map[string]struct{})
	toolIDByKey := make(map[string]string)
	toolTypeByKey := make(map[string]string)
	toolNameByKey := make(map[string]string)
	pendingToolArgsByKey := make(map[string]string)
	toolKeyByIndex := make(map[int]string)
	emittedToolUse := false
	var collectedText strings.Builder
	var collectedThinking strings.Builder
	legacyFunctionCallName := ""
	inThinkTag := false
	thinkTagCarry := ""

	appendMessageStart := func() {
		if messageStarted {
			return
		}
		messageStarted = true
		messageStart := map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":            messageId,
				"type":          "message",
				"role":          "assistant",
				"content":       []any{},
				"model":         modelName,
				"stop_reason":   nil,
				"stop_sequence": nil,
				"usage": map[string]int{
					"input_tokens":  inputTokens,
					"output_tokens": 0,
				},
			},
		}
		startData, _ := json.Marshal(messageStart)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: message_start\ndata: %s", string(startData)))
	}

	startTextBlock := func() int {
		if textIndex >= 0 {
			return textIndex
		}
		textIndex = nextIndex
		nextIndex++
		textBlockStart := map[string]any{
			"type":  "content_block_start",
			"index": textIndex,
			"content_block": map[string]any{
				"type": "text",
				"text": "",
			},
		}
		textBlockStartData, _ := json.Marshal(textBlockStart)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(textBlockStartData)))
		return textIndex
	}

	startThinkingBlock := func() int {
		if thinkingIndex >= 0 {
			return thinkingIndex
		}
		thinkingIndex = nextIndex
		nextIndex++
		thinkingBlockStart := map[string]any{
			"type":  "content_block_start",
			"index": thinkingIndex,
			"content_block": map[string]any{
				"type":      "thinking",
				"thinking":  "",
				"signature": "",
			},
		}
		thinkingBlockStartData, _ := json.Marshal(thinkingBlockStart)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(thinkingBlockStartData)))
		return thinkingIndex
	}

	startRedactedThinkingBlock := func() int {
		if redactedThinkingIndex >= 0 {
			return redactedThinkingIndex
		}
		redactedThinkingIndex = nextIndex
		nextIndex++
		blockStart := map[string]any{
			"type":  "content_block_start",
			"index": redactedThinkingIndex,
			"content_block": map[string]any{
				"type": "redacted_thinking",
				"data": "",
			},
		}
		blockStartData, _ := json.Marshal(blockStart)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(blockStartData)))
		return redactedThinkingIndex
	}

	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var chunk struct {
			Id      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
					Thinking         string `json:"thinking"`
					RedactedThinking string `json:"redacted_thinking"`
					Signature        string `json:"signature"`
					Citation         any    `json:"citation"`
					FunctionCall     *struct {
						Name      string `json:"name"`
						Arguments any    `json:"arguments"`
					} `json:"function_call"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						Type     string `json:"type"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments any    `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if messageId == "" && chunk.Id != "" {
			messageId = chunk.Id
		}
		if modelName == "" {
			if hideUpstreamModel {
				modelName = originModel
			} else {
				modelName = chunk.Model
			}
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		content, taggedThinking := splitThinkTaggedChunk(delta.Content, &inThinkTag, &thinkTagCarry)
		reasoningContent := delta.ReasoningContent
		redactedThinking := delta.RedactedThinking
		if reasoningContent == "" {
			reasoningContent = delta.Reasoning
		}
		if reasoningContent == "" {
			reasoningContent = delta.Thinking
		}
		if reasoningContent == "" && taggedThinking != "" {
			reasoningContent = taggedThinking
		}
		finishReason := chunk.Choices[0].FinishReason

		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}

		if finishReason != "" && finishReason != "null" {
			stopReasonStr = mapFinishReasonToAnthropic(finishReason)
		}

		if reasoningContent != "" {
			collectedThinking.WriteString(reasoningContent)
			appendMessageStart()
			idx := startThinkingBlock()
			thinkingDelta := map[string]any{
				"type":  "content_block_delta",
				"index": idx,
				"delta": map[string]string{
					"type":     "thinking_delta",
					"thinking": reasoningContent,
				},
			}
			thinkingDeltaData, _ := json.Marshal(thinkingDelta)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(thinkingDeltaData)))
			if delta.Signature != "" {
				signatureDelta := map[string]any{
					"type":  "content_block_delta",
					"index": idx,
					"delta": map[string]string{
						"type":      "signature_delta",
						"signature": delta.Signature,
					},
				}
				signatureDeltaData, _ := json.Marshal(signatureDelta)
				anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(signatureDeltaData)))
			}
		}
		if redactedThinking != "" {
			collectedThinking.WriteString(redactedThinking)
			appendMessageStart()
			idx := startRedactedThinkingBlock()
			redactedDelta := map[string]any{
				"type":  "content_block_delta",
				"index": idx,
				"delta": map[string]string{
					"type": "text_delta",
					"text": redactedThinking,
				},
			}
			redactedDeltaData, _ := json.Marshal(redactedDelta)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(redactedDeltaData)))
		}

		streamToolCalls := delta.ToolCalls
		if len(streamToolCalls) == 0 && delta.FunctionCall != nil {
			if strings.TrimSpace(delta.FunctionCall.Name) != "" {
				legacyFunctionCallName = delta.FunctionCall.Name
			}
			streamToolCalls = append(streamToolCalls, struct {
				Index    int    `json:"index"`
				Type     string `json:"type"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments any    `json:"arguments"`
				} `json:"function"`
			}{
				Index: 0,
				Type:  "function",
				ID:    "",
				Function: struct {
					Name      string `json:"name"`
					Arguments any    `json:"arguments"`
				}{
					Name:      legacyFunctionCallName,
					Arguments: delta.FunctionCall.Arguments,
				},
			})
		}
		// When tool calls are present, ignore whitespace-only text deltas.
		// Some clients may treat a leading blank text block as a normal completion path.
		if content != "" && !(len(streamToolCalls) > 0 && strings.TrimSpace(content) == "") {
			collectedText.WriteString(content)
			appendMessageStart()
			textBlockIndex := startTextBlock()
			contentDelta := map[string]any{
				"type":  "content_block_delta",
				"index": textBlockIndex,
				"delta": map[string]string{
					"type": "text_delta",
					"text": content,
				},
			}
			deltaData, _ := json.Marshal(contentDelta)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(deltaData)))
		}
		if delta.Citation != nil {
			appendMessageStart()
			textBlockIndex := startTextBlock()
			citationDelta := map[string]any{
				"type":  "content_block_delta",
				"index": textBlockIndex,
				"delta": map[string]any{
					"type":     "citations_delta",
					"citation": delta.Citation,
				},
			}
			citationDeltaData, _ := json.Marshal(citationDelta)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(citationDeltaData)))
		}

		for _, tc := range streamToolCalls {
			appendMessageStart()
			key := tc.ID
			if key != "" {
				toolKeyByIndex[tc.Index] = key
			} else if existingKey, ok := toolKeyByIndex[tc.Index]; ok {
				key = existingKey
			} else {
				key = fmt.Sprintf("tool_%d", tc.Index)
				toolKeyByIndex[tc.Index] = key
			}
			if _, ok := seenToolKeySet[key]; !ok {
				seenToolKeySet[key] = struct{}{}
				seenToolKeys = append(seenToolKeys, key)
			}
			if tc.Function.Name != "" {
				toolNameByKey[key] = tc.Function.Name
			}
			if tc.ID != "" {
				toolIDByKey[key] = tc.ID
			}
			if strings.TrimSpace(tc.Type) != "" {
				toolTypeByKey[key] = strings.TrimSpace(tc.Type)
			}
			pendingToolArgsByKey[key] = mergeToolArguments(pendingToolArgsByKey[key], tc.Function.Arguments)
		}
	}

	if stopReasonStr == "" {
		stopReasonStr = "end_turn"
	}

	appendMessageStart()
	if thinkingIndex >= 0 {
		thinkingBlockStop := map[string]any{"type": "content_block_stop", "index": thinkingIndex}
		thinkingStopData, _ := json.Marshal(thinkingBlockStop)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_stop\ndata: %s", string(thinkingStopData)))
	}
	if redactedThinkingIndex >= 0 {
		redactedBlockStop := map[string]any{"type": "content_block_stop", "index": redactedThinkingIndex}
		redactedStopData, _ := json.Marshal(redactedBlockStop)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_stop\ndata: %s", string(redactedStopData)))
	}
	if textIndex >= 0 {
		textBlockStop := map[string]any{"type": "content_block_stop", "index": textIndex}
		textStopData, _ := json.Marshal(textBlockStop)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_stop\ndata: %s", string(textStopData)))
	}
	for _, key := range seenToolKeys {
		name := strings.TrimSpace(toolNameByKey[key])
		if name == "" {
			continue
		}
		argsJSON := strings.TrimSpace(pendingToolArgsByKey[key])
		_, ok := parseToolInputJSON(argsJSON)
		if !ok {
			continue
		}
		toolType := "tool_use"
		if toolTypeByKey[key] == "server_tool_use" {
			toolType = "server_tool_use"
		}
		toolID := strings.TrimSpace(toolIDByKey[key])
		if toolID == "" {
			toolID = fmt.Sprintf("toolu_%s", key)
		}
		idx := nextIndex
		nextIndex++
		toolBlockStart := map[string]any{
			"type":  "content_block_start",
			"index": idx,
			"content_block": map[string]any{
				"type":  toolType,
				"id":    toolID,
				"name":  name,
				"input": map[string]any{},
			},
		}
		toolBlockData, _ := json.Marshal(toolBlockStart)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(toolBlockData)))
		toolInputDelta := map[string]any{
			"type":  "content_block_delta",
			"index": idx,
			"delta": map[string]string{
				"type":         "input_json_delta",
				"partial_json": argsJSON,
			},
		}
		toolInputDeltaData, _ := json.Marshal(toolInputDelta)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(toolInputDeltaData)))
		emittedToolUse = true
		toolBlockStop := map[string]any{"type": "content_block_stop", "index": idx}
		toolStopData, _ := json.Marshal(toolBlockStop)
		anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_stop\ndata: %s", string(toolStopData)))
	}
	// Fallback: some upstreams put tool call payload into thinking/text (e.g. <tool_call>{...}</tool_call>)
	// instead of structured tool_calls in stream. Recover a synthetic tool_use block.
	if !emittedToolUse {
		if toolName, toolInput, ok := tryExtractToolUseFromText(collectedThinking.String() + "\n" + collectedText.String()); ok {
			argsBytes, err := json.Marshal(toolInput)
			if err != nil {
				argsBytes = []byte("{}")
			}
			idx := nextIndex
			nextIndex++
			toolBlockStart := map[string]any{
				"type":  "content_block_start",
				"index": idx,
				"content_block": map[string]any{
					"type":  "tool_use",
					"id":    fmt.Sprintf("toolu_fallback_%d", idx),
					"name":  toolName,
					"input": map[string]any{},
				},
			}
			toolBlockData, _ := json.Marshal(toolBlockStart)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(toolBlockData)))
			toolInputDelta := map[string]any{
				"type":  "content_block_delta",
				"index": idx,
				"delta": map[string]string{
					"type":         "input_json_delta",
					"partial_json": string(argsBytes),
				},
			}
			toolInputDeltaData, _ := json.Marshal(toolInputDelta)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(toolInputDeltaData)))
			toolBlockStop := map[string]any{"type": "content_block_stop", "index": idx}
			toolStopData, _ := json.Marshal(toolBlockStop)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_stop\ndata: %s", string(toolStopData)))
			emittedToolUse = true
		}
	}
	// If upstream says tool_use but no valid tool_use block was emitted,
	// fallback to end_turn to avoid Anthropic SDK parse failure.
	if stopReasonStr == "tool_use" && !emittedToolUse {
		stopReasonStr = "end_turn"
	}
	if emittedToolUse {
		stopReasonStr = "tool_use"
	}

	messageDelta := map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReasonStr,
			"stop_sequence": nil,
		},
		"usage": map[string]int{"output_tokens": outputTokens},
	}
	deltaData, _ := json.Marshal(messageDelta)
	anthropicLines = append(anthropicLines, fmt.Sprintf("event: message_delta\ndata: %s", string(deltaData)))

	messageStop := map[string]any{"type": "message_stop"}
	stopEventData, _ := json.Marshal(messageStop)
	anthropicLines = append(anthropicLines, fmt.Sprintf("event: message_stop\ndata: %s", string(stopEventData)))

	result := strings.Join(anthropicLines, "\n\n")
	if result != "" {
		result += "\n\n"
	}
	return []byte(result), nil
}

// https://platform.openai.com/docs/api-reference/chat

func relayHelper(c *gin.Context, relayMode int) *model.ErrorWithStatusCode {
	var err *model.ErrorWithStatusCode
	switch relayMode {
	case relaymode.ImagesGenerations:
		err = controller.RelayImageHelper(c, relayMode)
	case relaymode.AudioSpeech:
		fallthrough
	case relaymode.AudioTranslation:
		fallthrough
	case relaymode.AudioTranscription:
		err = controller.RelayAudioHelper(c, relayMode)
	case relaymode.AudioPredict:
		err = controller.RelayAudioPredictHelper(c)
	case relaymode.FileParse:
		err = controller.RelayFileParseSubmitHelper(c)
	case relaymode.FileParseTask:
		err = controller.RelayFileParseTaskHelper(c, false)
	case relaymode.FileParseTaskResult:
		err = controller.RelayFileParseTaskHelper(c, true)
	case relaymode.Proxy:
		err = controller.RelayProxyHelper(c, relayMode)
	case relaymode.Rerank:
		err = controller.RelayRerankHelper(c)
	default:
		err = controller.RelayTextHelper(c)
	}
	return err
}

func Relay(c *gin.Context) {
	ctx := c.Request.Context()
	relayMode := relaymode.GetByPath(c.Request.URL.Path)
	if config.DebugEnabled {
		requestBody, _ := common.GetRequestBody(c)
		logger.Debugf(ctx, "request body: %s", string(requestBody))
	}
	channelId := c.GetInt(ctxkey.ChannelId)
	userId := c.GetInt(ctxkey.Id)
	bizErr := relayHelper(c, relayMode)
	if bizErr == nil {
		monitor.Emit(channelId, true)
		return
	}
	lastFailedChannelId := channelId
	channelName := c.GetString(ctxkey.ChannelName)
	originalModel := c.GetString(ctxkey.OriginalModel)
	go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)
	requestId := c.GetString(helper.RequestIdKey)
	retryTimes := config.RetryTimes
	if !shouldRetry(c, bizErr.StatusCode) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		retryTimes = 0
	}
	for i := retryTimes; i > 0; i-- {
		channel, err := dbmodel.CacheGetRandomSatisfiedChannelByModel(originalModel, i != retryTimes)
		if err != nil {
			logger.Errorf(ctx, "CacheGetRandomSatisfiedChannelByModel failed: %+v", err)
			break
		}
		logger.Infof(ctx, "using channel #%d to retry (remain times %d)", channel.Id, i)
		if channel.Id == lastFailedChannelId {
			continue
		}
		middleware.SetupContextForSelectedChannel(c, channel, originalModel)
		requestBody, err := common.GetRequestBody(c)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		bizErr = relayHelper(c, relayMode)
		if bizErr == nil {
			return
		}
		channelId := c.GetInt(ctxkey.ChannelId)
		lastFailedChannelId = channelId
		channelName := c.GetString(ctxkey.ChannelName)
		go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)
	}
	if bizErr != nil {
		errorMessage := bizErr.Error.Message
		if bizErr.StatusCode == http.StatusTooManyRequests {
			errorMessage = "当前分组上游负载已饱和，请稍后再试"
		}

		// Use value copy to avoid race condition
		errorMessage = helper.MessageWithRequestId(errorMessage, requestId)
		c.JSON(bizErr.StatusCode, gin.H{
			"error": model.Error{
				Message: errorMessage,
				Type:    bizErr.Error.Type,
				Param:   bizErr.Error.Param,
				Code:    bizErr.Error.Code,
			},
		})
	}
}

func shouldRetry(c *gin.Context, statusCode int) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode/100 == 5 {
		return true
	}
	if statusCode == http.StatusBadRequest {
		return false
	}
	if statusCode/100 == 2 {
		return false
	}
	return true
}

func processChannelRelayError(ctx context.Context, userId int, channelId int, channelName string, err model.ErrorWithStatusCode) {
	logger.Errorf(ctx, "relay error (channel id %d, user id: %d): %s", channelId, userId, err.Message)
	// https://platform.openai.com/docs/guides/error-codes/api-errors
	if monitor.ShouldDisableChannel(&err.Error, err.StatusCode) {
		monitor.DisableChannel(channelId, channelName, err.Message)
	} else {
		monitor.Emit(channelId, false)
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := model.Error{
		Message: "API not implemented",
		Type:    "one_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := model.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}

// RelayResponses handles OpenAI Responses API /v1/responses requests
// POST /v1/responses - create a response (streaming or non-streaming)
// GET/DELETE /v1/responses/:id - routed to RelayNotImplemented
func RelayResponses(c *gin.Context) {
	if !config.ResponsesAPIEnabled {
		err := model.Error{
			Message: "Responses API is not enabled. Set RESPONSES_API_ENABLED=true to enable.",
			Type:    "invalid_request_error",
			Param:   "",
			Code:    "responses_api_disabled",
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}

	method := c.Request.Method
	switch method {
	case "POST":
		// Delegate to relay controller
		bizErr := controller.RelayResponsesHelper(c)
		if bizErr != nil {
			c.JSON(bizErr.StatusCode, gin.H{"error": bizErr.Error})
			return
		}
	default:
		err := model.Error{
			Message: fmt.Sprintf("Method %s not allowed for /v1/responses", method),
			Type:    "invalid_request_error",
			Param:   "",
			Code:    "method_not_allowed",
		}
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": err})
		return
	}
}

// RelayAnthropicPassthrough handles Anthropic API /v1/messages requests in passthrough mode
// This is used for vLLM backends that natively support Anthropic API format
func RelayAnthropicPassthrough(c *gin.Context) {
	ctx := c.Request.Context()
	requestId := c.GetString(helper.RequestIdKey)

	// Get channel info from context (set by Distribute middleware)
	channelId := c.GetInt(ctxkey.ChannelId)
	channelName := c.GetString(ctxkey.ChannelName)
	baseURL := c.GetString(ctxkey.BaseURL)

	// Get user and token info from context (set by TokenAuth middleware)
	userId := c.GetInt(ctxkey.Id)
	tokenId := c.GetInt(ctxkey.TokenId)
	userGroup := c.GetString(ctxkey.Group)
	tokenName := c.GetString(ctxkey.TokenName)
	startTime := time.Now()

	// Pre-consume quota check (same logic as RelayTextHelper)
	// Parse request to get model name and estimate prompt tokens
	var anthropicReq AnthropicRequest
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Errorf(ctx, "failed to read request body: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": "Failed to read request body",
			},
		})
		return
	}
	if err := json.Unmarshal(requestBody, &anthropicReq); err != nil {
		logger.Errorf(ctx, "failed to parse anthropic request: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "invalid_request",
				"message": "Failed to parse request body",
			},
		})
		return
	}
	// Restore request body for later use
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	// Convert Anthropic request to OpenAI format for token counting / quota logic
	openaiReq := ConvertAnthropicToOpenAI(&anthropicReq)

	// Apply model mapping before pricing/quota to keep behavior aligned with RelayTextHelper
	if modelMapping := c.GetStringMapString(ctxkey.ModelMapping); modelMapping != nil {
		if mappedModel, ok := modelMapping[openaiReq.Model]; ok && mappedModel != "" {
			logger.Debugf(ctx, "model mapping (ctx): %s -> %s", openaiReq.Model, mappedModel)
			openaiReq.Model = mappedModel
		}
	}

	relayMode := relaymode.GetByPath(c.Request.URL.Path)
	promptTokens := controller.GetPromptTokens(openaiReq, relayMode)

	// Get model info for price calculation
	modelInfo, _ := dbmodel.GetModelById(openaiReq.Model)
	inputPrice := 0.0
	outputPrice := 0.0
	if modelInfo != nil {
		inputPrice = modelInfo.InputPrice
		outputPrice = modelInfo.OutputPrice
	}
	groupRatio := billingratio.GetGroupRatio(userGroup)

	// Build meta for quota pre-consumption
	meta := &relaymeta.Meta{
		UserId:          userId,
		TokenId:         tokenId,
		Group:           userGroup,
		ChannelId:       channelId,
		TokenName:       tokenName,
		OriginModelName: anthropicReq.Model,
		ActualModelName: openaiReq.Model,
		StartTime:       startTime,
		IsStream:        anthropicReq.Stream,
	}

	// Pre-consume quota using exact calculation
	preConsumedQuota, bizErr := controller.PreConsumeQuota(ctx, openaiReq, promptTokens, inputPrice, groupRatio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "preConsumeQuota failed: %+v", *bizErr)
		c.JSON(bizErr.StatusCode, gin.H{
			"error": gin.H{
				"type":    bizErr.Error.Type,
				"message": bizErr.Error.Message,
			},
		})
		return
	}

	// Store preConsumedQuota for observability/debug
	c.Set("pre_consumed_quota", preConsumedQuota)
	quotaSettled := false
	rollbackPreConsume := func() {
		if quotaSettled {
			return
		}
		relaybilling.ReturnPreConsumedQuota(ctx, preConsumedQuota, tokenId)
		quotaSettled = true
	}

	if config.DebugEnabled {
		logger.Debugf(ctx, "anthropic passthrough request body: %s", string(requestBody))
	}

	// Build target URL and determine if upstream is OpenAI-compatible
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	// Detect if upstream is Anthropic native or OpenAI-compatible
	isAnthropicUpstream := strings.Contains(baseURL, "anthropic.com") ||
		strings.Contains(baseURL, "api.anthropic")

	var targetURL string
	var isStream bool

	if isAnthropicUpstream {
		// Native Anthropic - passthrough as-is
		trimmedBaseURL := strings.TrimSuffix(baseURL, "/")
		if strings.HasSuffix(trimmedBaseURL, "/v1") {
			targetURL = fmt.Sprintf("%s/messages", trimmedBaseURL)
		} else {
			targetURL = fmt.Sprintf("%s/v1/messages", trimmedBaseURL)
		}
	} else {
		// OpenAI-compatible upstream - convert Anthropic format to OpenAI format
		trimmedBaseURL := strings.TrimSuffix(baseURL, "/")
		if strings.HasSuffix(trimmedBaseURL, "/v1") {
			targetURL = fmt.Sprintf("%s/chat/completions", trimmedBaseURL)
		} else {
			targetURL = fmt.Sprintf("%s/v1/chat/completions", trimmedBaseURL)
		}

		// Parse Anthropic request
		var anthropicReq AnthropicRequest
		if err := json.Unmarshal(requestBody, &anthropicReq); err != nil {
			logger.Errorf(ctx, "failed to parse anthropic request: %s", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"type":    "invalid_request",
					"message": "Failed to parse request body",
				},
			})
			return
		}
		isStream = anthropicReq.Stream

		requestBody, err = json.Marshal(openaiReq)
		if err != nil {
			logger.Errorf(ctx, "failed to marshal openai request: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"type":    "api_error",
					"message": "Failed to convert request",
				},
			})
			return
		}

		if config.DebugEnabled {
			logger.Debugf(ctx, "converted to openai request: %s", string(requestBody))
		}
		logger.Infof(ctx, "anthropic->openai route resolved: channel=%d base=%s target=%s model=%s", channelId, baseURL, targetURL, openaiReq.Model)
	}

	// Get API key
	apiToken := strings.TrimSpace(c.Request.Header.Get("Authorization"))
	if apiToken == "" {
		rollbackPreConsume()
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"type":    "authentication_error",
				"message": "Missing API key",
			},
		})
		return
	}
	apiToken = strings.TrimPrefix(apiToken, "Bearer ")

	// Create request to backend
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(requestBody))
	if err != nil {
		rollbackPreConsume()
		logger.Errorf(ctx, "failed to create request: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": "Failed to create request",
			},
		})
		return
	}

	// Set headers based on upstream type
	if isAnthropicUpstream {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiToken)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("anthropic-beta", "messages-2023-12-15")
		// Forward additional headers from client
		for _, h := range []string{"anthropic-version", "anthropic-beta"} {
			if v := c.Request.Header.Get(h); v != "" {
				req.Header.Set(h, v)
			}
		}
	} else {
		// OpenAI-compatible headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))
		// Forward stream options
		if isStream {
			req.Header.Set("X-Request-Type", "stream")
		}
	}

	// Send request
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		rollbackPreConsume()
		logger.Errorf(ctx, "failed to send request to backend: %s", err.Error())
		monitor.RecordChannelFailure(channelId, channelName, err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": helper.MessageWithRequestId("Failed to connect to backend", requestId),
			},
		})
		return
	}
	defer resp.Body.Close()

	// Record success
	monitor.RecordChannelSuccess(channelId)

	// Copy response headers with gateway-safe filtering.
	// We rewrite body in some branches, so hop-by-hop and length headers must be regenerated.
	copySafeResponseHeaders(c, resp.Header)

	// Track response content-type for final writeback.
	responseContentType := resp.Header.Get("Content-Type")
	if responseContentType == "" {
		responseContentType = "application/json"
	}

	// Copy response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		rollbackPreConsume()
		logger.Errorf(ctx, "failed to read response body: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": "Failed to read response",
			},
		})
		return
	}

	if config.DebugEnabled {
		logger.Debugf(ctx, "anthropic passthrough response status: %d, body: %s", resp.StatusCode, string(respBody))
	}
	if resp.StatusCode == http.StatusUnauthorized && len(respBody) == 0 {
		logger.Errorf(ctx, "anthropic passthrough got upstream 401 with empty body (target=%s, anthropic_upstream=%v)", targetURL, isAnthropicUpstream)
	}
	if resp.StatusCode != http.StatusOK {
		rollbackPreConsume()
	}
	if resp.StatusCode == http.StatusOK {
		usage := extractUsageFromPassthroughResponse(respBody, isStream)
		if usage == nil {
			// Stream mode may miss final usage on some upstreams. Fall back to prompt-only settlement.
			logger.Warnf(ctx, "anthropic passthrough missing usage, fallback settlement with prompt tokens only: stream=%v upstream_anthropic=%v", isStream, isAnthropicUpstream)
			usage = &model.Usage{
				PromptTokens:     promptTokens,
				CompletionTokens: 0,
				TotalTokens:      promptTokens,
			}
		}
		quotaSettled = true
		go controller.PostConsumeQuota(ctx, usage, meta, openaiReq, inputPrice, outputPrice, groupRatio, preConsumedQuota, false)
	}

	// If upstream was OpenAI-compatible, convert response back to Anthropic format
	if !isAnthropicUpstream && resp.StatusCode == 200 {
		// Get hide upstream model config from channel
		hideUpstreamModel := false
		originModel := c.GetString(ctxkey.RequestModel)
		if cfgRaw, ok := c.Get(ctxkey.Config); ok {
			cfg, _ := cfgRaw.(dbmodel.ChannelConfig)
			hideUpstreamModel = cfg.HideUpstreamModel
		}
		convertedBody, err := ConvertOpenAIResponseToAnthropic(respBody, isStream, originModel, hideUpstreamModel)
		if err != nil {
			logger.Errorf(ctx, "failed to convert response: %s", err.Error())
			c.Data(resp.StatusCode, responseContentType, respBody)
			return
		}
		respBody = convertedBody
		// Set content type based on stream mode
		if isStream {
			responseContentType = "text/event-stream; charset=utf-8"
		} else {
			responseContentType = "application/json"
		}
	}

	// Return response
	c.Writer.Header().Del("Content-Length")
	c.Writer.Header().Del("Transfer-Encoding")
	c.Writer.Header().Del("Connection")
	c.Writer.Header().Set("Content-Type", responseContentType)
	c.Data(resp.StatusCode, responseContentType, respBody)
}

func extractUsageFromPassthroughResponse(respBody []byte, isStream bool) *model.Usage {
	if isStream || len(respBody) == 0 {
		return nil
	}

	// OpenAI-compatible response usage
	var openaiResp struct {
		Usage *model.Usage `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &openaiResp); err == nil && openaiResp.Usage != nil {
		return openaiResp.Usage
	}

	// Anthropic-native response usage
	var anthropicResp struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil
	}
	if anthropicResp.Usage.InputTokens == 0 && anthropicResp.Usage.OutputTokens == 0 {
		return nil
	}
	return &model.Usage{
		PromptTokens:     anthropicResp.Usage.InputTokens,
		CompletionTokens: anthropicResp.Usage.OutputTokens,
		TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
	}
}

// CountTokensAnthropic handles Anthropic /v1/messages/count_tokens requests
// Reference: https://docs.anthropic.com/en/api/messages#count-tokens
// This endpoint validates auth and model permissions but does NOT deduct quota
func CountTokensAnthropic(c *gin.Context) {
	ctx := c.Request.Context()
	requestId := c.GetString(helper.RequestIdKey)

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": helper.MessageWithRequestId("Failed to read request body", requestId),
			},
		})
		return
	}

	// Parse request - use map to handle flexible body structure
	var reqBody map[string]any
	if err := json.Unmarshal(requestBody, &reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": helper.MessageWithRequestId(fmt.Sprintf("Failed to parse request: %s", err.Error()), requestId),
			},
		})
		return
	}

	// Get model from request
	modelName, _ := reqBody["model"].(string)

	// Validate model permission (TokenAuth middleware already set available models in context)
	availableModels := c.GetString(ctxkey.AvailableModels)
	if availableModels != "" && modelName != "" {
		// Check if model is in allowed list
		allowedModels := strings.Split(availableModels, ",")
		found := false
		for _, m := range allowedModels {
			if strings.TrimSpace(m) == modelName {
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"type":    "forbidden",
					"message": helper.MessageWithRequestId(fmt.Sprintf("Token not allowed to use model: %s", modelName), requestId),
				},
			})
			return
		}
	}

	// Get user info for logging
	userId := c.GetInt(ctxkey.Id)
	tokenName := c.GetString(ctxkey.TokenName)

	// Calculate input tokens (approximate counting)
	inputTokens := countAnthropicInputTokens(requestBody)

	// Log the request (but no quota consumption)
	logger.Infof(ctx, "count_tokens: model=%s tokens=%d user=%d token=%s",
		modelName, inputTokens, userId, tokenName)

	// Return response - Anthropic format
	c.JSON(http.StatusOK, gin.H{
		"input_tokens": inputTokens,
	})
}

// countAnthropicInputTokens calculates approximate token count for Anthropic request
func countAnthropicInputTokens(body []byte) int {
	// Simple approximation: count tokens by characters / 4
	// For accurate counting, would need proper tokenizer
	// Claude uses ~4 chars per token on average
	text := string(body)
	// Remove JSON structure overhead for more accurate count
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return len(text) / 4
	}

	// Count characters in key fields
	tokenCount := 0

	// Count model name
	if model, ok := raw["model"].(string); ok {
		tokenCount += len(model) / 4
	}

	// Count system prompt
	if system, ok := raw["system"]; ok {
		switch s := system.(type) {
		case string:
			tokenCount += len(s) / 4
		case []any:
			for _, item := range s {
				if m, ok := item.(map[string]any); ok {
					if text, ok := m["text"].(string); ok {
						tokenCount += len(text) / 4
					}
				}
			}
		}
	}

	// Count messages
	if messages, ok := raw["messages"].([]any); ok {
		for _, msg := range messages {
			if m, ok := msg.(map[string]any); ok {
				// Count role
				if role, ok := m["role"].(string); ok {
					tokenCount += len(role) / 4
				}
				// Count content
				if content, ok := m["content"]; ok {
					switch c := content.(type) {
					case string:
						tokenCount += len(c) / 4
					case []any:
						for _, item := range c {
							if itemMap, ok := item.(map[string]any); ok {
								if text, ok := itemMap["text"].(string); ok {
									tokenCount += len(text) / 4
								}
							}
						}
					}
				}
			}
		}
	}

	// Add overhead for JSON structure (~10 tokens per message + system)
	numMessages := 0
	if messages, ok := raw["messages"].([]any); ok {
		numMessages = len(messages)
	}
	tokenCount += numMessages * 10
	if _, ok := raw["system"]; ok {
		tokenCount += 10
	}

	// Minimum 1 token
	if tokenCount < 1 {
		tokenCount = 1
	}

	return tokenCount
}
