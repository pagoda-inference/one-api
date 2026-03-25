package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

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
	group := c.GetString(ctxkey.Group)
	originalModel := c.GetString(ctxkey.OriginalModel)
	go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)
	requestId := c.GetString(helper.RequestIdKey)
	retryTimes := config.RetryTimes
	if !shouldRetry(c, bizErr.StatusCode) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		retryTimes = 0
	}
	for i := retryTimes; i > 0; i-- {
		channel, err := dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, i != retryTimes)
		if err != nil {
			logger.Errorf(ctx, "CacheGetRandomSatisfiedChannel failed: %+v", err)
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
	apiKey := c.Request.Header.Get("Authorization")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"type":    "authentication_error",
				"message": "Missing API key",
			},
		})
		return
	}

	// Use channel's API key if available
	channel, err := dbmodel.GetChannelById(channelId, true)
	if err == nil && channel.Key != "" {
		apiKey = fmt.Sprintf("Bearer %s", channel.Key)
	}

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

	// Build target URL
	if baseURL == "" && channel != nil {
		baseURL = channel.GetBaseURL()
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	targetURL := fmt.Sprintf("%s/v1/messages", baseURL)

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

	// Set headers - Anthropic API format
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "messages-2023-12-15")

	// Forward additional headers from client
	forwardHeaders := []string{
		"anthropic-version",
		"anthropic-beta",
	}
	for _, h := range forwardHeaders {
		if v := c.Request.Header.Get(h); v != "" {
			req.Header.Set(h, v)
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

	// Return response
	c.Data(resp.StatusCode, "application/json", respBody)
}
