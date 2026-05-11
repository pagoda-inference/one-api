package controller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
	relaymeta "github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
	"github.com/pagoda-inference/one-api/common/helper"
	billing "github.com/pagoda-inference/one-api/relay/billing"
	billingratio "github.com/pagoda-inference/one-api/relay/billing/ratio"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/relay"
)

// RelayResponsesHelper handles the actual relay logic for OpenAI Responses API
func RelayResponsesHelper(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	startTime := time.Now()

	// 1. Parse request
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return openai.ErrorWrapper(fmt.Errorf("failed to read request body: %w", err), "invalid_request", http.StatusBadRequest)
	}

	responsesReq, err := openai.ParseResponsesRequest(requestBody)
	if err != nil {
		return openai.ErrorWrapper(fmt.Errorf("failed to parse Responses request: %w", err), "invalid_request", http.StatusBadRequest)
	}

	// Validate request
	if err := openai.ValidateResponsesRequest(responsesReq); err != nil {
		return openai.ErrorWrapper(err, "invalid_request", http.StatusBadRequest)
	}

	// 2. Build meta
	meta := relaymeta.GetByContext(c)
	meta.Mode = relaymode.Responses
	meta.OriginModelName = responsesReq.Model
	meta.IsStream = responsesReq.Stream

	// 3. Model mapping
	responsesReq.Model, _ = getMappedModelName(responsesReq.Model, meta.ModelMapping)
	meta.ActualModelName = responsesReq.Model

	// 4. Get model info for pricing
	modelInfo, err := getModelById(responsesReq.Model)
	if err != nil {
		logger.Warnf(ctx, "model not found: %s, using default price 0", responsesReq.Model)
		modelInfo = nil
	}
	inputPrice := 0.0
	outputPrice := 0.0
	if modelInfo != nil {
		inputPrice = modelInfo.InputPrice
		outputPrice = modelInfo.OutputPrice
	}
	groupRatio := billingratio.GetGroupRatio(meta.Group)

	// 5. Count prompt tokens
	promptTokens := openai.CountResponsesInputTokens(responsesReq.Input, responsesReq.Model)
	if responsesReq.Instructions != "" {
		promptTokens += openai.GetResponsesInstructionTokens(responsesReq.Instructions, responsesReq.Model)
	}
	meta.PromptTokens = promptTokens

	// 6. Convert to Chat request (needed for pre-consume and upstream)
	chatReq := relaymodel.ConvertResponsesToChatRequest(responsesReq)

	// 7. Pre-consume quota
	preConsumedQuota, bizErr := PreConsumeQuota(ctx, chatReq, promptTokens, inputPrice, groupRatio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "PreConsumeQuota failed: %+v", *bizErr)
		return bizErr
	}

	// 8. Marshal chat request
	chatBody, err := json.Marshal(chatReq)
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("failed to marshal chat request: %w", err), "convert_request_failed", http.StatusInternalServerError)
	}

	// 8. Get adaptor and make request
	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// Build request URL
	requestURL, err := adaptor.GetRequestURL(meta)
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(err, "get_request_url_failed", http.StatusInternalServerError)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(chatBody))
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(err, "create_request_failed", http.StatusInternalServerError)
	}

	// Setup headers
	if err := adaptor.SetupRequestHeader(c, req, meta); err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(err, "setup_request_header_failed", http.StatusInternalServerError)
	}

	// Do request
	logger.Debugf(ctx, "DoRequest: APIType=%d, Mode=%d, ChannelType=%d", meta.APIType, meta.Mode, meta.ChannelType)
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}

	// 9. Handle response
	if responsesReq.Stream {
		return handleResponsesStream(c, resp, meta, preConsumedQuota, inputPrice, outputPrice, groupRatio, startTime)
	}
	return handleResponsesNonStream(c, resp, meta, preConsumedQuota, inputPrice, outputPrice, groupRatio, startTime)
}

