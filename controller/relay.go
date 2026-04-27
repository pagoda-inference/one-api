package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	"github.com/pagoda-inference/one-api/relay/controller"
	"github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
)

// AnthropicRequest is the incoming Anthropic /v1/messages format
type AnthropicRequest struct {
	Model          string                   `json:"model"`
	Messages      []AnthropicMessage       `json:"messages"`
	System        any                       `json:"system,omitempty"`           // string or []SystemBlock
	MaxTokens     int                      `json:"max_tokens"`
	Stream        bool                     `json:"stream,omitempty"`
	StreamOptions *AnthropicStreamOptions  `json:"stream_options,omitempty"`
	Temperature   *float64                 `json:"temperature,omitempty"`
	TopP          *float64                 `json:"top_p,omitempty"`
	TopK          *int                     `json:"top_k,omitempty"`
	Tools         []AnthropicTool         `json:"tools,omitempty"`
	ToolChoice    any                       `json:"tool_choice,omitempty"`     // string or ToolChoiceBlock
	Metadata      map[string]any           `json:"metadata,omitempty"`
	StopSequences []string                `json:"stop_sequences,omitempty"`
	Thinking      *AnthropicThinking       `json:"thinking,omitempty"`
	Betas         []string                  `json:"betas,omitempty"`
	ExtraFields   map[string]any           `json:"-"`
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

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []AnthropicContent
}

type AnthropicContent struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	Source       *struct {
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
		openaiMsg := model.Message{
			Role: msg.Role,
		}

		// Convert content blocks - handle both string and array formats
		contentBlocks := parseAnthropicContent(msg.Content)
		if len(contentBlocks) == 1 && contentBlocks[0].Type == "text" && contentBlocks[0].ToolUseId == "" {
			openaiMsg.Content = contentBlocks[0].Text
		} else {
			contentList := make([]any, 0, len(contentBlocks))
			for _, c := range contentBlocks {
				if c.Type == "text" {
					contentList = append(contentList, map[string]string{"type": "text", "text": c.Text})
				} else if c.Type == "tool_result" {
					// tool_result -> OpenAI tool message
					contentList = append(contentList, map[string]any{
						"type":         "tool",
						"tool_call_id": c.ToolUseId,
						"content":      c.Text,
					})
				} else if c.Type == "image" && c.Source != nil {
					if c.Source.Url != "" {
						// Remote URL image
						contentList = append(contentList, map[string]any{
							"type": "image_url",
							"image_url": map[string]string{"url": c.Source.Url},
						})
					} else if c.Source.Data != "" {
						// Base64 image
						contentList = append(contentList, map[string]any{
							"type": "image_url",
							"image_url": map[string]string{
								"url": fmt.Sprintf("data:%s;base64,%s", c.Source.MediaType, c.Source.Data),
							},
						})
					}
				}
			}
			openaiMsg.Content = contentList
		}
		messages = append(messages, openaiMsg)
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
	}

	// Handle thinking budget (some backends support this via custom fields)
	if req.Thinking != nil && req.Thinking.BudgetTokens > 0 {
		openaiReq.ExtraFields = map[string]any{
			"thinking_budget_tokens": req.Thinking.BudgetTokens,
		}
	}

	// Preserve any extra fields from original request
	if len(req.ExtraFields) > 0 {
		if openaiReq.ExtraFields == nil {
			openaiReq.ExtraFields = make(map[string]any)
		}
		for k, v := range req.ExtraFields {
			if k != "thinking_budget_tokens" { // Don't overwrite if already set
				openaiReq.ExtraFields[k] = v
			}
		}
	}

	return openaiReq
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
				return map[string]string{"type": "function", "function": name}
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
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
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
		switch openaiResp.Choices[0].FinishReason {
		case "stop":
			stopReason = "end_turn"
		case "length":
			stopReason = "max_tokens"
		case "tool_calls":
			stopReason = "tool_use"
		}
	}

	// Build content blocks - text and/or tool_use
	var contentBlocks []map[string]any

	// Text content (may be empty if only tool_calls)
	textContent := ""
	if len(openaiResp.Choices) > 0 {
		textContent = openaiResp.Choices[0].Message.Content
		// Fallback to reasoning_content if content is empty/null (e.g., GLM model)
		if textContent == "" && openaiResp.Choices[0].Message.ReasoningContent != "" {
			textContent = openaiResp.Choices[0].Message.ReasoningContent
		}
	}
	if textContent != "" {
		contentBlocks = append(contentBlocks, map[string]any{
			"type": "text",
			"text": textContent,
		})
	}

	// Tool use blocks
	if len(openaiResp.Choices) > 0 {
		for _, tc := range openaiResp.Choices[0].Message.ToolCalls {
			// Parse arguments JSON string to map
			var toolInput any
			if tc.Function.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &toolInput)
			}
			// Generate tool use ID if not provided
			toolUseId := tc.ID
			if toolUseId == "" {
				toolUseId = fmt.Sprintf("toolu_%d", len(contentBlocks))
			}
			contentBlocks = append(contentBlocks, map[string]any{
				"type": "tool_use",
				"id":   toolUseId,
				"name": tc.Function.Name,
				"input": toolInput,
			})
		}
	}

	// If no content blocks, add empty text block
	if len(contentBlocks) == 0 {
		contentBlocks = append(contentBlocks, map[string]any{
			"type": "text",
			"text": "",
		})
	}

	anthropicResp := map[string]any{
		"id":          openaiResp.Id,
		"type":        "message",
		"role":        "assistant",
		"content":     contentBlocks,
		"model":       originModel,
		"stop_reason":  stopReason,
		"usage": map[string]int{
			"input_tokens":  openaiResp.Usage.PromptTokens,
			"output_tokens": openaiResp.Usage.CompletionTokens,
		},
	}

	// Apply HideUpstreamModel: replace with originModel if hideUpstreamModel is true
	// If hideUpstreamModel is false, keep actual upstream model name
	if !hideUpstreamModel {
		anthropicResp["model"] = openaiResp.Model
	}

	return json.Marshal(anthropicResp)
}

