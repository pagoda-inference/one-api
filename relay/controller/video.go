package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	"github.com/pagoda-inference/one-api/relay/channeltype"
	"github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
	"github.com/pagoda-inference/one-api/relay/relaymode"
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
	// UpstreamExtra preserves any unknown fields the upstream returned
	// (progress, seconds, quality, file_name, error, media_type, etc.) so the
	// retrieve/list handlers can reserialize and return the upstream envelope
	// verbatim rather than dropping fields the client expects.
	UpstreamExtra map[string]json.RawMessage `json:"-"`
}

// ---- Sync defaults for /v1/videos/sync polling ----

const (
	videoSyncDefaultTimeoutSeconds = 300
	videoSyncMinPollInterval       = 3
	videoSyncDefaultPollInterval   = 5
)

// OpenAIVideoCreateRequest is the OpenAI-compatible create request accepted by
// POST /v1/videos and POST /v1/videos/sync. It supports both a compact form
// (model + prompt / model + image_url) and a full multipart form (content[]).
// The vLLM-OMNI extension fields (width/height/num_frames/guidance_scale/...)
// are accepted on the JSON path and forwarded to upstreams that understand them;
// they are ignored by the Ark content[] upstream.
type OpenAIVideoCreateRequest struct {
	Model      string             `json:"model"`
	Prompt     string             `json:"prompt,omitempty"`
	ImageURL   string             `json:"image_url,omitempty"`
	Content    []VideoContentItem `json:"content,omitempty"`
	Ratio      string             `json:"ratio,omitempty"`
	Resolution string             `json:"resolution,omitempty"`
	Duration   int                `json:"duration,omitempty"`
	FPS        int                `json:"fps,omitempty"`
	Seed       int                `json:"seed,omitempty"`
	// Timeout is the maximum number of seconds the sync endpoint will poll the
	// upstream before returning a 408 timeout. Only honored by /v1/videos/sync.
	Timeout int `json:"timeout,omitempty"`
	// PollInterval is the seconds between upstream polls in sync mode.
	PollInterval int `json:"poll_interval,omitempty"`

	// ---- vLLM-OMNI extension fields (JSON path) ----
	// Output dimensions and frame controls.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// Seconds overrides Duration when set (vLLM-OMNI uses "seconds").
	Seconds           float64 `json:"seconds,omitempty"`
	NumFrames         int     `json:"num_frames,omitempty"`
	NumInferenceSteps int     `json:"num_inference_steps,omitempty"`
	// Diffusion guidance controls.
	GuidanceScale  *float64 `json:"guidance_scale,omitempty"`
	GuidanceScale2 *float64 `json:"guidance_scale_2,omitempty"`
	BoundaryRatio  *float64 `json:"boundary_ratio,omitempty"`
	FlowShift      *float64 `json:"flow_shift,omitempty"`
	TrueCFGScale   *float64 `json:"true_cfg_scale,omitempty"`
	// Reference media for i2v/v2v/s2v (JSON form; mirrors vLLM-OMNI multipart fields).
	ImageReference string `json:"image_reference,omitempty"`
	VideoReference string `json:"video_reference,omitempty"`
	AudioReference string `json:"audio_reference,omitempty"`
	// Audio generation controls.
	GenerateSound bool     `json:"generate_sound,omitempty"`
	SoundDuration *float64 `json:"sound_duration,omitempty"`
	// Negative prompt and extras.
	NegativePrompt string `json:"negative_prompt,omitempty"`
	User           string `json:"user,omitempty"`
	Lora           string `json:"lora,omitempty"`
	ExtraParams    string `json:"extra_params,omitempty"`
	// Frame interpolation.
	EnableFrameInterpolation    bool     `json:"enable_frame_interpolation,omitempty"`
	FrameInterpolationExp       int      `json:"frame_interpolation_exp,omitempty"`
	FrameInterpolationScale     *float64 `json:"frame_interpolation_scale,omitempty"`
	FrameInterpolationModelPath string   `json:"frame_interpolation_model_path,omitempty"`
}

// toCreateVideoTaskRequest normalizes an OpenAIVideoCreateRequest into the
// internal CreateVideoTaskRequest shape that the upstream Ark API expects.
// When content[] is empty, it is synthesized from prompt / image_url so the
// compact OpenAI-style form transparently maps to the multipart upstream form.
func (r *OpenAIVideoCreateRequest) toCreateVideoTaskRequest() *CreateVideoTaskRequest {
	req := &CreateVideoTaskRequest{
		Model:      r.Model,
		Content:    r.Content,
		Ratio:      r.Ratio,
		Resolution: r.Resolution,
		Duration:   r.Duration,
		FPS:        r.FPS,
		Seed:       r.Seed,
	}
	if len(req.Content) == 0 {
		if strings.TrimSpace(r.ImageURL) != "" {
			req.Content = append(req.Content, VideoContentItem{
				Type:     videoContentImageURL,
				ImageURL: &VideoContentItemImageURL{URL: strings.TrimSpace(r.ImageURL)},
			})
		}
		if strings.TrimSpace(r.Prompt) != "" {
			req.Content = append(req.Content, VideoContentItem{
				Type: videoContentText,
				Text: strings.TrimSpace(r.Prompt),
			})
		}
	}
	return req
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

// mapVideoTaskFromModel maps a DB task into the legacy Ark-style response
// shape (used by the /api/v1/contents/generations/tasks endpoints).
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

// openAIVideoResponseObject is the OpenAI-compatible video object returned by
// the /v1/videos endpoints. The `Video` field carries the generated video URL
// once the task has succeeded.
type openAIVideoResponseObject struct {
	ID              string           `json:"id"`
	Object          string           `json:"object"`
	Model           string           `json:"model"`
	Status          string           `json:"status"`
	CreatedAt       int64            `json:"created_at"`
	UpdatedAt       int64            `json:"updated_at,omitempty"`
	Seed            int              `json:"seed,omitempty"`
	Resolution      string           `json:"resolution,omitempty"`
	Ratio           string           `json:"ratio,omitempty"`
	Duration        int              `json:"duration,omitempty"`
	FramesPerSecond int              `json:"frames_per_second,omitempty"`
	Video           *openAIVideoFile `json:"video,omitempty"`
	Usage           *VideoTaskUsage  `json:"usage,omitempty"`
}

type openAIVideoFile struct {
	URL string `json:"url"`
}

// mapVideoTaskToOpenAIResponse maps a DB task into the OpenAI-compatible
// response object used by the /v1/videos endpoints.
func mapVideoTaskToOpenAIResponse(task *model.VideoGenerationTask) *openAIVideoResponseObject {
	if task == nil {
		return nil
	}
	resp := &openAIVideoResponseObject{
		ID:              task.TaskId,
		Object:          "video",
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
		resp.Video = &openAIVideoFile{URL: task.VideoURL}
	}
	return resp
}

// parseUpstreamVideoTask parses an upstream video task JSON envelope and
// normalizes the status field ("completed" -> "succeeded"). The full parsed
// body is returned so callers can pass it through to the client without
// losing fields (progress, seconds, quality, file_name, etc.) the upstream
// may include. Any field not in VideoTaskResponse is preserved on the JSON
// envelope through UpstreamExtra, allowing passthrough forwarding.
func parseUpstreamVideoTask(body []byte) *VideoTaskResponse {
	var resp VideoTaskResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	if resp.Status == "completed" {
		resp.Status = "succeeded"
	}
	// Capture unknown fields so we can reserialize the upstream response
	// verbatim after status normalization.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		known := map[string]struct{}{
			"id": {}, "model": {}, "status": {}, "created_at": {},
			"updated_at": {}, "seed": {}, "resolution": {}, "ratio": {},
			"duration": {}, "framespersecond": {},
			"content": {}, "usage": {},
		}
		extras := make(map[string]json.RawMessage)
		for k, v := range raw {
			if _, ok := known[k]; ok {
				continue
			}
			extras[k] = v
		}
		if len(extras) > 0 {
			resp.UpstreamExtra = extras
		}
	}
	return &resp
}

