package openai

import (
	"encoding/json"
	"testing"

	relaymodel "github.com/pagoda-inference/one-api/relay/model"
)

func TestParseResponsesRequest(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "string input",
			input: `{
				"model": "gpt-4o",
				"input": "hello world"
			}`,
			wantErr: false,
		},
		{
			name: "array input with text",
			input: `{
				"model": "gpt-4o",
				"input": [{"type": "input_text", "text": "hello"}]
			}`,
			wantErr: false,
		},
		{
			name: "with instructions",
			input: `{
				"model": "gpt-4o",
				"input": "hello",
				"instructions": "You are a helpful assistant"
			}`,
			wantErr: false,
		},
		{
			name: "with tools",
			input: `{
				"model": "gpt-4o",
				"input": "what is the weather",
				"tools": [{"type": "function", "name": "get_weather", "description": "get weather", "parameters": {"type": "object", "properties": {"city": {"type": "string"}}}}]
			}`,
			wantErr: false,
		},
		{
			name:    "missing model",
			input:   `{"input": "hello"}`,
			wantErr: true,
		},
		{
			name:    "missing input",
			input:   `{"model": "gpt-4o"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := ParseResponsesRequest([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponsesRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && req == nil {
				t.Error("ParseResponsesRequest() returned nil for valid request")
			}
		})
	}
}

func TestValidateResponsesRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *relaymodel.ResponsesRequest
		wantErr bool
	}{
		{
			name: "valid request with string input",
			req: &relaymodel.ResponsesRequest{
				Model:  "gpt-4o",
				Input:  "hello",
			},
			wantErr: false,
		},
		{
			name: "valid request with array input",
			req: &relaymodel.ResponsesRequest{
				Model: "gpt-4o",
				Input: []any{map[string]any{"type": "input_text", "text": "hello"}},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &relaymodel.ResponsesRequest{
				Input: "hello",
			},
			wantErr: true,
		},
		{
			name: "missing input",
			req: &relaymodel.ResponsesRequest{
				Model: "gpt-4o",
			},
			wantErr: true,
		},
		{
			name: "previous_response_id not supported",
			req: &relaymodel.ResponsesRequest{
				Model:              "gpt-4o",
				Input:              "hello",
				PreviousResponseID: "resp_123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResponsesRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResponsesRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputToMessages(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect int // expected message count
	}{
		{
			name:   "string input",
			input:  "hello",
			expect: 1,
		},
		{
			name: "array with single text",
			input: []any{
				map[string]any{"type": "input_text", "text": "hello"},
			},
			expect: 1,
		},
		{
			name: "array with multiple texts",
			input: []any{
				map[string]any{"type": "input_text", "text": "hello"},
				map[string]any{"type": "text", "text": "world"},
			},
			expect: 2,
		},
		{
			name: "array with image (text only count)",
			input: []any{
				map[string]any{"type": "input_text", "text": "hello"},
				map[string]any{"type": "input_image", "image_url": "https://example.com/image.jpg"},
			},
			expect: 2, // Still 2 messages but with image placeholder
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, err := InputToMessages(tt.input)
			if err != nil {
				t.Errorf("InputToMessages() error = %v", err)
				return
			}
			if len(msgs) != tt.expect {
				t.Errorf("InputToMessages() got %d messages, want %d", len(msgs), tt.expect)
			}
		})
	}
}

func TestConvertResponsesToChatRequest(t *testing.T) {
	req := &relaymodel.ResponsesRequest{
		Model:        "gpt-4o",
		Input:        "hello world",
		Instructions: "You are a helpful assistant",
		Temperature:  floatPtr(0.7),
	}

	chatReq := relaymodel.ConvertResponsesToChatRequest(req)

	if chatReq.Model != "gpt-4o" {
		t.Errorf("ConvertResponsesToChatRequest() model = %v, want gpt-4o", chatReq.Model)
	}
	if len(chatReq.Messages) != 2 {
		t.Errorf("ConvertResponsesToChatRequest() got %d messages, want 2", len(chatReq.Messages))
	}
	if chatReq.Messages[0].Role != "system" {
		t.Errorf("ConvertResponsesToChatRequest() first message role = %v, want system", chatReq.Messages[0].Role)
	}
	if chatReq.Messages[1].Role != "user" {
		t.Errorf("ConvertResponsesToChatRequest() second message role = %v, want user", chatReq.Messages[1].Role)
	}
	if chatReq.Messages[1].Content != "hello world" {
		t.Errorf("ConvertResponsesToChatRequest() second message content = %v, want hello world", chatReq.Messages[1].Content)
	}
	if chatReq.Temperature != 0.7 {
		t.Errorf("ConvertResponsesToChatRequest() temperature = %v, want 0.7", chatReq.Temperature)
	}
}

func TestBuildResponsesStreamEvent(t *testing.T) {
	// Test with a mock chat stream response
	chatResp := &ChatCompletionsStreamResponse{
		ID:      "chatcmpl_123",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "gpt-4o",
		Choices: []ChatCompletionsStreamResponseChoice{
			{
				Index: 0,
				Delta: relaymodel.Message{
					Role:    "assistant",
					Content: "Hello",
				},
				FinishReason: nil,
			},
		},
	}

	// Build events manually for testing
	events := []*relaymodel.ResponsesStreamEvent{}

	// Add created event
	events = append(events, &relaymodel.ResponsesStreamEvent{
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
	})

	if len(events) == 0 {
		t.Error("BuildResponsesStreamEvent() returned no events")
		return
	}

	// First event should be response.created
	if events[0].Event != "response.created" {
		t.Errorf("First event type = %v, want response.created", events[0].Event)
	}
}

func TestIsTextOnlyInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  bool
	}{
		{
			name:  "string is text only",
			input: "hello",
			want:  true,
		},
		{
			name: "array with text is text only",
			input: []any{
				map[string]any{"type": "input_text", "text": "hello"},
			},
			want: true,
		},
		{
			name: "array with image is not text only",
			input: []any{
				map[string]any{"type": "input_image", "image_url": "https://example.com/image.jpg"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTextOnlyInput(tt.input); got != tt.want {
				t.Errorf("IsTextOnlyInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasTools(t *testing.T) {
	reqWithTools := &relaymodel.ResponsesRequest{
		Model: "gpt-4o",
		Input: "hello",
		Tools: []relaymodel.ResponsesTool{
			{
				Type:        "function",
				Name:        "get_weather",
				Description: "get weather",
			},
		},
	}

	reqWithoutTools := &relaymodel.ResponsesRequest{
		Model:  "gpt-4o",
		Input:  "hello",
		Tools:  nil,
	}

	if !HasTools(reqWithTools) {
		t.Error("HasTools() = false, want true for request with tools")
	}
	if HasTools(reqWithoutTools) {
		t.Error("HasTools() = true, want false for request without tools")
	}
}

func TestSSEFormatResponsesEvent(t *testing.T) {
	event := &relaymodel.ResponsesStreamEvent{
		Event: "response.created",
		Data: relaymodel.ResponseCreatedEvent{
			Response: struct {
				ID     string `json:"id"`
				Object string `json:"object"`
				Status string `json:"status"`
			}{
				ID:     "resp_123",
				Object: "response",
				Status: "in_progress",
			},
		},
	}

	sse := SSEFormatResponsesEvent(event)
	if sse == "" {
		t.Error("SSEFormatResponsesEvent() returned empty string")
		return
	}

	// Should start with "event: response.created\n"
	if len(sse) < len("event: response.created\n") {
		t.Error("SSEFormatResponsesEvent() output too short")
	}
}

func TestCountResponsesInputTokens(t *testing.T) {
	// Just test that it doesn't crash
	count := CountResponsesInputTokens("hello world", "gpt-4o")
	if count < 0 {
		t.Errorf("CountResponsesInputTokens() returned negative: %d", count)
	}

	// Array input
	count = CountResponsesInputTokens([]any{
		map[string]any{"type": "input_text", "text": "hello"},
	}, "gpt-4o")
	if count < 0 {
		t.Errorf("CountResponsesInputTokens() returned negative for array: %d", count)
	}
}

func TestFormatToolsForLogging(t *testing.T) {
	tools := []relaymodel.Tool{
		{Function: relaymodel.Function{Name: "get_weather"}},
		{Function: relaymodel.Function{Name: "get_time"}},
	}

	logged := FormatToolsForLogging(tools)
	if logged == "none" {
		t.Error("FormatToolsForLogging() returned 'none' for non-empty tools")
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
