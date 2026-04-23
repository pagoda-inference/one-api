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
	Model       string                 `json:"model"`
	Messages    []AnthropicMessage     `json:"messages"`
	System      string                 `json:"system,omitempty"`
	MaxTokens   int                    `json:"max_tokens"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
}

type AnthropicMessage struct {
	Role    string       `json:"role"`
	Content any          `json:"content"` // string or []AnthropicContent
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

	// Insert system prompt as first message if present
	if req.System != "" {
		messages = append(messages, model.Message{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		openaiMsg := model.Message{
			Role: msg.Role,
		}

		// Convert content blocks - handle both string and array formats
		contentBlocks := parseAnthropicContent(msg.Content)
		if len(contentBlocks) == 1 && contentBlocks[0].Type == "text" {
			openaiMsg.Content = contentBlocks[0].Text
		} else {
			contentList := make([]any, 0, len(contentBlocks))
			for _, c := range contentBlocks {
				if c.Type == "text" {
					contentList = append(contentList, map[string]string{"type": "text", "text": c.Text})
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

	return openaiReq
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
				Content         string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
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

	content := ""
	stopReason := "end_turn"
	if len(openaiResp.Choices) > 0 {
		content = openaiResp.Choices[0].Message.Content
		// Fallback to reasoning_content if content is empty/null (e.g., GLM model)
		if content == "" && openaiResp.Choices[0].Message.ReasoningContent != "" {
			content = openaiResp.Choices[0].Message.ReasoningContent
		}
		switch openaiResp.Choices[0].FinishReason {
		case "stop":
			stopReason = "end_turn"
		case "length":
			stopReason = "max_tokens"
		case "tool_calls":
			stopReason = "tool_use"
		}
	}

	anthropicResp := map[string]any{
		"id":         openaiResp.Id,
		"type":       "message",
		"role":       "assistant",
		"content":    []map[string]string{{"type": "text", "text": content}},
		"model":      originModel,
		"stop_reason": stopReason,
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
	// SSE lines: data: {"id":"...","choices":[{"delta":{"content":"..."}}]}
	lines := strings.Split(string(respBody), "\n")
	var anthropicLines []string

	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			anthropicLines = append(anthropicLines, "data: [DONE]")
			continue
		}

		var chunk struct {
			Id      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content         string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
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

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content == "" && chunk.Choices[0].Delta.ReasoningContent != "" {
			content = chunk.Choices[0].Delta.ReasoningContent
		}
		finishReason := chunk.Choices[0].FinishReason

		// Build Anthropic SSE format
		if content != "" {
			anthropicLines = append(anthropicLines, fmt.Sprintf(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text","text":"%s"}}`, content))
		}
		if finishReason != "" && finishReason != "null" {
			stopReason := "end_turn"
			switch finishReason {
			case "stop":
				stopReason = "end_turn"
			case "length":
				stopReason = "max_tokens"
			}
			anthropicLines = append(anthropicLines, fmt.Sprintf(`data: {"type":"message_delta","index":0,"delta":{"stop_reason":"%s"},"usage":{"output_tokens":0}}`, stopReason))
		}
		if chunk.Usage != nil {
			anthropicLines = append(anthropicLines, fmt.Sprintf(`data: {"type":"message_stop","index":0}`))
		}
	}

	result := strings.Join(anthropicLines, "\n")
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
			c.Data(resp.StatusCode, "application/json", respBody)
			return
		}
		respBody = convertedBody
		// Set Anthropic SSE content type
		c.Header("Content-Type", "text/event-stream")
	}

	// Return response
	c.Data(resp.StatusCode, "application/json", respBody)
}
