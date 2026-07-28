package controller

import (
	"net/http"
	"testing"

	"github.com/pagoda-inference/one-api/model"
)

func TestDetectVideoTaskKind(t *testing.T) {
	cases := []struct {
		name    string
		content []VideoContentItem
		want    string
	}{
		{"text only", []VideoContentItem{{Type: "text", Text: "a cat"}}, videoKindTextToVideo},
		{"image only", []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentItemImageURL{URL: "https://x.com/a.png"}}}, videoKindImageToVideo},
		{"mixed text+image -> i2v", []VideoContentItem{
			{Type: "text", Text: "look at camera"},
			{Type: "image_url", ImageURL: &VideoContentItemImageURL{URL: "https://x.com/a.png"}},
		}, videoKindImageToVideo},
		{"image_url with empty url -> t2v", []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentItemImageURL{URL: " "}}}, videoKindTextToVideo},
		{"empty -> t2v", nil, videoKindTextToVideo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectVideoTaskKind(tc.content); got != tc.want {
				t.Errorf("detectVideoTaskKind = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateCreateVideoTaskRequest(t *testing.T) {
	cases := []struct {
		name    string
		req     *CreateVideoTaskRequest
		wantErr bool
	}{
		{"nil", nil, true},
		{"empty content", &CreateVideoTaskRequest{Model: "m"}, true},
		{"bad type", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "audio"}}}, true},
		{"image_url missing url", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentItemImageURL{}}}}, true},
		{"text only ok", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "text", Text: "hi"}}, Resolution: "720p", Ratio: "16:9", Duration: 5, FPS: 16}, false},
		{"image_url ok", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentItemImageURL{URL: "https://x.com/a.png"}}}, Resolution: "720p", Ratio: "16:9", Duration: 5, FPS: 16}, false},
		{"bad resolution", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "text", Text: "hi"}}, Resolution: "999p", Ratio: "16:9", Duration: 5, FPS: 16}, true},
		{"bad ratio", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "text", Text: "hi"}}, Resolution: "720p", Ratio: "21:9", Duration: 5, FPS: 16}, true},
		{"duration <=0", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "text", Text: "hi"}}, Resolution: "720p", Ratio: "16:9", Duration: 0, FPS: 16}, true},
		{"fps <=0", &CreateVideoTaskRequest{Model: "m", Content: []VideoContentItem{{Type: "text", Text: "hi"}}, Resolution: "720p", Ratio: "16:9", Duration: 5, FPS: 0}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCreateVideoTaskRequest(tc.req)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateCreateVideoTaskRequest err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNormalizeCreateVideoTaskRequest(t *testing.T) {
	t.Run("defaults applied", func(t *testing.T) {
		req := &CreateVideoTaskRequest{Content: []VideoContentItem{{Type: "text", Text: "hi"}}}
		normalizeCreateVideoTaskRequest(req)
		if req.Model == "" {
			t.Error("model should default by kind, got empty")
		}
		if req.Resolution != "720p" || req.Ratio != "16:9" || req.Duration != 5 || req.FPS != 16 || req.Seed != 42 {
			t.Errorf("defaults not applied: %+v", req)
		}
	})
	t.Run("explicit values preserved", func(t *testing.T) {
		req := &CreateVideoTaskRequest{Model: "wan2.2-i2v", Resolution: "1080p", Ratio: "1:1", Duration: 8, FPS: 24, Seed: 7}
		normalizeCreateVideoTaskRequest(req)
		if req.Model != "wan2.2-i2v" || req.Resolution != "1080p" || req.Ratio != "1:1" || req.Duration != 8 || req.FPS != 24 || req.Seed != 7 {
			t.Errorf("explicit values overwritten: %+v", req)
		}
	})
	t.Run("t2v kind defaults to wan2.2-t2v", func(t *testing.T) {
		req := &CreateVideoTaskRequest{Content: []VideoContentItem{{Type: "text", Text: "hi"}}}
		normalizeCreateVideoTaskRequest(req)
		if req.Model != "wan2.2-t2v" {
			t.Errorf("t2v default model = %q, want wan2.2-t2v", req.Model)
		}
	})
	t.Run("i2v kind defaults to wan2.2-i2v", func(t *testing.T) {
		req := &CreateVideoTaskRequest{Content: []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentItemImageURL{URL: "https://x.com/a.png"}}}}
		normalizeCreateVideoTaskRequest(req)
		if req.Model != "wan2.2-i2v" {
			t.Errorf("i2v default model = %q, want wan2.2-i2v", req.Model)
		}
	})
}

