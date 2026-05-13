package controller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/relay"
	adaptoriface "github.com/pagoda-inference/one-api/relay/adaptor"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	"github.com/pagoda-inference/one-api/relay/apitype"
	billing "github.com/pagoda-inference/one-api/relay/billing"
	billingratio "github.com/pagoda-inference/one-api/relay/billing/ratio"
	relaymeta "github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
)

var fallbackToolIDRegex = regexp.MustCompile(`^call_\d+_(\d+)$`)
var functionStyleToolIDRegex = regexp.MustCompile(`^functions\.[^:]+:(\d+)$`)

func parseToolPosFromItemID(itemID string) (int, bool) {
	id := strings.TrimSpace(itemID)
	if id == "" {
		return 0, false
	}
	if m := fallbackToolIDRegex.FindStringSubmatch(id); len(m) == 2 {
		n, err := strconv.Atoi(m[1])
		if err == nil {
			return n, true
		}
	}
	if m := functionStyleToolIDRegex.FindStringSubmatch(id); len(m) == 2 {
		n, err := strconv.Atoi(m[1])
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

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

	// 8. Marshal request bodies for dual-path upstream routing.
	responsesBody, err := json.Marshal(responsesReq)
	if err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("failed to marshal responses request: %w", err), "convert_request_failed", http.StatusInternalServerError)
	}

	// Fallback body for chat-completions compatible upstreams.
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

	useNativeResponses := shouldUseNativeResponsesUpstream(meta) && !config.ResponsesStrictCompat
	var resp *http.Response
	if useNativeResponses {
		// Primary path: passthrough to native responses endpoint.
		requestURL, urlErr := adaptor.GetRequestURL(meta)
		if urlErr != nil {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			return openai.ErrorWrapper(urlErr, "get_request_url_failed", http.StatusInternalServerError)
		}
		req, reqErr := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(responsesBody))
		if reqErr != nil {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			return openai.ErrorWrapper(reqErr, "create_request_failed", http.StatusInternalServerError)
		}
		if err = adaptor.SetupRequestHeader(c, req, meta); err != nil {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			return openai.ErrorWrapper(err, "setup_request_header_failed", http.StatusInternalServerError)
		}
		logger.Debugf(ctx, "Responses route: native passthrough, APIType=%d, ChannelType=%d, base=%s", meta.APIType, meta.ChannelType, meta.BaseURL)
		resp, err = client.HTTPClient.Do(req)
		if err != nil {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
			return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
		}

		// If native path is incompatible, fallback to chat/completions.
		if meta.APIType == apitype.OpenAI && resp.StatusCode != http.StatusOK {
			probeBody, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if shouldFallbackToChatCompletions(resp.StatusCode, probeBody) {
				logger.Warnf(ctx, "responses upstream incompatible, fallback to chat/completions: status=%d body=%s", resp.StatusCode, truncateForLog(probeBody, 512))
				resp, err = retryResponsesAsChat(ctx, c, adaptor, meta, chatBody)
				if err != nil {
					billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
					return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
				}
			} else {
				resp.Body = io.NopCloser(bytes.NewBuffer(probeBody))
			}
		}
	} else {
		// Compatibility path (same spirit as anthropic->chat conversion):
		// non-native responses upstreams should directly use chat/completions.
		logger.Debugf(ctx, "Responses route: direct chat fallback, APIType=%d, ChannelType=%d, base=%s", meta.APIType, meta.ChannelType, meta.BaseURL)
		resp, err = retryResponsesAsChat(ctx, c, adaptor, meta, chatBody)
		if err != nil {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
		}
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

	// Native Responses payload passthrough.
	if isNativeResponsesBody(body) {
		var nativeResp relaymodel.ResponsesResponse
		usageSource := "exact"
		if err := json.Unmarshal(body, &nativeResp); err != nil {
			usageSource = "fallback"
		}
		if nativeResp.Usage == nil || nativeResp.Usage.TotalTokens == 0 {
			usageSource = "fallback"
		}
		logResponse(ctx, meta, preConsumedQuota, nativeResp.Usage, usageSource, startTime, nativeResp.Status)
		if nativeResp.Usage != nil && nativeResp.Usage.TotalTokens > 0 {
			go PostConsumeQuota(ctx, nativeResp.Usage, meta, nil, inputPrice, outputPrice, groupRatio, preConsumedQuota, false)
		} else {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		}
		for k, v := range resp.Header {
			if len(v) > 0 {
				c.Writer.Header().Set(k, v[0])
			}
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, _ = c.Writer.Write(body)
		return nil
	}

	// Parse Chat response
	var chatResp relaymodel.ChatCompletionsResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("failed to unmarshal chat response: %w", err), "invalid_response", http.StatusInternalServerError)
	}

	// Check for error in response
	if chatResp.Error != nil && chatResp.Error.Message != "" {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return &relaymodel.ErrorWithStatusCode{
			Error:      *chatResp.Error,
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

	// Upstream returned non-200 before any stream content.
	// Surface the real upstream error instead of misleading EOF-prelude errors.
	if resp.StatusCode != http.StatusOK {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return relayErrorHandler(resp)
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
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
	sentOutputItems := map[string]bool{}
	canonicalToolIDByPos := map[int]string{}
	requestID := c.GetString(helper.RequestIdKey)

	// Create context for cancellation
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Use bufio.Reader for proper SSE line parsing
	br := bufio.NewReader(resp.Body)
	firstEventLine := ""
	firstDataLine := ""
	rawPrelude := ""
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			return openai.ErrorWrapper(fmt.Errorf("failed to read stream prelude: %w", err), "stream_read_error", http.StatusInternalServerError)
		}
		rawPrelude += line
		trimmed := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(trimmed, "event:") && firstEventLine == "" {
			firstEventLine = trimmed
		}
		if strings.HasPrefix(trimmed, "data:") && firstDataLine == "" {
			firstDataLine = strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		}
		// End of first SSE event block.
		if trimmed == "" {
			break
		}
	}
	if strings.HasPrefix(firstEventLine, "event: response.") || isNativeResponsesDataLine(firstDataLine) {
		return handleNativeResponsesStream(c, resp, br, firstEventLine, firstDataLine, meta, preConsumedQuota, inputPrice, outputPrice, groupRatio, startTime)
	}
	br = bufio.NewReader(io.MultiReader(strings.NewReader(rawPrelude), br))

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
		streamEnded := false

		// Read lines until empty line (end of SSE event) or EOF
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Stream ended, break both loops gracefully
					streamEnded = true
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

		// EOF reached with no pending event data: finish outer loop
		if streamEnded && eventData == "" {
			break
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
		if !responseCreatedSent && chatResp.Id != "" {
			responseID = chatResp.Id
			createdEvent := &relaymodel.ResponsesStreamEvent{
				Event: "response.created",
				Data: relaymodel.ResponseCreatedEvent{
					Response: struct {
						ID     string `json:"id"`
						Object string `json:"object"`
						Status string `json:"status"`
					}{
						ID:     chatResp.Id,
						Object: "response",
						Status: "in_progress",
					},
				},
			}
			_, _ = c.Writer.Write([]byte(openai.SSEFormatResponsesEvent(createdEvent)))
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

			// Canonicalize tool call item IDs to avoid mixed IDs like:
			// "functions.get_weather:0" and "call_0_0" in the same stream.
			if event.Event == "response.output_item.added" {
				if itemEvent, ok := event.Data.(relaymodel.OutputItemCreatedEvent); ok {
					itemID := strings.TrimSpace(itemEvent.Item.ID)
					if pos, ok := parseToolPosFromItemID(itemID); ok {
						if strings.HasPrefix(itemID, "call_") {
							if canonical := canonicalToolIDByPos[pos]; canonical != "" && canonical != itemID {
								itemEvent.Item.ID = canonical
								if itemEvent.Item.FunctionCall != nil {
									itemEvent.Item.FunctionCall.ID = canonical
								}
								event.Data = itemEvent
								itemID = canonical
							}
						} else {
							if canonicalToolIDByPos[pos] == "" {
								canonicalToolIDByPos[pos] = itemID
							}
						}
					}
				}
			}
			if event.Event == "response.function_call_arguments.delta" {
				if deltaEvent, ok := event.Data.(relaymodel.FunctionCallArgumentsDeltaEvent); ok {
					itemID := strings.TrimSpace(deltaEvent.ItemID)
					if pos, ok := parseToolPosFromItemID(itemID); ok && strings.HasPrefix(itemID, "call_") {
						if canonical := canonicalToolIDByPos[pos]; canonical != "" && canonical != itemID {
							deltaEvent.ItemID = canonical
							event.Data = deltaEvent
						}
					}
				}
			}

			// Deduplicate output_item.added for the same item id within one stream.
			if event.Event == "response.output_item.added" {
				if itemEvent, ok := event.Data.(relaymodel.OutputItemCreatedEvent); ok {
					itemID := strings.TrimSpace(itemEvent.Item.ID)
					if itemID != "" {
						if sentOutputItems[itemID] {
							continue
						}
						sentOutputItems[itemID] = true
					}
				}
			}

			// Track usage from completion events
			if event.Event == "response.done" || event.Event == "response.completed" {
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
				_, _ = c.Writer.Write([]byte(sseLine))
				flusher.Flush()
			}
		}

		// Check for completion
		if len(chatResp.Choices) > 0 && chatResp.Choices[0].FinishReason != nil {
			status = *chatResp.Choices[0].FinishReason
		}
	}

	// Ensure response.completed is always sent
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
		Event: "response.completed",
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
	_, _ = c.Writer.Write([]byte(openai.SSEFormatResponsesEvent(doneEvent)))
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
func getModelById(modelId string) (*model.ModelInfo, error) {
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
		TotalTokens:      int(estimatedTotal),
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

func shouldUseNativeResponsesUpstream(meta *relaymeta.Meta) bool {
	if meta == nil {
		return false
	}
	// Only OpenAI-compatible channels with a real responses endpoint should use native passthrough.
	// Everything else behaves like anthropic: convert to chat/completions.
	switch meta.APIType {
	case apitype.OpenAI:
		base := strings.ToLower(strings.TrimSpace(meta.BaseURL))
		if strings.Contains(base, "api.openai.com") {
			return true
		}
		// Other OpenAI-compatible providers remain chat fallback by default unless explicitly whitelisted later.
		return false
	default:
		return false
	}
}

func isNativeResponsesBody(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	obj, _ := m["object"].(string)
	_, hasOutput := m["output"]
	_, hasStatus := m["status"]
	return obj == "response" || (hasOutput && hasStatus)
}

func isNativeResponsesDataLine(data string) bool {
	if data == "" || data == "[DONE]" {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return false
	}
	_, hasResponse := m["response"]
	_, hasDelta := m["delta"]
	_, hasItemID := m["item_id"]
	return hasResponse || (hasDelta && hasItemID)
}

func handleNativeResponsesStream(c *gin.Context, resp *http.Response, br *bufio.Reader, firstEventLine, firstDataLine string, meta *relaymeta.Meta, preConsumedQuota int64, inputPrice, outputPrice float64, groupRatio float64, startTime time.Time) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return openai.ErrorWrapper(fmt.Errorf("streaming not supported"), "stream_not_supported", http.StatusInternalServerError)
	}

	// Forward the first event block already read.
	if firstEventLine != "" {
		_, _ = c.Writer.Write([]byte(firstEventLine + "\n"))
	}
	if firstDataLine != "" {
		_, _ = c.Writer.Write([]byte("data: " + firstDataLine + "\n\n"))
	}
	flusher.Flush()

	var finalUsage *relaymodel.Usage
	parseDoneUsage := func(data string) {
		var envelope map[string]any
		if err := json.Unmarshal([]byte(data), &envelope); err != nil {
			return
		}
		respObj, _ := envelope["response"].(map[string]any)
		if respObj == nil {
			return
		}
		usageObj, _ := respObj["usage"].(map[string]any)
		if usageObj == nil {
			return
		}
		pt, _ := usageObj["prompt_tokens"].(float64)
		ct, _ := usageObj["completion_tokens"].(float64)
		tt, _ := usageObj["total_tokens"].(float64)
		finalUsage = &relaymodel.Usage{
			PromptTokens:     int(pt),
			CompletionTokens: int(ct),
			TotalTokens:      int(tt),
		}
	}
	if (strings.HasPrefix(firstEventLine, "event: response.done") || strings.HasPrefix(firstEventLine, "event: response.completed")) && firstDataLine != "" {
		parseDoneUsage(firstDataLine)
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
			return openai.ErrorWrapper(fmt.Errorf("failed to read native responses stream: %w", err), "stream_read_error", http.StatusInternalServerError)
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(trimmed, "event: response.done") || strings.HasPrefix(trimmed, "event: response.completed") {
			_, _ = c.Writer.Write([]byte(line))
			// done block should have following data line(s); parse on those lines.
			for {
				nextLine, err := br.ReadString('\n')
				if err != nil {
					break
				}
				_, _ = c.Writer.Write([]byte(nextLine))
				nextTrimmed := strings.TrimRight(nextLine, "\r\n")
				if strings.HasPrefix(nextTrimmed, "data:") {
					parseDoneUsage(strings.TrimSpace(strings.TrimPrefix(nextTrimmed, "data:")))
				}
				if nextTrimmed == "" {
					break
				}
			}
			flusher.Flush()
			continue
		}
		_, _ = c.Writer.Write([]byte(line))
		if strings.TrimSpace(line) == "" {
			flusher.Flush()
		}
	}

	usageSource := "exact"
	if finalUsage == nil || finalUsage.TotalTokens == 0 {
		usageSource = "fallback"
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		logResponse(ctx, meta, preConsumedQuota, nil, usageSource, startTime, "completed")
		return nil
	}
	logResponse(ctx, meta, preConsumedQuota, finalUsage, usageSource, startTime, "completed")
	go PostConsumeQuota(ctx, finalUsage, meta, nil, inputPrice, outputPrice, groupRatio, preConsumedQuota, false)
	return nil
}