func handleResponsesNonStream(c *gin.Context, resp *http.Response, meta *relaymeta.Meta, preConsumedQuota int64, inputPrice, outputPrice float64, groupRatio float64, startTime time.Time) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()

	// Ensure resp.Body is closed
	defer resp.Body.Close()

	// Check for errors from upstream
	if resp.StatusCode != http.StatusOK {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return relayErrorHandler(resp)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("failed to read response body: %w", err), "read_response_failed", http.StatusInternalServerError)
	}

	// Parse Chat response
	var chatResp relaymodel.ChatCompletionsResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("failed to unmarshal chat response: %w", err), "invalid_response", http.StatusInternalServerError)
	}

	// Check for error in response
	if chatResp.Error.Message != "" {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return &relaymodel.ErrorWithStatusCode{
			Error:      chatResp.Error,
			StatusCode: resp.StatusCode,
		}
	}

	// Convert to Responses format
	responsesResp := relaymodel.ConvertChatResponseToResponses(&chatResp, meta.RequestURLPath)

	// Log usage
	usageSource := "exact"
	if responsesResp.Usage == nil || responsesResp.Usage.TotalTokens == 0 {
		usageSource = "fallback"
		// Try to estimate from response content
		responsesResp.Usage = estimateUsageFromResponse(chatResp, meta.ActualModelName, meta.PromptTokens, inputPrice, outputPrice, groupRatio)
	}

	logResponse(ctx, meta, preConsumedQuota, responsesResp.Usage, usageSource, startTime, "")

	// Post-consume quota async (or rollback if usage is still nil)
	if responsesResp.Usage != nil && responsesResp.Usage.TotalTokens > 0 {
		go PostConsumeQuota(ctx, responsesResp.Usage, meta, nil, inputPrice, outputPrice, groupRatio, preConsumedQuota, false)
	} else {
		// Usage still nil/unavailable - rollback pre-consumed quota
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
	}

	c.JSON(http.StatusOK, responsesResp)
	return nil
}