func convertOpenAIStreamToAnthropic(respBody []byte, originModel string, hideUpstreamModel bool) ([]byte, error) {
	// OpenAI SSE: data: {"id":"...","choices":[{"delta":{"content":"..."}}]}
	lines := strings.Split(string(respBody), "\n")
	var anthropicLines []string
	var messageId, modelName string
	var stopReasonStr string
	var inputTokens, outputTokens int
	contentStarted := false

	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue // Don't output [DONE] in Anthropic SSE
		}

		var chunk struct {
			Id      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ToolCalls        []struct {
						Index int `json:"index"`
						Type  string `json:"type"`
						ID    string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
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

		// Capture message metadata from first chunk
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
		content := delta.Content
		if content == "" && delta.ReasoningContent != "" {
			content = delta.ReasoningContent
		}
		finishReason := chunk.Choices[0].FinishReason

		// Capture usage info
		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}

		// Capture stop reason
		if finishReason != "" && finishReason != "null" {
			switch finishReason {
			case "stop":
				stopReasonStr = "end_turn"
			case "length":
				stopReasonStr = "max_tokens"
			case "tool_calls":
				stopReasonStr = "tool_use"
			default:
				stopReasonStr = "end_turn"
			}
		}

		// Output message_start on first content
		if content != "" && !contentStarted {
			contentStarted = true
			// message_start event
			messageStart := map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id":          messageId,
					"type":        "message",
					"role":        "assistant",
					"content":     []any{},
					"model":       modelName,
					"stop_reason": nil,
					"stop_sequence": nil,
					"usage": map[string]int{
						"input_tokens":  inputTokens,
						"output_tokens": 0,
					},
				},
			}
			startData, _ := json.Marshal(messageStart)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: message_start\ndata: %s", string(startData)))

			// content_block_start event
			contentBlockStart := map[string]any{
				"type":  "content_block_start",
				"index": 0,
				"content_block": map[string]any{
					"type": "text",
					"text": "",
				},
			}
			blockStartData, _ := json.Marshal(contentBlockStart)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(blockStartData)))
		}

		// content_block_delta event - use text_delta (not text)
		if content != "" {
			contentDelta := map[string]any{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]string{
					"type": "text_delta",
					"text": content,
				},
			}
			deltaData, _ := json.Marshal(contentDelta)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(deltaData)))
		}

		// Handle tool calls in delta (streaming tool_use)
		for _, tc := range delta.ToolCalls {
			// content_block_start for tool_use
			toolBlockStart := map[string]any{
				"type":  "content_block_start",
				"index": tc.Index,
				"content_block": map[string]any{
					"type": "tool_use",
					"id":   tc.ID,
					"name": tc.Function.Name,
					"input": map[string]any{},
				},
			}
			toolBlockData, _ := json.Marshal(toolBlockStart)
			anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_start\ndata: %s", string(toolBlockData)))

			// input_json_delta for tool arguments
			if tc.Function.Arguments != "" {
				inputDelta := map[string]any{
					"type":  "content_block_delta",
					"index": tc.Index,
					"delta": map[string]string{
						"type":       "input_json_delta",
						"partial_json": tc.Function.Arguments,
					},
				}
				inputDeltaData, _ := json.Marshal(inputDelta)
				anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_delta\ndata: %s", string(inputDeltaData)))
			}
		}
	}

	// Ensure we have a stop_reason
	if stopReasonStr == "" {
		stopReasonStr = "end_turn"
	}

	// content_block_stop event
	contentBlockStop := map[string]any{
		"type":  "content_block_stop",
		"index": 0,
	}
	stopData, _ := json.Marshal(contentBlockStop)
	anthropicLines = append(anthropicLines, fmt.Sprintf("event: content_block_stop\ndata: %s", string(stopData)))

	// message_delta event with stop_reason
	messageDelta := map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReasonStr,
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"output_tokens": outputTokens,
		},
	}
	deltaData, _ := json.Marshal(messageDelta)
	anthropicLines = append(anthropicLines, fmt.Sprintf("event: message_delta\ndata: %s", string(deltaData)))

	// message_stop event
	messageStop := map[string]any{
		"type": "message_stop",
	}
	stopEventData, _ := json.Marshal(messageStop)
	anthropicLines = append(anthropicLines, fmt.Sprintf("event: message_stop\ndata: %s", string(stopEventData)))

	// SSE events must be separated by an empty line.
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