// upstreamVideoURL returns the URL of the generated video as reported by the
// upstream, checking the fields the various providers use: content.video_url
// (Ark-style), file_name / url / video_url (top-level on vLLM-OMNI and
// similar). The first non-empty value wins.
func upstreamVideoURL(r *VideoTaskResponse) string {
	if r == nil {
		return ""
	}
	if r.Content != nil && strings.TrimSpace(r.Content.VideoURL) != "" {
		return strings.TrimSpace(r.Content.VideoURL)
	}
	if v, ok := r.UpstreamExtra["video_url"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	if v, ok := r.UpstreamExtra["url"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	if v, ok := r.UpstreamExtra["file_name"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// forwardUpstreamTask serializes a parsed upstream task back to JSON for the
// client. Status normalization ("completed"->"succeeded") is already applied
// by parseUpstreamVideoTask. The base fields are emitted first so they win
// over any keys of the same name in UpstreamExtra. We also add `updated_at`
// from the task's local updated_time so the client gets a stable, monotonic
// value even when the upstream doesn't refresh it.
//
// originModel overrides the upstream-returned "model" field with the
// user-facing model name (so clients see "bedi/wan2.2-i2v-a14b" rather than
// the upstream's internal path like "/data/weight/Wan2.2-I2V-A14B-Diffusers").
func forwardUpstreamTask(c *gin.Context, upTask *VideoTaskResponse, updatedAt int64, originModel string) error {
	out := make(map[string]json.RawMessage, len(upTask.UpstreamExtra)+8)
	for k, v := range upTask.UpstreamExtra {
		if k == "model" {
			continue // override below
		}
		out[k] = v
	}
	if upTask.ID != "" {
		b, _ := json.Marshal(upTask.ID)
		out["id"] = b
	}
	if upTask.Model != "" {
		b, _ := json.Marshal(upTask.Model)
		out["model"] = b
	}
	if originModel != "" {
		b, _ := json.Marshal(originModel)
		out["model"] = b
	}
	if upTask.Status != "" {
		b, _ := json.Marshal(upTask.Status)
		out["status"] = b
	}
	if upTask.CreatedAt != 0 {
		b, _ := json.Marshal(upTask.CreatedAt)
		out["created_at"] = b
	}
	if updatedAt != 0 {
		b, _ := json.Marshal(updatedAt)
		out["updated_at"] = b
	}
	// Always include a `usage` zero-placeholder so clients can rely on the
	// field being present, even when the upstream does not report it.
	usage := upTask.Usage
	if usage == nil {
		usage = &VideoTaskUsage{}
	}
	b, _ := json.Marshal(usage)
	out["usage"] = b
	// For Ark-style upstreams the content block is the natural place for the
	// video URL; preserve it.
	if upTask.Content != nil {
		b, _ := json.Marshal(upTask.Content)
		out["content"] = b
	}
	c.JSON(http.StatusOK, out)
	return nil
}

// forwardUpstreamTaskJSON serializes a parsed upstream task to a raw JSON
// object (rather than writing through gin.Context), applying the same
// overrides as forwardUpstreamTask. Used by the list endpoint, which emits
// multiple tasks into a single response and so cannot call c.JSON per item.
func forwardUpstreamTaskJSON(upTask *VideoTaskResponse, updatedAt int64, originModel string) (json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(upTask.UpstreamExtra)+8)
	for k, v := range upTask.UpstreamExtra {
		if k == "model" {
			continue
		}
		out[k] = v
	}
	if upTask.ID != "" {
		b, _ := json.Marshal(upTask.ID)
		out["id"] = b
	}
	if upTask.Model != "" {
		b, _ := json.Marshal(upTask.Model)
		out["model"] = b
	}
	if originModel != "" {
		b, _ := json.Marshal(originModel)
		out["model"] = b
	}
	if upTask.Status != "" {
		b, _ := json.Marshal(upTask.Status)
		out["status"] = b
	}
	if upTask.CreatedAt != 0 {
		b, _ := json.Marshal(upTask.CreatedAt)
		out["created_at"] = b
	}
	if updatedAt != 0 {
		b, _ := json.Marshal(updatedAt)
		out["updated_at"] = b
	}
	usage := upTask.Usage
	if usage == nil {
		usage = &VideoTaskUsage{}
	}
	b, _ := json.Marshal(usage)
	out["usage"] = b
	if upTask.Content != nil {
		b, _ := json.Marshal(upTask.Content)
		out["content"] = b
	}
	return json.Marshal(out)
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

// buildVideoCreateUpstreamURL resolves the upstream URL for a video create
// request (POST /v1/videos or POST /v1/videos/sync), honoring the channel type:
//
//   - OpenAICompatible channels forward the original request path verbatim
//     (e.g. /v1/videos/sync -> <baseURL>/videos/sync), exactly like the
//     chat relay does via GetFullRequestURL. This serves vLLM-OMNI upstreams
//     which expose /videos and /videos/sync. Only engaged when requestPath is
//     non-empty (i.e. the OpenAI /v1/videos route); the legacy Ark route
//     passes an empty requestPath to keep the /contents/generations/tasks path.
//   - ArkVideo / CustomVideo channels keep the Ark /contents/generations/tasks
//     collection path (buildVideoTasksURL); sync appends /sync.
func buildVideoCreateUpstreamURL(channel *model.Channel, requestPath string, relayMode int) (string, error) {
	if channel.Type == channeltype.OpenAICompatible && strings.TrimSpace(requestPath) != "" {
		// Strip any query string, then forward via the OpenAI path convention.
		path := strings.SplitN(requestPath, "?", 2)[0]
		return openai.GetFullRequestURL(channel.GetBaseURL(), path, channeltype.OpenAICompatible), nil
	}
	upstreamURL, err := buildVideoTasksURL(channel.GetBaseURL())
	if err != nil {
		return "", err
	}
	if relayMode == relaymode.VideoSync {
		upstreamURL = strings.TrimRight(upstreamURL, "/") + "/sync"
	}
	return upstreamURL, nil
}

// buildVideoItemUpstreamURL resolves the upstream URL for a single-task
// operation (GET/DELETE /v1/videos/:id), honoring the channel type:
//
//   - OpenAICompatible channels forward /v1/videos/<id> verbatim (vLLM-OMNI).
//   - ArkVideo / CustomVideo channels keep the Ark /contents/generations/tasks/<id> path.
func buildVideoItemUpstreamURL(channel *model.Channel, taskID string) (string, error) {
	escapedID := url.PathEscape(taskID)
	if channel.Type == channeltype.OpenAICompatible {
		return openai.GetFullRequestURL(channel.GetBaseURL(), "/v1/videos/"+escapedID, channeltype.OpenAICompatible), nil
	}
	return buildVideoTaskItemURL(channel.GetBaseURL(), taskID)
}

// rewriteMultipartModel rewrites the "model" form field in a multipart/form-data
// body to the mapped upstream model name, preserving all other fields and file
// parts (input_reference, etc.). It returns the rebuilt body and its
// Content-Type (with a fresh boundary). If the original model equals the
// mapped model, or parsing fails, the original body/Content-Type is returned
// unchanged so behavior is never degraded.
//
// This is required for the OpenAI-compatible channel path: the client sends a
// public model name (e.g. "bedi/wan2.2-i2v-a14b") but the upstream vLLM-OMNI
// server expects its internal path (e.g.
// "/data/weight/Wan2.2-I2V-A14B-Diffusers"). Channel ModelMapping carries that
// translation, so we must rewrite the field inside the opaque multipart stream
// before forwarding — we cannot simply forward the bytes verbatim.
func rewriteMultipartModel(originalBody []byte, originalContentType, mappedModel string) (body []byte, contentType string, ok bool) {
	mappedModel = strings.TrimSpace(mappedModel)
	if mappedModel == "" {
		return originalBody, originalContentType, false
	}
	mediaType, params, err := mime.ParseMediaType(originalContentType)
	if err != nil || mediaType != "multipart/form-data" {
		return originalBody, originalContentType, false
	}
	boundary, hasBoundary := params["boundary"]
	if !hasBoundary {
		return originalBody, originalContentType, false
	}
	// Parse the original multipart, rewriting the "model" field.
	mr := multipart.NewReader(bytes.NewReader(originalBody), boundary)
	var out bytes.Buffer
	rw := multipart.NewWriter(&out)
	rewrote := false
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Malformed multipart — bail out and forward the original bytes.
			return originalBody, originalContentType, false
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return originalBody, originalContentType, false
		}
		fieldName := part.FormName()
		if fieldName == "model" {
			if string(data) != mappedModel {
				data = []byte(mappedModel)
			}
			rewrote = true
		}
		// Preserve file vs. plain-field distinction.
		var wp io.Writer
		if part.FileName() != "" {
			wp, err = rw.CreateFormFile(fieldName, part.FileName())
		} else {
			wp, err = rw.CreateFormField(fieldName)
		}
		if err != nil {
			return originalBody, originalContentType, false
		}
		if _, err := wp.Write(data); err != nil {
			return originalBody, originalContentType, false
		}
	}
	if err := rw.Close(); err != nil {
		return originalBody, originalContentType, false
	}
	return out.Bytes(), rw.FormDataContentType(), rewrote
}

// summarizeMultipartRequest extracts a UTF-8-safe JSON metadata summary of a
// multipart/form-data video request: field values and file part names only
// (no file bytes). The raw multipart body may contain binary content (e.g. a
// PNG reference image starting with 0x89), which cannot be stored in a text
// column ("invalid byte sequence for encoding UTF8"). This summary is what we
// persist as RequestPayload for multipart requests. It always returns valid
// JSON; on any parse error it returns a small placeholder object.
func summarizeMultipartRequest(body []byte, contentType string) string {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" {
		return `{"content_type":"` + contentType + `","note":"non-multipart"}`
	}
	boundary, ok := params["boundary"]
	if !ok {
		return `{"note":"missing multipart boundary"}`
	}
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	fields := make(map[string]string)
	var files []map[string]string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return `{"note":"multipart parse error: ` + err.Error() + `"}`
		}
		name := part.FormName()
		if part.FileName() != "" {
			files = append(files, map[string]string{"field": name, "filename": part.FileName()})
			// Do not read file bytes — they may be binary.
			continue
		}
		data, err := io.ReadAll(io.LimitReader(part, 1<<20)) // 1MiB cap per field
		if err != nil {
			continue
		}
		fields[name] = string(data)
	}
	summary := map[string]any{
		"content_type": "multipart/form-data",
		"fields":       fields,
	}
	if len(files) > 0 {
		summary["files"] = files
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return `{"note":"summary marshal failed"}`
	}
	return string(b)
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