func handleResponsesStream(c *gin.Context, resp *http.Response, meta *relaymeta.Meta, preConsumedQuota int64, inputPrice, outputPrice float64, groupRatio float64, startTime time.Time) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()

	if !config.ResponsesStreamEnabled {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("streaming is not enabled for Responses API"), "stream_disabled", http.StatusBadRequest)
	}

	// Ensure resp.Body is closed
	defer resp.Body.Close()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(gin.Flusher)
	if !ok {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("streaming not supported"), "stream_not_supported", http.StatusInternalServerError)
	}

	// Track stream state
	accumulatedText := ""
	var finalUsage *relaymodel.Usage
	var status string
	var responseCreatedSent bool
	var responseID string
	requestID := c.GetString(helper.RequestIdKey)

	// Create context for cancellation
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Use bufio.Reader for proper SSE line parsing
	br := bufio.NewReader(resp.Body)

	for {
		select {
		case <-streamCtx.Done():
			// Client disconnected or context cancelled
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			logResponse(ctx, meta, preConsumedQuota, nil, "interrupted", startTime, "client_disconnect")
			return openai.ErrorWrapper(fmt.Errorf("stream interrupted"), "stream_interrupted", http.StatusInternalServerError)
		default:
		}

		// SSE event accumulator
		var eventData string

		// Read lines until empty line (end of SSE event) or EOF
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Stream ended
					break
				}
				billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
				logResponse(ctx, meta, preConsumedQuota, nil, "interrupted", startTime, "read_error")
				return openai.ErrorWrapper(fmt.Errorf("failed to read stream: %w", err), "stream_read_error", http.StatusInternalServerError)
			}

			line = strings.TrimRight(line, "\r\n")

			// Empty line marks end of SSE event
			if line == "" {
				break
			}

			// Comment line, skip
			if strings.HasPrefix(line, ":") {
				continue
			}

			// Non-data: lines (like event: type), skip
			if !strings.HasPrefix(line, "data:") {
				continue
			}

			// Extract data content
			dataStr := strings.TrimPrefix(line, "data:")
			dataStr = strings.TrimSpace(dataStr)
			if dataStr == "" {
				continue
			}
			if dataStr == "[DONE]" {
				break
			}

			// Accumulate multi-line data (SSE concatenates with newlines)
			if eventData != "" {
				eventData += "\n"
			}
			eventData += dataStr
		}

		// Skip empty events
		if eventData == "" {
			continue
		}

		// Parse the accumulated event data as JSON
		var chatResp relaymodel.ChatCompletionsStreamResponse
		if err := json.Unmarshal([]byte(eventData), &chatResp); err != nil {
			logger.Warnf(ctx, "failed to unmarshal SSE data: %v, data: %s", err, eventData)
			continue
		}

		// Send response.created only once (first chunk)
		if !responseCreatedSent && chatResp.ID != "" {
			responseID = chatResp.ID
			createdEvent := &relaymodel.ResponsesStreamEvent{
				Event: "response.created",
				Data: relaymodel.ResponseCreatedEvent{
					Response: struct {
						ID     string `json:"id"`
						Object string `json:"object"`
						Status string `json:"status"`
					}{
						ID:     chatResp.ID,
						Object: "response",
						Status: "in_progress",
					},
				},
			}
			flusher.Write([]byte(openai.SSEFormatResponsesEvent(createdEvent)))
			flusher.Flush()
			responseCreatedSent = true
		}

		// Build and send delta events
		events, err := openai.BuildResponsesStreamEvent(&chatResp)
		if err != nil {
			logger.Warnf(ctx, "BuildResponsesStreamEvent error: %v", err)
			continue
		}

		for _, event := range events {
			// Skip response.created (already sent above)
			if event.Event == "response.created" {
				continue
			}

			// Track usage from done event
			if event.Event == "response.done" {
				if doneEvent, ok := event.Data.(relaymodel.ResponseDoneEvent); ok {
					finalUsage = doneEvent.Response.Usage
				}
			}

			// Track text for fallback usage calculation
			if event.Event == "response.output_text.delta" {
				if deltaEvent, ok := event.Data.(relaymodel.OutputTextDeltaEvent); ok {
					accumulatedText += deltaEvent.Delta
				}
			}

			sseLine := openai.SSEFormatResponsesEvent(event)
			if sseLine != "" {
				flusher.Write([]byte(sseLine))
				flusher.Flush()
			}
		}

		// Check for completion
		if len(chatResp.Choices) > 0 && chatResp.Choices[0].FinishReason != nil {
			status = *chatResp.Choices[0].FinishReason
		}
	}

	// Ensure response.done is always sent
	if finalUsage == nil {
		// Use fallback usage if no usage from upstream
		if accumulatedText != "" {
			finalUsage = &relaymodel.Usage{
				PromptTokens:     meta.PromptTokens,
				CompletionTokens: openai.CountTokenInput(accumulatedText, meta.ActualModelName),
				TotalTokens:      meta.PromptTokens + openai.CountTokenInput(accumulatedText, meta.ActualModelName),
			}
		}
	}

	// Fallback for responseID if never set
	if responseID == "" {
		if requestID != "" {
			responseID = "resp_" + requestID
		} else {
			responseID = "resp_" + fmt.Sprintf("%d", time.Now().UnixNano())
		}
	}

	doneEvent := &relaymodel.ResponsesStreamEvent{
		Event: "response.done",
		Data: relaymodel.ResponseDoneEvent{
			Response: struct {
				ID     string            `json:"id"`
				Object string            `json:"object"`
				Status string            `json:"status"`
				Usage  *relaymodel.Usage `json:"usage,omitempty"`
			}{
				ID:     responseID,
				Object: "response",
				Status: "completed",
				Usage:  finalUsage,
			},
		},
	}
	flusher.Write([]byte(openai.SSEFormatResponsesEvent(doneEvent)))
	flusher.Flush()

	// Finalize quota
	usageSource := "exact"
	if finalUsage == nil || finalUsage.TotalTokens == 0 {
		usageSource = "fallback"
		if finalUsage == nil {
			// No usage at all - return pre-consumed quota
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			logResponse(ctx, meta, preConsumedQuota, nil, usageSource, startTime, status)
			return nil
		}
	}

	logResponse(ctx, meta, preConsumedQuota, finalUsage, usageSource, startTime, status)

	// Post-consume with actual/fallback usage
	go PostConsumeQuota(ctx, finalUsage, meta, nil, inputPrice, outputPrice, groupRatio, preConsumedQuota, false)

	return nil
}