func retryResponsesAsChat(ctx context.Context, c *gin.Context, adaptor adaptoriface.Adaptor, meta *relaymeta.Meta, chatBody []byte) (*http.Response, error) {
	upstreamMeta := *meta
	upstreamMeta.Mode = relaymode.ChatCompletions
	if upstreamMeta.APIType == apitype.OpenAI {
		upstreamMeta.RequestURLPath = "/v1/chat/completions"
	}
	requestURL, err := adaptor.GetRequestURL(&upstreamMeta)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewBuffer(chatBody))
	if err != nil {
		return nil, err
	}
	if err := adaptor.SetupRequestHeader(c, req, meta); err != nil {
		return nil, err
	}
	return client.HTTPClient.Do(req)
}

func shouldFallbackToChatCompletions(statusCode int, body []byte) bool {
	if statusCode == http.StatusOK {
		return false
	}
	switch statusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnprocessableEntity:
	default:
		return false
	}

	s := strings.ToLower(string(body))
	if strings.Contains(s, "missing") && strings.Contains(s, "input") {
		return true
	}
	if strings.Contains(s, "field required") && strings.Contains(s, "input") {
		return true
	}
	if strings.Contains(s, "/v1/responses") && strings.Contains(s, "not found") {
		return true
	}
	if strings.Contains(s, "unsupported") && strings.Contains(s, "responses") {
		return true
	}
	if strings.Contains(s, "unknown") && strings.Contains(s, "responses") {
		return true
	}
	return false
}

func truncateForLog(body []byte, max int) string {
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "..."
}