// videoChannelFromContext resolves the channel to use for a /v1/videos request.
// Under the relay router, Distribute() has already selected a channel for the
// model and populated the context; we build a lightweight *model.Channel view
// from the context values so the core create logic stays channel-source-agnostic.
// Model mapping is applied by the caller before createVideoTaskCore, so it is
// intentionally not reconstructed here.
func videoChannelFromContext(c *gin.Context) *model.Channel {
	channelId := c.GetInt(ctxkey.ChannelId)
	if channelId == 0 {
		return nil
	}
	ch := &model.Channel{Id: channelId}
	ch.Type = c.GetInt(ctxkey.Channel)
	ch.Key = c.GetString("channel_api_key")
	ch.Name = c.GetString(ctxkey.ChannelName)
	if baseURL := c.GetString(ctxkey.BaseURL); baseURL != "" {
		ch.BaseURL = &baseURL
	}
	return ch
}

func selectVideoChannel(modelName string) (*model.Channel, error) {
	channel, err := model.GetRandomSatisfiedChannelByModelAndTypes(modelName, []int{channeltype.ArkVideo, channeltype.CustomVideo, channeltype.OpenAICompatible})
	if err == nil && channel != nil {
		return channel, nil
	}
	return nil, err
}

// ---- Reusable cores (shared by legacy Ark handlers and OpenAI handlers) ----