// getModelById returns model info by ID
func getModelById(modelId string) (*relaymodel.ModelInfo, error) {
	return model.GetModelById(modelId)
}

// estimateUsageFromResponse estimates usage when upstream doesn't provide it
func estimateUsageFromResponse(chatResp relaymodel.ChatCompletionsResponse, modelName string, promptTokens int, inputPrice, outputPrice, groupRatio float64) *relaymodel.Usage {
	// Calculate completion tokens from response
	completionTokens := 0
	if len(chatResp.Choices) > 0 {
		content := chatResp.Choices[0].Message.StringContent()
		completionTokens = openai.CountTokenInput(content, modelName)
	}

	multiplier := config.ResponsesUsageFallbackMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// Apply multiplier for fallback estimation
	estimatedTotal := int64(float64(promptTokens+completionTokens) * multiplier)

	return &relaymodel.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:       int(estimatedTotal),
	}
}

// logResponse logs structured information about the response
func logResponse(ctx context.Context, meta *relaymeta.Meta, preConsumedQuota int64, usage *relaymodel.Usage, usageSource string, startTime time.Time, status string) {
	latencyMs := time.Since(startTime).Milliseconds()

	if usage == nil {
		logger.Infof(ctx, "[Responses] request_id=%s user_id=%d token_id=%d model=%s stream=%v pre_consumed=%d final_quota=0 usage_source=%s status=%s latency_ms=%d error_type=none",
			meta.RequestURLPath, meta.UserId, meta.TokenId, meta.ActualModelName, meta.IsStream, preConsumedQuota, usageSource, status, latencyMs)
		return
	}

	logger.Infof(ctx, "[Responses] request_id=%s user_id=%d token_id=%d model=%s stream=%v pre_consumed=%d final_quota=%d usage_source=%s status=%s latency_ms=%d error_type=none prompt_tokens=%d completion_tokens=%d total_tokens=%d",
		meta.RequestURLPath, meta.UserId, meta.TokenId, meta.ActualModelName, meta.IsStream, preConsumedQuota, usage.TotalTokens, usageSource, status, latencyMs, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
}

// relayErrorHandler handles error responses
func relayErrorHandler(resp *http.Response) *relaymodel.ErrorWithStatusCode {
	body, _ := io.ReadAll(resp.Body)
	var errResp map[string]any
	json.Unmarshal(body, &errResp)

	message := "upstream error"
	if err, ok := errResp["error"].(map[string]any); ok {
		if msg, ok := err["message"].(string); ok {
			message = msg
		}
	}

	return &relaymodel.ErrorWithStatusCode{
		Error: relaymodel.Error{
			Message: message,
			Type:    "upstream_error",
			Param:   "",
			Code:    fmt.Sprintf("status_%d", resp.StatusCode),
		},
		StatusCode: resp.StatusCode,
	}
}

// getMappedModelName applies model name mapping
func getMappedModelName(modelName string, modelMapping map[string]string) (string, bool) {
	if modelMapping == nil {
		return modelName, false
	}
	if mapped, ok := modelMapping[modelName]; ok {
		return mapped, true
	}
	return modelName, false
}
