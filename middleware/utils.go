package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/common/logger"
)

func abortWithMessage(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": helper.MessageWithRequestId(message, c.GetString(helper.RequestIdKey)),
			"type":    "one_api_error",
		},
	})
	c.Abort()
	logger.Error(c.Request.Context(), message)
}

func getRequestModel(c *gin.Context) (string, error) {
	if strings.HasPrefix(c.Request.URL.Path, "/v1/file_parse") {
		modelName := strings.TrimSpace(c.PostForm("model"))
		if modelName != "" {
			return modelName, nil
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/predict") {
		modelName := strings.TrimSpace(c.PostForm("model"))
		if modelName != "" {
			return modelName, nil
		}
	}
	// Video create (POST /v1/videos and /v1/videos/sync) supports both JSON and
	// multipart/form-data (vLLM-OMNI wire format). For multipart we must read
	// the request body into memory BEFORE calling c.PostForm: net/http's
	// FormValue triggers ParseMultipartForm, which consumes c.Request.Body.
	// If we let that happen first, a later GetRequestBody / UnMarshalBodyReusable
	// will see an empty stream and the multipart body never reaches the
	// upstream — causing upstream "prompt field required" errors. Buffering
	// first also lets TokenAuth's downstream Distribute / channel handlers
	// reuse the bytes via c.Get(ctxkey.KeyRequestBody).
	//
	// GetRequestBody closes c.Request.Body after reading, so we must reset it
	// to a fresh reader over the buffered bytes before calling PostForm;
	// otherwise ParseMultipartForm sees an empty body, returns no fields, and
	// the request model ends up empty — which makes Distribute skip channel
	// selection and the handler reports "no available video channel".
	if strings.HasPrefix(c.Request.URL.Path, "/v1/videos") && c.Request.Method == http.MethodPost {
		if bodyBytes, err := common.GetRequestBody(c); err == nil {
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			modelName := strings.TrimSpace(c.PostForm("model"))
			if modelName != "" {
				return modelName, nil
			}
		}
	}
	var modelRequest ModelRequest
	err := common.UnmarshalBodyReusable(c, &modelRequest)
	if err != nil {
		return "", fmt.Errorf("common.UnmarshalBodyReusable failed: %w", err)
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/moderations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "text-moderation-stable"
		}
	}
	if strings.HasSuffix(c.Request.URL.Path, "embeddings") {
		if modelRequest.Model == "" {
			modelRequest.Model = c.Param("model")
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/images/generations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "dall-e-2"
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") || strings.HasPrefix(c.Request.URL.Path, "/v1/audio/translations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "whisper-1"
		}
	}
	return modelRequest.Model, nil
}

func isModelInList(modelName string, models string) bool {
	modelList := strings.Split(models, ",")
	for _, model := range modelList {
		if modelName == model {
			return true
		}
	}
	return false
}