// createVideoTaskCore submits the create request to the upstream and persists
// the resulting task. The channel is supplied by the caller (from Distribute
// for /v1/videos, or from selectVideoChannel for the legacy route), so the
// core stays agnostic to how the channel was chosen. It returns the persisted
// task, the raw upstream response body, the upstream HTTP status, and an
// OpenAI-style error envelope when the upstream call fails.
func createVideoTaskCore(ctx context.Context, userId, tokenId int, channel *model.Channel, originModel string, req *CreateVideoTaskRequest, requestPath string, relayMode int) (*model.VideoGenerationTask, []byte, int, *relaymodel.ErrorWithStatusCode) {
	if channel == nil {
		return nil, nil, 0, videoErrorf(http.StatusServiceUnavailable, "no_available_channel", "no available video channel")
	}
	upstreamURL, err := buildVideoCreateUpstreamURL(channel, requestPath, relayMode)
	if err != nil {
		return nil, nil, 0, videoErrorf(http.StatusBadRequest, "invalid_channel_base_url", "invalid channel base_url")
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
		return nil, nil, 0, videoErrorf(http.StatusInternalServerError, "build_upstream_request_failed", err.Error())
	}
	httpResp, err := client.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, nil, 0, videoErrorf(http.StatusBadGateway, "upstream_request_failed", err.Error())
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		// Surface the upstream status so /v1/videos can retry on 429/5xx via controller.Relay.
		return nil, respBody, httpResp.StatusCode, videoErrorFromUpstream(httpResp.StatusCode, respBody)
	}

	upTask := parseUpstreamVideoTask(respBody)
	if upTask == nil || strings.TrimSpace(upTask.ID) == "" {
		return nil, respBody, httpResp.StatusCode, videoErrorf(http.StatusBadGateway, "invalid_upstream_response", "invalid upstream response")
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
		return nil, respBody, httpResp.StatusCode, videoErrorf(http.StatusInternalServerError, "persist_task_failed", err.Error())
	}
	return task, respBody, httpResp.StatusCode, nil
}

// refreshVideoTaskFromUpstream fetches the latest task state from the upstream
// provider (when its channel is enabled), persists the update, and settles
// quota once the task reaches a terminal state. Returns the refreshed task.
func refreshVideoTaskFromUpstream(ctx context.Context, task *model.VideoGenerationTask) (*model.VideoGenerationTask, *relaymodel.ErrorWithStatusCode) {
	if task == nil {
		return nil, videoErrorf(http.StatusInternalServerError, "nil_task", "nil task")
	}
	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil || channel == nil || channel.Status != model.ChannelStatusEnabled {
		return task, nil
	}
	itemURL, urlErr := buildVideoItemUpstreamURL(channel, task.ProviderTaskId)
	if urlErr != nil {
		return task, nil
	}
	httpReq, reqErr := createVideoUpstreamRequest(http.MethodGet, itemURL, channel.Key, nil)
	if reqErr != nil {
		return task, nil
	}
	httpResp, doErr := client.GetHTTPClient().Do(httpReq)
	if doErr != nil {
		return task, nil
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return task, nil
	}
	upTask := parseUpstreamVideoTask(body)
	if upTask == nil {
		return task, nil
	}
	// Discover the video file URL from any of the upstream's known field names
	// (content.video_url, top-level video_url / url / file_name). This makes
	// the /v1/videos/:id/content proxy work for both Ark-style and
	// vLLM-OMNI-style responses.
	videoURL := upstreamVideoURL(upTask)
	updates := map[string]any{
		"status":            upTask.Status,
		"provider_status":   upTask.Status,
		"response_payload":  string(body),
		"updated_time":      helper.GetTimestamp(),
		"video_url":         videoURL,
		"resolution":        upTask.Resolution,
		"ratio":             upTask.Ratio,
		"duration":          upTask.Duration,
		"frames_per_second": upTask.FramesPerSecond,
	}
	if upTask.Status == model.VideoTaskStatusSucceeded || upTask.Status == model.VideoTaskStatusFailed {
		updates["finished_time"] = helper.GetTimestamp()
	}
	_ = model.UpdateVideoTaskByTaskId(task.TaskId, updates)
	refreshed, getErr := model.GetVideoTaskByTaskIdAndUser(task.TaskId, task.UserId)
	if getErr != nil || refreshed == nil {
		return task, nil
	}
	task = refreshed
	if task.Status == model.VideoTaskStatusSucceeded || task.Status == model.VideoTaskStatusFailed {
		settleVideoQuota(ctx, task)
		refreshed, _ = model.GetVideoTaskByTaskIdAndUser(task.TaskId, task.UserId)
		if refreshed != nil {
			task = refreshed
		}
	}
	return task, nil
}

