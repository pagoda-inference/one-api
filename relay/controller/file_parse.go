package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/client"
	relaymeta "github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
)

const (
	defaultFileParseBackend = "vlm-auto-engine"
)

type fileParseTaskRouteInfo struct {
	BaseURL string
}

var fileParseTaskRouteMap sync.Map // map[taskID]fileParseTaskRouteInfo

type fileParseSubmitResponse struct {
	TaskID    string `json:"task_id"`
	StatusURL string `json:"status_url,omitempty"`
	ResultURL string `json:"result_url,omitempty"`
}

func fileParseForwardPath(baseURL string) string {
	b := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(b), "/tasks") {
		return b
	}
	return b + "/tasks"
}

func copyResponseToClient(c *gin.Context, resp *http.Response) *relaymodel.ErrorWithStatusCode {
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		lk := strings.ToLower(strings.TrimSpace(k))
		if lk == "content-length" || lk == "transfer-encoding" || lk == "connection" {
			continue
		}
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		return openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
	}
	return nil
}

func rewriteFileParseURLs(c *gin.Context, body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	taskID, _ := payload["task_id"].(string)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return body
	}
	payload["status_url"] = "/v1/file_parse/tasks/" + url.PathEscape(taskID)
	payload["result_url"] = "/v1/file_parse/tasks/" + url.PathEscape(taskID) + "/result"
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return rewritten
}

func RelayFileParseSubmitHelper(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	meta := relaymeta.GetByContext(c)
	targetURL := fileParseForwardPath(meta.BaseURL)

	incomingFile, incomingHeader, err := c.Request.FormFile("file")
	if err != nil {
		return openai.ErrorWrapper(err, "missing_file", http.StatusBadRequest)
	}
	defer incomingFile.Close()

	modelName := strings.TrimSpace(c.PostForm("model"))
	if modelName == "" {
		return openai.ErrorWrapper(fmt.Errorf("model is required"), "invalid_text_request", http.StatusBadRequest)
	}

	backend := strings.TrimSpace(c.PostForm("backend"))
	if backend == "" {
		backend = defaultFileParseBackend
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	outFile, err := writer.CreateFormFile("files", incomingHeader.Filename)
	if err != nil {
		return openai.ErrorWrapper(err, "create_files_part_failed", http.StatusInternalServerError)
	}
	if _, err = io.Copy(outFile, incomingFile); err != nil {
		return openai.ErrorWrapper(err, "copy_file_failed", http.StatusInternalServerError)
	}
	_ = writer.WriteField("model", modelName)
	_ = writer.WriteField("backend", backend)
	if err = writer.Close(); err != nil {
		return openai.ErrorWrapper(err, "close_multipart_writer_failed", http.StatusInternalServerError)
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, &body)
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}

	// Cache task -> baseURL mapping for follow-up status/result retrieval without exposing channel id.
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr == nil && len(bodyBytes) > 0 {
		var submitResp fileParseSubmitResponse
		if json.Unmarshal(bodyBytes, &submitResp) == nil && strings.TrimSpace(submitResp.TaskID) != "" {
			fileParseTaskRouteMap.Store(strings.TrimSpace(submitResp.TaskID), fileParseTaskRouteInfo{BaseURL: meta.BaseURL})
		}
		bodyBytes = rewriteFileParseURLs(c, bodyBytes)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	return copyResponseToClient(c, resp)
}

func RelayFileParseTaskHelper(c *gin.Context, withResult bool) *relaymodel.ErrorWithStatusCode {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		return openai.ErrorWrapper(fmt.Errorf("task_id is required"), "invalid_request_error", http.StatusBadRequest)
	}

	meta := relaymeta.GetByContext(c)
	baseURL := meta.BaseURL
	if v, ok := fileParseTaskRouteMap.Load(taskID); ok {
		if route, ok2 := v.(fileParseTaskRouteInfo); ok2 && strings.TrimSpace(route.BaseURL) != "" {
			baseURL = route.BaseURL
		}
	}
	if strings.TrimSpace(baseURL) == "" {
		return openai.ErrorWrapper(fmt.Errorf("task route not found"), "invalid_request_error", http.StatusNotFound)
	}

	root := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	targetPath := "/tasks/" + url.PathEscape(taskID)
	if withResult {
		targetPath += "/result"
	}
	targetURL := root + targetPath

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr == nil && len(bodyBytes) > 0 {
		bodyBytes = rewriteFileParseURLs(c, bodyBytes)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return copyResponseToClient(c, resp)
}