func TestOpenAIVideoCreateRequest_ToCreateVideoTaskRequest(t *testing.T) {
	t.Run("prompt synthesized to content", func(t *testing.T) {
		r := &OpenAIVideoCreateRequest{Model: "m", Prompt: "a cat"}
		got := r.toCreateVideoTaskRequest()
		if len(got.Content) != 1 || got.Content[0].Type != "text" || got.Content[0].Text != "a cat" {
			t.Errorf("prompt not synthesized: %+v", got.Content)
		}
	})
	t.Run("image_url synthesized to content", func(t *testing.T) {
		r := &OpenAIVideoCreateRequest{Model: "m", ImageURL: "https://x.com/a.png"}
		got := r.toCreateVideoTaskRequest()
		if len(got.Content) != 1 || got.Content[0].Type != "image_url" || got.Content[0].ImageURL == nil || got.Content[0].ImageURL.URL != "https://x.com/a.png" {
			t.Errorf("image_url not synthesized: %+v", got.Content)
		}
	})
	t.Run("both prompt+image -> two items, image first", func(t *testing.T) {
		r := &OpenAIVideoCreateRequest{Model: "m", Prompt: "caption", ImageURL: "https://x.com/a.png"}
		got := r.toCreateVideoTaskRequest()
		if len(got.Content) != 2 || got.Content[0].Type != "image_url" || got.Content[1].Type != "text" {
			t.Errorf("both not synthesized correctly: %+v", got.Content)
		}
	})
	t.Run("explicit content wins over prompt/image_url", func(t *testing.T) {
		r := &OpenAIVideoCreateRequest{Model: "m", Prompt: "ignored", Content: []VideoContentItem{{Type: "text", Text: "explicit"}}}
		got := r.toCreateVideoTaskRequest()
		if len(got.Content) != 1 || got.Content[0].Text != "explicit" {
			t.Errorf("explicit content not preserved: %+v", got.Content)
		}
	})
	t.Run("passes through scalar fields", func(t *testing.T) {
		r := &OpenAIVideoCreateRequest{Model: "m", Prompt: "p", Resolution: "1080p", Ratio: "1:1", Duration: 8, FPS: 24, Seed: 7}
		got := r.toCreateVideoTaskRequest()
		if got.Resolution != "1080p" || got.Ratio != "1:1" || got.Duration != 8 || got.FPS != 24 || got.Seed != 7 {
			t.Errorf("scalars not passed through: %+v", got)
		}
	})
}

func TestEstimateVideoOutputTokens(t *testing.T) {
	// 720p 16:9 = 1248x704, duration 5, fps 16 -> 5*1248*704*16/1024
	want := int64(5) * 1248 * 704 * 16 / 1024
	if got := estimateVideoOutputTokens("720p", "16:9", 5, 16); got != want {
		t.Errorf("720p 16:9 = %d, want %d", got, want)
	}
	t.Run("zero duration defaults to 5", func(t *testing.T) {
		got := estimateVideoOutputTokens("720p", "16:9", 0, 16)
		if got != want {
			t.Errorf("zero duration = %d, want %d (defaulted)", got, want)
		}
	})
	t.Run("zero fps defaults to 16", func(t *testing.T) {
		got := estimateVideoOutputTokens("720p", "16:9", 5, 0)
		if got != want {
			t.Errorf("zero fps = %d, want %d (defaulted)", got, want)
		}
	})
	t.Run("unknown resolution falls to 720p-ish dimensions", func(t *testing.T) {
		got := estimateVideoOutputTokens("999p", "16:9", 5, 16)
		if got != want {
			t.Errorf("unknown resolution = %d, want %d (default dims)", got, want)
		}
	})
}