func getVideoTaskCore(ctx context.Context, userId int, taskID string) (*model.VideoGenerationTask, *relaymodel.ErrorWithStatusCode) {
	task, err := model.GetVideoTaskByTaskIdAndUser(taskID, userId)
	if err != nil {
		if model.IsNotFound(err) {
			return nil, videoErrorf(http.StatusNotFound, "task_not_found", "Task not found")
		}
		return nil, videoErrorf(http.StatusInternalServerError, "get_task_failed", err.Error())
	}
	return refreshVideoTaskFromUpstream(ctx, task)
}

func deleteVideoTaskCore(ctx context.Context, userId int, taskID string) (*model.VideoGenerationTask, *relaymodel.ErrorWithStatusCode) {
	task, err := model.GetVideoTaskByTaskIdAndUser(taskID, userId)
	if err != nil {
		if model.IsNotFound(err) {
			return nil, videoErrorf(http.StatusNotFound, "task_not_found", "Task not found")
		}
		return nil, videoErrorf(http.StatusInternalServerError, "get_task_failed", err.Error())
	}
	if task.Status == model.VideoTaskStatusRunning {
		return nil, videoErrorf(http.StatusBadRequest, "cannot_cancel_running", "Cannot cancel running request")
	}
	settleVideoQuota(ctx, task)

	channel, chErr := model.GetChannelById(task.ChannelId, true)
	if chErr == nil && channel != nil && channel.Status == model.ChannelStatusEnabled {
		itemURL, urlErr := buildVideoItemUpstreamURL(channel, task.ProviderTaskId)
		if urlErr == nil {
			httpReq, reqErr := createVideoUpstreamRequest(http.MethodDelete, itemURL, channel.Key, nil)
			if reqErr == nil {
				httpResp, doErr := client.GetHTTPClient().Do(httpReq)
				if doErr == nil {
					defer httpResp.Body.Close()
					if httpResp.StatusCode >= 400 && httpResp.StatusCode != http.StatusNotFound {
						body, _ := io.ReadAll(httpResp.Body)
						return nil, videoErrorFromUpstream(httpResp.StatusCode, body)
					}
				}
			}
		}
	}

	if err := model.MarkVideoTaskDeleted(taskID); err != nil {
		return nil, videoErrorf(http.StatusInternalServerError, "delete_task_failed", err.Error())
	}
	return task, nil
}

func listVideoTasksCore(ctx context.Context, userId, pageNum, pageSize int) ([]*model.VideoGenerationTask, int64, *relaymodel.ErrorWithStatusCode) {
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	tasks, total, err := model.ListVideoTasksByUser(userId, pageNum, pageSize)
	if err != nil {
		return nil, 0, videoErrorf(http.StatusInternalServerError, "list_tasks_failed", err.Error())
	}
	return tasks, total, nil
}

// ---- Legacy Ark-style handlers (/api/v1/contents/generations/tasks) ----
// Thin wrappers over the shared cores; response shapes are unchanged for
// backward compatibility with existing clients.

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

	task, respBody, httpStatus, bizErr := createVideoTaskCore(ctx, userId, tokenId, channel, originModel, &req, "", relaymode.Video)
	if bizErr != nil {
		// Preserve the legacy behavior: surface the raw upstream body for upstream
		// errors, otherwise the structured error message.
		if httpStatus != 0 && (httpStatus < 200 || httpStatus >= 300) && respBody != nil {
			c.Data(httpStatus, "application/json", respBody)
			return
		}
		c.JSON(bizErr.StatusCode, gin.H{"error": gin.H{"message": bizErr.Error.Message}})
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
	task, bizErr := getVideoTaskCore(c.Request.Context(), userId, taskID)
	if bizErr != nil {
		c.JSON(bizErr.StatusCode, gin.H{"error": gin.H{"message": bizErr.Error.Message}})
		return
	}
	resp := mapVideoTaskFromModel(task)
	c.JSON(http.StatusOK, resp)
}

