package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/relay/channeltype"
)

// VideoContentItemImageURL mirrors the OpenAI vision `image_url` object shape
// (relay/model.MessageContent.ImageURL), so the request body we forward to the
// upstream Ark/visual-generation API already matches its expected schema.
type VideoContentItemImageURL struct {
	URL string `json:"url,omitempty"`
}

type VideoContentItem struct {
	Type     string                    `json:"type"`
	Text     string                    `json:"text,omitempty"`
	ImageURL *VideoContentItemImageURL `json:"image_url,omitempty"`
}

// Content type identifiers used in the content[] array. These match the
// Ark visual-generation `content[].type` values for t2v / i2v inputs.
const (
	videoContentText      = "text"
	videoContentImageURL  = "image_url"
	videoKindTextToVideo  = "t2v"
	videoKindImageToVideo = "i2v"
)

type CreateVideoTaskRequest struct {
	Model      string             `json:"model"`
	Content    []VideoContentItem `json:"content"`
	Ratio      string             `json:"ratio"`
	Resolution string             `json:"resolution"`
	Duration   int                `json:"duration"`
	FPS        int                `json:"fps"`
	Seed       int                `json:"seed"`
}

type VideoTaskUsage struct {
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type VideoTaskResponse struct {
	ID              string `json:"id"`
	Model           string `json:"model,omitempty"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"created_at,omitempty"`
	UpdatedAt       int64  `json:"updated_at,omitempty"`
	Seed            int    `json:"seed,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
	Ratio           string `json:"ratio,omitempty"`
	Duration        int    `json:"duration,omitempty"`
	FramesPerSecond int    `json:"framespersecond,omitempty"`
	Content         *struct {
		VideoURL string `json:"video_url"`
	} `json:"content,omitempty"`
	Usage *VideoTaskUsage `json:"usage,omitempty"`
}

func estimateVideoOutputTokens(resolution, ratio string, durationSeconds, fps int) int64 {
	if durationSeconds <= 0 {
		durationSeconds = 5
	}
	if fps <= 0 {
		fps = 16
	}
	var width, height int64
	switch resolution {
	case "480p":
		switch ratio {
		case "4:3":
			width, height = 736, 544
		case "1:1":
			width, height = 640, 640
		default:
			width, height = 864, 480
		}
	case "1080p":
		switch ratio {
		case "4:3":
			width, height = 1664, 1248
		case "1:1":
			width, height = 1440, 1440
		default:
			width, height = 1920, 1088
		}
	default:
		switch ratio {
		case "4:3":
			width, height = 1120, 832
		case "1:1":
			width, height = 960, 960
		default:
			width, height = 1248, 704
		}
	}
	return int64(durationSeconds) * width * height * int64(fps) / 1024
}

func computeVideoFinalQuota(task *model.VideoGenerationTask) int64 {
	if task == nil || task.Status != model.VideoTaskStatusSucceeded {
		return 0
	}
	modelInfo, err := model.GetModelById(task.Model)
	if err != nil || modelInfo == nil || modelInfo.InputPrice <= 0 {
		return 0
	}
	tokenNum := estimateVideoOutputTokens(task.Resolution, task.Ratio, task.Duration, task.FramesPerSecond)
	if tokenNum <= 0 {
		return 0
	}
	quota := float64(tokenNum) * modelInfo.InputPrice / 1_000_000.0
	if quota <= 0 {
		return 0
	}
	return int64(quota + 0.5)
}

func settleVideoQuota(ctx context.Context, task *model.VideoGenerationTask) {
	if task == nil || task.QuotaSettled {
		return
	}
	finalQuota := computeVideoFinalQuota(task)

	// quota = -1 means unlimited, skip all quota settlement but still record usage log
	userQuota, err := model.CacheGetUserQuota(ctx, task.UserId)
	if err != nil {
		logger.SysError("error get user quota for video settlement: " + err.Error())
	}
	if err == nil && userQuota == -1 {
		logger.Info(ctx, fmt.Sprintf("user %d has unlimited quota, skipping video quota settlement", task.UserId))
		if finalQuota > 0 {
			logContent := fmt.Sprintf("视频计费：finalQuota=%d", finalQuota)
			model.RecordConsumeLog(ctx, &model.Log{
				UserId:           task.UserId,
				ChannelId:        task.ChannelId,
				PromptTokens:     0,
				CompletionTokens: 0,
				ModelName:        task.Model,
				Quota:            int(finalQuota),
				Content:          logContent,
			})
		}
		_ = model.UpdateVideoTaskByTaskId(task.TaskId, map[string]any{
			"quota_settled": true,
			"final_quota":   finalQuota,
			"updated_time":  helper.GetTimestamp(),
		})
		return
	}

	if task.TokenId != 0 && finalQuota > 0 {
		_ = model.PostConsumeTokenQuota(task.TokenId, finalQuota)
	} else if task.TokenId == 0 && finalQuota > 0 {
		_ = model.DecreaseUserQuota(task.UserId, finalQuota)
	}
	_ = model.CacheUpdateUserQuota(ctx, task.UserId)
	if finalQuota > 0 {
		model.UpdateUserUsedQuotaAndRequestCount(task.UserId, finalQuota)
		model.UpdateChannelUsedQuota(task.ChannelId, finalQuota)
	}
	_ = model.UpdateVideoTaskByTaskId(task.TaskId, map[string]any{
		"quota_settled": true,
		"final_quota":   finalQuota,
		"updated_time":  helper.GetTimestamp(),
	})
}

// detectVideoTaskKind inspects the content items to decide whether this is a
// text-to-video (t2v) or image-to-video (i2v) request. A task is treated as
// i2v when at least one content item carries an image_url; otherwise t2v.
// Mixed inputs (text + image_url) are allowed and classify as i2v, matching
// the upstream API which accepts a prompt plus a reference image.
func detectVideoTaskKind(content []VideoContentItem) string {
	for _, item := range content {
		if strings.TrimSpace(item.Type) == videoContentImageURL && item.ImageURL != nil && strings.TrimSpace(item.ImageURL.URL) != "" {
			return videoKindImageToVideo
		}
	}
	return videoKindTextToVideo
}

func defaultVideoModelByKind(kind string) string {
	if kind == videoKindImageToVideo {
		return "wan2.2-i2v"
	}
	return "wan2.2-t2v"
}

func normalizeCreateVideoTaskRequest(req *CreateVideoTaskRequest) {
	if req.Model == "" {
		req.Model = defaultVideoModelByKind(detectVideoTaskKind(req.Content))
	}
	if req.Resolution == "" {
		req.Resolution = "720p"
	}
	if req.Ratio == "" {
		req.Ratio = "16:9"
	}
	if req.Duration <= 0 {
		req.Duration = 5
	}
	if req.FPS <= 0 {
		req.FPS = 16
	}
	if req.Seed == 0 {
		req.Seed = 42
	}
}

func validateCreateVideoTaskRequest(req *CreateVideoTaskRequest) error {
	if req == nil {
		return fmt.Errorf("invalid request")
	}
	if len(req.Content) == 0 {
		return fmt.Errorf("content is required")
	}
	hasText := false
	hasImage := false
	for _, item := range req.Content {
		switch strings.TrimSpace(item.Type) {
		case videoContentText:
			if strings.TrimSpace(item.Text) != "" {
				hasText = true
			}
		case videoContentImageURL:
			if item.ImageURL == nil || strings.TrimSpace(item.ImageURL.URL) == "" {
				return fmt.Errorf("content[].image_url.url is required for type image_url")
			}
			if _, err := url.Parse(item.ImageURL.URL); err != nil {
				return fmt.Errorf("content[].image_url.url is invalid: %v", err)
			}
			hasImage = true
		default:
			return fmt.Errorf("unsupported content[].type: %q", item.Type)
		}
	}
	// A video task must carry either a text prompt (t2v) or a reference image (i2v);
	// the upstream API accepts both together (image + caption) as an i2v request.
	if !hasText && !hasImage {
		return fmt.Errorf("content[].text or content[].image_url is required")
	}
	switch req.Resolution {
	case "480p", "720p", "1080p":
	default:
		return fmt.Errorf("invalid resolution")
	}
	switch req.Ratio {
	case "16:9", "4:3", "1:1":
	default:
		return fmt.Errorf("invalid ratio")
	}
	if req.Duration <= 0 {
		return fmt.Errorf("duration must be greater than 0")
	}
	if req.FPS <= 0 {
		return fmt.Errorf("fps must be greater than 0")
	}
	return nil
}

func mapVideoTaskFromModel(task *model.VideoGenerationTask) *VideoTaskResponse {
	if task == nil {
		return nil
	}
	resp := &VideoTaskResponse{
		ID:              task.TaskId,
		Model:           task.Model,
		Status:          task.Status,
		CreatedAt:       task.CreatedTime,
		UpdatedAt:       task.UpdatedTime,
		Seed:            task.Seed,
		Resolution:      task.Resolution,
		Ratio:           task.Ratio,
		Duration:        task.Duration,
		FramesPerSecond: task.FramesPerSecond,
		Usage: &VideoTaskUsage{
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}
	if strings.TrimSpace(task.VideoURL) != "" {
		resp.Content = &struct {
			VideoURL string `json:"video_url"`
		}{VideoURL: task.VideoURL}
	}
	return resp
}

func parseUpstreamVideoTask(body []byte) *VideoTaskResponse {
	var resp VideoTaskResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	if resp.Status == "completed" {
		resp.Status = "succeeded"
	}
	return &resp
}

func buildVideoTasksURL(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(trimmed, "/contents/generations/tasks") || strings.HasSuffix(trimmed, "/tasks") {
		return u.String(), nil
	}
	u.Path = strings.TrimRight(trimmed, "/") + "/contents/generations/tasks"
	return u.String(), nil
}

func buildVideoTaskItemURL(baseURL string, taskID string) (string, error) {
	tasksURL, err := buildVideoTasksURL(baseURL)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(tasksURL, "/") + "/" + url.PathEscape(taskID), nil
}

func createVideoUpstreamRequest(method string, reqURL string, apiKey string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewBuffer(body)
	}
	req, err := http.NewRequest(method, reqURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	return req, nil
}

func selectVideoChannel(modelName string) (*model.Channel, error) {
	channel, err := model.GetRandomSatisfiedChannelByModelAndTypes(modelName, []int{channeltype.ArkVideo, channeltype.CustomVideo})
	if err == nil && channel != nil {
		return channel, nil
	}
	return nil, err
}

func CreateVideoGenerationTask(c *gin.Context) {
	ctx := c.Request.Context()
	userId := c.GetInt(ctxkey.Id)
	tokenId := c.GetInt(ctxkey.TokenId)

	var req CreateVideoTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	normalizeCreateVideoTaskRequest(&req)
	if err := validateCreateVideoTaskRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	channel, err := selectVideoChannel(req.Model)
	if err != nil || channel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "no available video channel"}})
		return
	}
	originModel := req.Model
	if mapped := channel.GetModelMapping(); mapped != nil {
		if upstreamModel, ok := mapped[req.Model]; ok && strings.TrimSpace(upstreamModel) != "" {
			req.Model = strings.TrimSpace(upstreamModel)
		}
	}

	baseURL := channel.GetBaseURL()
	upstreamURL, err := buildVideoTasksURL(baseURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid channel base_url"}})
		return
	}
	provider := "custom"
	if channel.Type == channeltype.ArkVideo {
		provider = "ark"
	}
	kind := detectVideoTaskKind(req.Content)
	logger.Infof(ctx, "[VideoTask] create route kind=%s provider=%s channel=%d model=%s upstream_model=%s", kind, provider, channel.Id, originModel, req.Model)

	bodyBytes, _ := json.Marshal(req)
	contentBytes, _ := json.Marshal(req.Content)
	httpReq, err := createVideoUpstreamRequest(http.MethodPost, upstreamURL, channel.Key, bodyBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		c.Data(httpResp.StatusCode, "application/json", respBody)
		return
	}

	upTask := parseUpstreamVideoTask(respBody)
	if upTask == nil || strings.TrimSpace(upTask.ID) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": "invalid upstream response"}})
		return
	}

	now := helper.GetTimestamp()
	task := &model.VideoGenerationTask{
		TaskId:           upTask.ID,
		ProviderTaskId:   upTask.ID,
		UserId:           userId,
		TokenId:          tokenId,
		ChannelId:        channel.Id,
		Model:            originModel,
		Status:           upTask.Status,
		ProviderStatus:   upTask.Status,
		Seed:             req.Seed,
		Resolution:       req.Resolution,
		Ratio:            req.Ratio,
		Duration:         req.Duration,
		FramesPerSecond:  req.FPS,
		RequestPayload:   string(bodyBytes),
		ResponsePayload:  string(respBody),
		ContentJSON:      string(contentBytes),
		PreConsumedQuota: 0,
		FinalQuota:       0,
		CreatedTime:      now,
		UpdatedTime:      now,
	}
	if err := model.CreateVideoTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":     task.TaskId,
		"status": task.Status,
	})
}

func GetVideoGenerationTask(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	taskID := c.Param("id")
	task, err := model.GetVideoTaskByTaskIdAndUser(taskID, userId)
	if err != nil {
		if model.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Task not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	channel, err := model.GetChannelById(task.ChannelId, true)
	if err == nil && channel != nil && channel.Status == model.ChannelStatusEnabled {
		itemURL, urlErr := buildVideoTaskItemURL(channel.GetBaseURL(), task.ProviderTaskId)
		if urlErr == nil {
			httpReq, reqErr := createVideoUpstreamRequest(http.MethodGet, itemURL, channel.Key, nil)
			if reqErr == nil {
				httpResp, doErr := http.DefaultClient.Do(httpReq)
				if doErr == nil {
					defer httpResp.Body.Close()
					body, _ := io.ReadAll(httpResp.Body)
					if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
						upTask := parseUpstreamVideoTask(body)
						if upTask != nil {
							updates := map[string]any{
								"status":            upTask.Status,
								"provider_status":   upTask.Status,
								"response_payload":  string(body),
								"updated_time":      helper.GetTimestamp(),
								"video_url":         "",
								"resolution":        upTask.Resolution,
								"ratio":             upTask.Ratio,
								"duration":          upTask.Duration,
								"frames_per_second": upTask.FramesPerSecond,
							}
							if upTask.Content != nil {
								updates["video_url"] = upTask.Content.VideoURL
							}
							if upTask.Status == model.VideoTaskStatusSucceeded || upTask.Status == model.VideoTaskStatusFailed {
								updates["finished_time"] = helper.GetTimestamp()
							}
							_ = model.UpdateVideoTaskByTaskId(task.TaskId, updates)
							if refreshed, getErr := model.GetVideoTaskByTaskIdAndUser(taskID, userId); getErr == nil && refreshed != nil {
								task = refreshed
								if task.Status == model.VideoTaskStatusSucceeded {
									settleVideoQuota(c.Request.Context(), task)
									task, _ = model.GetVideoTaskByTaskIdAndUser(taskID, userId)
								} else if task.Status == model.VideoTaskStatusFailed {
									settleVideoQuota(c.Request.Context(), task)
									task, _ = model.GetVideoTaskByTaskIdAndUser(taskID, userId)
								}
							}
						}
					}
				}
			}
		}
	}

	resp := mapVideoTaskFromModel(task)
	c.JSON(http.StatusOK, resp)
}

func ListVideoGenerationTasks(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	pageNum, _ := strconv.Atoi(c.DefaultQuery("page_num", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	tasks, total, err := model.ListVideoTasksByUser(userId, pageNum, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	respItems := make([]*VideoTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		respItems = append(respItems, mapVideoTaskFromModel(t))
	}
	c.JSON(http.StatusOK, gin.H{
		"page_num":  pageNum,
		"page_size": pageSize,
		"total":     total,
		"data":      respItems,
	})
}

func DeleteVideoGenerationTask(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	taskID := c.Param("id")
	task, err := model.GetVideoTaskByTaskIdAndUser(taskID, userId)
	if err != nil {
		if model.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Task not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if task.Status == model.VideoTaskStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Cannot cancel running request"}})
		return
	}
	settleVideoQuota(c.Request.Context(), task)

	channel, chErr := model.GetChannelById(task.ChannelId, true)
	if chErr == nil && channel != nil && channel.Status == model.ChannelStatusEnabled {
		itemURL, urlErr := buildVideoTaskItemURL(channel.GetBaseURL(), task.ProviderTaskId)
		if urlErr == nil {
			httpReq, reqErr := createVideoUpstreamRequest(http.MethodDelete, itemURL, channel.Key, nil)
			if reqErr == nil {
				httpResp, doErr := http.DefaultClient.Do(httpReq)
				if doErr == nil {
					defer httpResp.Body.Close()
					if httpResp.StatusCode >= 400 && httpResp.StatusCode != http.StatusNotFound {
						body, _ := io.ReadAll(httpResp.Body)
						c.Data(httpResp.StatusCode, "application/json", body)
						return
					}
				}
			}
		}
	}

	if err := model.MarkVideoTaskDeleted(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ProxyVideoGenerationTaskContent proxies generated video bytes through same-origin endpoint.
// This avoids browser mixed-content/CORS issues when upstream video_url is internal http.
func ProxyVideoGenerationTaskContent(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	taskID := c.Param("id")
	task, err := model.GetVideoTaskByTaskIdAndUser(taskID, userId)
	if err != nil {
		if model.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Task not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	videoURL := strings.TrimSpace(task.VideoURL)
	if videoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "video not ready"}})
		return
	}
	u, parseErr := url.Parse(videoURL)
	if parseErr != nil || (u.Scheme != "http" && u.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid video url"}})
		return
	}

	req, reqErr := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, videoURL, nil)
	if reqErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": reqErr.Error()}})
		return
	}
	if rg := c.GetHeader("Range"); rg != "" {
		req.Header.Set("Range", rg)
	}

	resp, doErr := http.DefaultClient.Do(req)
	if doErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": doErr.Error()}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	c.Header("Content-Type", contentType)
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		c.Header("Content-Length", cl)
	}
	if ar := resp.Header.Get("Accept-Ranges"); ar != "" {
		c.Header("Accept-Ranges", ar)
	}
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		c.Header("Content-Range", cr)
	}
	c.Header("Cache-Control", "private, max-age=300")

	if c.Query("download") == "1" {
		ext := path.Ext(u.Path)
		if ext == "" {
			exts, _ := mime.ExtensionsByType(contentType)
			if len(exts) > 0 {
				ext = exts[0]
			} else {
				ext = ".mp4"
			}
		}
		filename := fmt.Sprintf("%s%s", task.TaskId, ext)
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	}

	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