// RelayAnthropicPassthrough handles Anthropic API /v1/messages requests in passthrough mode
// This is used for vLLM backends that natively support Anthropic API format
func RelayAnthropicPassthrough(c *gin.Context) {
	ctx := c.Request.Context()
	requestId := c.GetString(helper.RequestIdKey)

	// Get channel info from context (set by Distribute middleware)
	channelId := c.GetInt(ctxkey.ChannelId)
	channelName := c.GetString(ctxkey.ChannelName)
	baseURL := c.GetString(ctxkey.BaseURL)

	// Read request body
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

		// Convert to OpenAI format
		openaiReq := ConvertAnthropicToOpenAI(&anthropicReq)

		// Apply model name mapping using context mapping (same source as RelayTextHelper)
		modelMapping := c.GetStringMapString(ctxkey.ModelMapping)
		if modelMapping != nil {
			if mappedModel, ok := modelMapping[openaiReq.Model]; ok && mappedModel != "" {
				logger.Debugf(ctx, "model mapping (ctx): %s -> %s", openaiReq.Model, mappedModel)
				openaiReq.Model = mappedModel
			}
		}

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

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Header(k, v)
		}
	}

	// Track response content-type for final writeback.
	responseContentType := resp.Header.Get("Content-Type")
	if responseContentType == "" {
		responseContentType = "application/json"
	}

	// Copy response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
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
	c.Data(resp.StatusCode, responseContentType, respBody)
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