func ListVideoGenerationTasks(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	pageNum, _ := strconv.Atoi(c.DefaultQuery("page_num", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	tasks, total, bizErr := listVideoTasksCore(c.Request.Context(), userId, pageNum, pageSize)
	if bizErr != nil {
		c.JSON(bizErr.StatusCode, gin.H{"error": gin.H{"message": bizErr.Error.Message}})
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
	_, bizErr := deleteVideoTaskCore(c.Request.Context(), userId, taskID)
	if bizErr != nil {
		c.JSON(bizErr.StatusCode, gin.H{"error": gin.H{"message": bizErr.Error.Message}})
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

	resp, doErr := client.GetHTTPClient().Do(req)
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

// RelayVideoContent handles GET /v1/videos/:id/content. It is an OpenAI-style
// alias for ProxyVideoGenerationTaskContent that streams the generated video
// bytes through the same origin (supporting Range and ?download=1) to avoid
// browser mixed-content/CORS issues.
func RelayVideoContent(c *gin.Context) {
	ProxyVideoGenerationTaskContent(c)
}

// isMultipartVideoRequest reports whether the incoming request uses
// multipart/form-data, the content type required by the vLLM-OMNI video API
// (which carries file uploads like input_reference alongside form fields).
func isMultipartVideoRequest(c *gin.Context) bool {
	return strings.HasPrefix(c.ContentType(), "multipart/form-data")
}

// videoModelFromMultipart returns the "model" form field from a multipart
// request. The body and MultipartForm are already parsed/cached earlier in
// the middleware chain (UnmarshalBodyReusable in getRequestModel buffers the
// raw bytes into ctxkey.KeyRequestBody and populates MultipartForm), so this
// never re-reads the network stream. The original multipart bytes are still
// available for the later raw passthrough.
func videoModelFromMultipart(c *gin.Context) string {
	return strings.TrimSpace(c.PostForm("model"))
}

// ---- OpenAI-compatible /v1/videos handlers ----

// relayVideoMultipart handles multipart/form-data create requests (the
// vLLM-OMNI wire format). It forwards the original multipart body verbatim to
// the upstream video API, preserving file parts (input_reference) and form
// fields, then either returns the async job record (POST /v1/videos) or streams
// the raw video bytes back (POST /v1/videos/sync). This mirrors the audio
// transcription passthrough pattern.
func relayVideoMultipart(c *gin.Context, m *meta.Meta, relayMode int) *relaymodel.ErrorWithStatusCode {
	channel := videoChannelFromContext(c)
	if channel == nil {
		return videoErrorf(http.StatusServiceUnavailable, "no_available_channel", "no available video channel")
	}
	upstreamURL, err := buildVideoCreateUpstreamURL(channel, m.RequestURLPath, relayMode)
	if err != nil {
		return videoErrorf(http.StatusBadRequest, "invalid_channel_base_url", "invalid channel base_url")
	}

	// The original multipart body was buffered into ctxkey.KeyRequestBody by
	// UnmarshalBodyReusable during getRequestModel, so it is retry-safe.
	bodyBytes, err := common.GetRequestBody(c)
	if err != nil {
		return videoErrorf(http.StatusBadRequest, "read_body_failed", "failed to read request body: %v", err)
	}
	// Reset the body so controller.Relay can retry with a fresh channel.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Apply channel ModelMapping: the client sends a public model name (e.g.
	// "bedi/wan2.2-i2v-a14b") but the upstream vLLM-OMNI server expects its
	// internal path (e.g. "/data/weight/Wan2.2-I2V-A14B-Diffusers"). Rewrite
	// the "model" form field inside the buffered multipart stream before
	// forwarding. JSON path uses req.Model + mapping; multipart path requires
	// this byte-level rewrite because we never re-serialize the body.
	clientContentType := c.GetHeader("Content-Type")
	if mapped, ok := m.ModelMapping[m.OriginModelName]; ok && strings.TrimSpace(mapped) != "" {
		if rewrittenBody, rewrittenCT, didRewrite := rewriteMultipartModel(bodyBytes, clientContentType, mapped); didRewrite {
			bodyBytes = rewrittenBody
			clientContentType = rewrittenCT
		}
	}

	httpReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return videoErrorf(http.StatusInternalServerError, "build_upstream_request_failed", err.Error())
	}
	httpReq.Header.Set("Content-Type", clientContentType)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	if v := c.GetHeader("Accept"); v != "" {
		httpReq.Header.Set("Accept", v)
	}

	httpResp, err := client.GetHTTPClient().Do(httpReq)
	if err != nil {
		return videoErrorf(http.StatusBadGateway, "upstream_request_failed", err.Error())
	}
	defer httpResp.Body.Close()

	// Sync: the upstream returns raw video bytes directly. Stream them back.
	if relayMode == relaymode.VideoSync {
		contentType := httpResp.Header.Get("Content-Type")
		// vLLM-OMNI sync must return either a video/* / application/octet-stream
		// body (success) or a JSON error envelope. Distinguish the two by
		// status + content-type without buffering the entire body: peek at the
		// first bytes to detect a JSON '[' or '{' prefix.
		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			errBody, _ := io.ReadAll(httpResp.Body)
			return videoErrorFromUpstream(httpResp.StatusCode, errBody)
		}
		if contentType == "" {
			contentType = "video/mp4"
		}
		isVideoCT := strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "application/octet-stream")
		if !isVideoCT {
			// Upstream returned JSON (error envelope or unexpected shape).
			// Forward verbatim with the upstream's status and content type.
			body, _ := io.ReadAll(httpResp.Body)
			c.Data(httpResp.StatusCode, contentType, body)
			return nil
		}
		c.Header("Content-Type", contentType)
		if cl := httpResp.Header.Get("Content-Length"); cl != "" {
			c.Header("Content-Length", cl)
		}
		c.Status(httpResp.StatusCode)
		_, _ = io.Copy(c.Writer, httpResp.Body)
		return nil
	}

	// Async: the upstream returns a JSON job record. Forward it and persist.
	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return videoErrorFromUpstream(httpResp.StatusCode, respBody)
	}
	upTask := parseUpstreamVideoTask(respBody)
	if upTask == nil || strings.TrimSpace(upTask.ID) == "" {
		// The upstream may return a non-standard envelope; forward it verbatim.
		c.Data(httpResp.StatusCode, "application/json", respBody)
		return nil
	}
	now := helper.GetTimestamp()
	task := &model.VideoGenerationTask{
		TaskId:          upTask.ID,
		ProviderTaskId:  upTask.ID,
		UserId:          m.UserId,
		TokenId:         m.TokenId,
		ChannelId:       channel.Id,
		Model:           m.OriginModelName,
		Status:          upTask.Status,
		ProviderStatus:  upTask.Status,
		Seed:            upTask.Seed,
		Resolution:      upTask.Resolution,
		Ratio:           upTask.Ratio,
		Duration:        upTask.Duration,
		FramesPerSecond: upTask.FramesPerSecond,
		// Persist a UTF-8-safe summary of the multipart body, not the raw bytes:
		// the body may contain binary file content (e.g. PNG 0x89) that
		// PostgreSQL text columns reject with "invalid byte sequence for
		// encoding UTF8". The JSON path stores its body directly, which is safe.
		RequestPayload:   summarizeMultipartRequest(bodyBytes, clientContentType),
		ResponsePayload:  string(respBody),
		PreConsumedQuota: 0,
		FinalQuota:       0,
		CreatedTime:      now,
		UpdatedTime:      now,
	}
	if err := model.CreateVideoTask(task); err != nil {
		return videoErrorf(http.StatusInternalServerError, "persist_task_failed", err.Error())
	}
	// Forward the upstream create response verbatim (status normalization
	// already applied), so fields the upstream includes (size, progress,
	// seconds, quality, ...) are visible to the client immediately. The
	// "model" field is overridden with the user-facing model name (origin
	// model, not the upstream internal name).
	if parsed := parseUpstreamVideoTask(respBody); parsed != nil {
		_ = forwardUpstreamTask(c, parsed, now, m.OriginModelName)
		return nil
	}
	// Upstream returned a non-standard envelope; fall back to a minimal one.
	c.JSON(http.StatusAccepted, gin.H{
		"id":         task.TaskId,
		"object":     "video",
		"model":      task.Model,
		"status":     task.Status,
		"created_at": task.CreatedTime,
	})
	return nil
}

// RelayVideoHelper handles POST /v1/videos (async) and POST /v1/videos/sync.
// It runs under controller.Relay, so the request body is buffered (retry-safe)
// and transient upstream failures (429/5xx) are automatically retried on a new
// channel. The channel is supplied by the Distribute middleware, which selects
// any enabled channel exposing the requested model (including OpenAI-compatible,
// Ark video and custom video channel types).
func RelayVideoHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	m := meta.GetByContext(c)

	// Multipart/form-data (vLLM-OMNI wire format): forward the raw body to the
	// upstream and either return the job record (async) or stream back the raw
	// video bytes (sync). This path does not parse content[] and supports file
	// uploads (input_reference) and the full vLLM-OMNI field set.
	if isMultipartVideoRequest(c) {
		// The model field was already extracted by getRequestModel and used by
		// TokenAuth/Distribute to select a channel. Ensure it is mirrored onto
		// the meta for logging/billing.
		if m.OriginModelName == "" {
			m.OriginModelName = videoModelFromMultipart(c)
		}
		return relayVideoMultipart(c, m, relayMode)
	}

	var openReq OpenAIVideoCreateRequest
	if err := common.UnmarshalBodyReusable(c, &openReq); err != nil {
		return videoErrorf(http.StatusBadRequest, "invalid_request", "invalid request body: %v", err)
	}
	if strings.TrimSpace(openReq.Model) == "" {
		return videoErrorf(http.StatusBadRequest, "model_required", "model is required")
	}

	req := openReq.toCreateVideoTaskRequest()
	normalizeCreateVideoTaskRequest(req)
	if err := validateCreateVideoTaskRequest(req); err != nil {
		return videoErrorf(http.StatusBadRequest, "invalid_request", err.Error())
	}

	// Map the requested model onto the upstream channel model (same convention
	// as the text relay: Distribute selected a channel for the original model).
	originModel := openReq.Model
	m.OriginModelName = originModel
	if mapped := m.ModelMapping; mapped != nil {
		if upstreamModel, ok := mapped[originModel]; ok && strings.TrimSpace(upstreamModel) != "" {
			req.Model = strings.TrimSpace(upstreamModel)
			m.ActualModelName = req.Model
		} else {
			m.ActualModelName = originModel
			req.Model = originModel
		}
	} else {
		m.ActualModelName = originModel
	}

	channel := videoChannelFromContext(c)
	task, respBody, httpStatus, bizErr := createVideoTaskCore(ctx, m.UserId, m.TokenId, channel, originModel, req, m.RequestURLPath, relayMode)
	if bizErr != nil {
		// For upstream failures, surface the upstream status so controller.Relay
		// can retry on 429/5xx with a fresh channel. For non-retryable upstream
		// errors (4xx), return the upstream body directly to the client.
		if respBody != nil && len(respBody) > 0 && httpStatus != 0 && (httpStatus < 200 || httpStatus >= 300) {
			c.Data(httpStatus, "application/json", respBody)
			// Returning nil keeps controller.Relay from double-writing a generic
			// error envelope; the upstream response has already been forwarded.
			// But we still want retry on 429/5xx, so return the structured error
			// (which carries the upstream status) for those cases instead.
			if httpStatus == http.StatusTooManyRequests || httpStatus/100 == 5 {
				return bizErr
			}
			return nil
		}
		return bizErr
	}

	// Async: return the task id immediately.
	if relayMode == relaymode.Video {
		c.JSON(http.StatusAccepted, gin.H{
			"id":         task.TaskId,
			"object":     "video",
			"model":      task.Model,
			"status":     task.Status,
			"created_at": task.CreatedTime,
		})
		return nil
	}

	// Sync: poll the upstream until terminal or timeout.
	timeoutSeconds := openReq.Timeout
	if timeoutSeconds <= 0 {
		timeoutSeconds = videoSyncDefaultTimeoutSeconds
	}
	pollInterval := openReq.PollInterval
	if pollInterval < videoSyncMinPollInterval {
		pollInterval = videoSyncDefaultPollInterval
	}

	// Always check the task state immediately — the upstream may have already
	// finished between create and now, so don't waste a poll interval waiting
	// on the ticker before the first check.
	if refreshed, refreshErr := refreshVideoTaskFromUpstream(ctx, task); refreshErr != nil {
		return refreshErr
	} else if refreshed != nil {
		task = refreshed
	}
	if task.Status == model.VideoTaskStatusSucceeded || task.Status == model.VideoTaskStatusFailed {
		return videoSyncRespond(c, task)
	}

	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Prefer the client-disconnect signal when both ctx.Done() and
			// ticker.C are ready (Go select is randomized; check ctx first
			// to avoid an extra poll interval of delay on disconnect).
			return videoErrorf(http.StatusRequestTimeout, "video_sync_timeout", "client disconnected while waiting for video")
		case <-ticker.C:
		}
		if time.Now().After(deadline) {
			resp := mapVideoTaskToOpenAIResponse(task)
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("video generation did not finish within %d seconds", timeoutSeconds),
					"type":    "video_sync_timeout",
					"code":    "video_sync_timeout",
				},
				"video": resp,
			})
			return nil
		}
		refreshed, refreshErr := refreshVideoTaskFromUpstream(ctx, task)
		if refreshErr != nil {
			return refreshErr
		}
		if refreshed != nil {
			task = refreshed
		}
		if task.Status == model.VideoTaskStatusSucceeded || task.Status == model.VideoTaskStatusFailed {
			return videoSyncRespond(c, task)
		}
	}
}