func TestBuildVideoTasksURL(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{"https://ark.cn-beijing.volces.com", "https://ark.cn-beijing.volces.com/contents/generations/tasks"},
		{"https://ark.cn-beijing.volces.com/", "https://ark.cn-beijing.volces.com/contents/generations/tasks"},
		{"https://host/api/v1", "https://host/api/v1/contents/generations/tasks"},
		{"https://host/contents/generations/tasks", "https://host/contents/generations/tasks"},
		{"https://host/contents/generations/tasks/", "https://host/contents/generations/tasks/"},
		{"https://host/path/tasks", "https://host/path/tasks"},
	}
	for _, tc := range cases {
		got, err := buildVideoTasksURL(tc.base)
		if err != nil {
			t.Fatalf("err for %s: %v", tc.base, err)
		}
		if got != tc.want {
			t.Errorf("buildVideoTasksURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}

func TestBuildVideoTaskItemURL(t *testing.T) {
	got, err := buildVideoTaskItemURL("https://ark.cn-beijing.volces.com", "cgt-abc 123")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := "https://ark.cn-beijing.volces.com/contents/generations/tasks/cgt-abc%20123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMapVideoTaskToOpenAIResponse(t *testing.T) {
	t.Run("with video url", func(t *testing.T) {
		task := &model.VideoGenerationTask{
			TaskId: "cgt-1", Model: "wan2.2-t2v", Status: model.VideoTaskStatusSucceeded,
			CreatedTime: 100, UpdatedTime: 200, Seed: 7, Resolution: "720p", Ratio: "16:9",
			Duration: 5, FramesPerSecond: 16, VideoURL: "https://x.com/v.mp4",
		}
		r := mapVideoTaskToOpenAIResponse(task)
		if r.ID != "cgt-1" || r.Object != "video" || r.Model != "wan2.2-t2v" || r.Status != model.VideoTaskStatusSucceeded {
			t.Errorf("bad core fields: %+v", r)
		}
		if r.Video == nil || r.Video.URL != "https://x.com/v.mp4" {
			t.Errorf("video url not mapped: %+v", r.Video)
		}
		if r.Usage == nil || r.Usage.TotalTokens != 0 {
			t.Errorf("usage not initialized: %+v", r.Usage)
		}
	})
	t.Run("without video url", func(t *testing.T) {
		task := &model.VideoGenerationTask{TaskId: "cgt-2", Status: model.VideoTaskStatusRunning}
		r := mapVideoTaskToOpenAIResponse(task)
		if r.Video != nil {
			t.Errorf("video should be nil when no url, got %+v", r.Video)
		}
	})
	t.Run("nil task", func(t *testing.T) {
		if r := mapVideoTaskToOpenAIResponse(nil); r != nil {
			t.Errorf("nil task should map to nil, got %+v", r)
		}
	})
}

func TestVideoErrorType(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{400, "invalid_request_error"},
		{401, "authentication_error"},
		{403, "forbidden_error"},
		{404, "not_found_error"},
		{408, "invalid_request_error"},
		{429, "rate_limit_error"},
		{500, "server_error"},
		{502, "server_error"},
		{503, "server_error"},
		{200, "api_error"},
	}
	for _, tc := range cases {
		if got := videoErrorType(tc.code); got != tc.want {
			t.Errorf("videoErrorType(%d) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestVideoErrorf(t *testing.T) {
	e := videoErrorf(http.StatusServiceUnavailable, "no_available_channel", "no available video channel")
	if e.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", e.StatusCode, http.StatusServiceUnavailable)
	}
	if e.Error.Code != "no_available_channel" {
		t.Errorf("code = %q, want no_available_channel", e.Error.Code)
	}
	if e.Error.Type != "server_error" {
		t.Errorf("type = %q, want server_error", e.Error.Type)
	}
	if e.Error.Message != "no available video channel" {
		t.Errorf("message = %q", e.Error.Message)
	}
}