// videoSyncRespond sends the final response for /v1/videos/sync on the JSON
// path. On success it streams the raw video bytes (vLLM-OMNI sync semantics);
// on failure it returns the JSON task envelope.
func videoSyncRespond(c *gin.Context, task *model.VideoGenerationTask) *relaymodel.ErrorWithStatusCode {
	if task.Status == model.VideoTaskStatusSucceeded && strings.TrimSpace(task.VideoURL) != "" {
		if bizErr := streamVideoBytes(c, task.VideoURL); bizErr != nil {
			return bizErr
		}
		return nil
	}
	c.JSON(http.StatusOK, mapVideoTaskToOpenAIResponse(task))
	return nil
}

// streamVideoBytes fetches the video at videoURL and streams the raw bytes to
// the client with a video/* Content-Type, mirroring the vLLM-OMNI sync response
// (raw video bytes rather than a JSON envelope).
func streamVideoBytes(c *gin.Context, videoURL string) *relaymodel.ErrorWithStatusCode {
	u, err := url.Parse(strings.TrimSpace(videoURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return videoErrorf(http.StatusBadRequest, "invalid_video_url", "invalid video url")
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, videoURL, nil)
	if err != nil {
		return videoErrorf(http.StatusBadGateway, "build_video_request_failed", err.Error())
	}
	if rg := c.GetHeader("Range"); rg != "" {
		req.Header.Set("Range", rg)
	}
	resp, err := client.GetHTTPClient().Do(req)
	if err != nil {
		return videoErrorf(http.StatusBadGateway, "fetch_video_failed", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return videoErrorFromUpstream(resp.StatusCode, body)
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
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
	return nil
}

// RelayVideoRetrieve handles GET /v1/videos/:id — returns the upstream video
// task envelope as-is, refreshing from the upstream when possible. The full
// upstream response (progress, seconds, quality, file_name, error, ...) is
// preserved; only `status: completed` is normalized to `succeeded` and an
// `updated_at` is added based on the local task state. Falls back to a
// minimal envelope built from the local task when no upstream response is
// available.
func RelayVideoRetrieve(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	taskID := c.Param("id")
	task, bizErr := getVideoTaskCore(c.Request.Context(), userId, taskID)
	if bizErr != nil {
		videoWriteError(c, bizErr)
		return
	}
	if task != nil && strings.TrimSpace(task.ResponsePayload) != "" {
		if upTask := parseUpstreamVideoTask([]byte(task.ResponsePayload)); upTask != nil {
			_ = forwardUpstreamTask(c, upTask, task.UpdatedTime, task.Model)
			return
		}
	}
	// Fallback: minimal envelope when the upstream is unavailable.
	c.JSON(http.StatusOK, mapVideoTaskToOpenAIResponse(task))
}

// RelayVideoList handles GET /v1/videos — lists the caller's video tasks.
// Supports OpenAI-style limit/after as well as page/page_size pagination.
func RelayVideoList(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	pageNum, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if pageNum < 1 {
		pageNum = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", c.DefaultQuery("limit", "20")))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	tasks, total, bizErr := listVideoTasksCore(c.Request.Context(), userId, pageNum, pageSize)
	if bizErr != nil {
		videoWriteError(c, bizErr)
		return
	}
	items := make([]json.RawMessage, 0, len(tasks))
	for _, t := range tasks {
		// Prefer the persisted upstream envelope, which preserves all fields
		// (progress, seconds, quality, file_name, ...); fall back to the
		// synthesized envelope when no upstream response is stored.
		forwarded := false
		if t != nil && strings.TrimSpace(t.ResponsePayload) != "" {
			if upTask := parseUpstreamVideoTask([]byte(t.ResponsePayload)); upTask != nil {
				buf, err := forwardUpstreamTaskJSON(upTask, t.UpdatedTime, t.Model)
				if err == nil {
					items = append(items, buf)
					forwarded = true
				}
			}
		}
		if !forwarded {
			b, _ := json.Marshal(mapVideoTaskToOpenAIResponse(t))
			items = append(items, b)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"object":    "list",
		"data":      items,
		"total":     total,
		"page":      pageNum,
		"page_size": pageSize,
	})
}

// RelayVideoDelete handles DELETE /v1/videos/:id — cancels/deletes a task.
func RelayVideoDelete(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	taskID := c.Param("id")
	task, bizErr := deleteVideoTaskCore(c.Request.Context(), userId, taskID)
	if bizErr != nil {
		videoWriteError(c, bizErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      taskID,
		"object":  "video",
		"deleted": true,
		"status":  task.Status,
	})
}

// ---- Centralized OpenAI-style error helpers ----

// videoErrorf builds an OpenAI-style error envelope (without writing it),
// carrying the status code so controller.Relay can classify retryability.
func videoErrorf(statusCode int, code, format string, args ...any) *relaymodel.ErrorWithStatusCode {
	return &relaymodel.ErrorWithStatusCode{
		Error: relaymodel.Error{
			Message: fmt.Sprintf(format, args...),
			Type:    videoErrorType(statusCode),
			Code:    code,
		},
		StatusCode: statusCode,
	}
}

// videoErrorFromUpstream maps a non-2xx upstream response into an OpenAI-style
// error envelope, preserving the upstream status code so the relay retry loop
// can act on 429/5xx.
func videoErrorFromUpstream(statusCode int, body []byte) *relaymodel.ErrorWithStatusCode {
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = fmt.Sprintf("upstream returned status %d", statusCode)
	}
	return &relaymodel.ErrorWithStatusCode{
		Error: relaymodel.Error{
			Message: msg,
			Type:    videoErrorType(statusCode),
			Code:    fmt.Sprintf("upstream_%d", statusCode),
		},
		StatusCode: statusCode,
	}
}

// videoWriteError serializes a relaymodel.ErrorWithStatusCode to the client in
// the standard OpenAI error envelope.
func videoWriteError(c *gin.Context, bizErr *relaymodel.ErrorWithStatusCode) {
	if bizErr == nil {
		return
	}
	c.JSON(bizErr.StatusCode, gin.H{
		"error": gin.H{
			"message": bizErr.Error.Message,
			"type":    bizErr.Error.Type,
			"code":    bizErr.Error.Code,
		},
	})
}

// videoErrorType returns the OpenAI-style error type for a status code.
func videoErrorType(statusCode int) string {
	switch {
	case statusCode == http.StatusUnauthorized:
		return "authentication_error"
	case statusCode == http.StatusForbidden:
		return "forbidden_error"
	case statusCode == http.StatusNotFound:
		return "not_found_error"
	case statusCode == http.StatusTooManyRequests:
		return "rate_limit_error"
	case statusCode >= 500 && statusCode < 600:
		return "server_error"
	case statusCode >= 400 && statusCode < 500:
		return "invalid_request_error"
	default:
		return "api_error"
	}
}
