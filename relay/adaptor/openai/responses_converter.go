package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pagoda-inference/one-api/common/logger"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
)

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		if b, err := json.Marshal(val); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", val)
	}
}

// ConvertResponsesRequest converts an OpenAI Responses request to a format suitable for upstream
func ConvertResponsesRequest(req *relaymodel.ResponsesRequest) (*relaymodel.GeneralOpenAIRequest, error) {
	chatReq := relaymodel.ConvertResponsesToChatRequest(req)
	return chatReq, nil
}

// ResponsesToChatRequestJSON converts Responses request to Chat request JSON bytes
func ResponsesToChatRequestJSON(req *relaymodel.ResponsesRequest) ([]byte, error) {
	chatReq := relaymodel.ConvertResponsesToChatRequest(req)
	return json.Marshal(chatReq)
}

// ConvertChatResponseToResponses converts Chat completions response to Responses format
func ConvertChatResponseToResponses(chatResp *relaymodel.ChatCompletionsResponse, requestID string) *relaymodel.ResponsesResponse {
	return relaymodel.ConvertChatResponseToResponses(chatResp, requestID)
}

// ConvertChatStreamResponseToResponsesEvent converts a streaming chat delta to Responses stream event
func ConvertChatStreamResponseToResponsesEvent(chatResp *relaymodel.ChatCompletionsStreamResponse, isFinal bool) (*relaymodel.ResponsesStreamEvent, error) {
	if len(chatResp.Choices) == 0 {
		return nil, nil
	}

	choice := chatResp.Choices[0]

	// Handle the final chunk with usage
	if isFinal && chatResp.Usage != nil {
		event := &relaymodel.ResponsesStreamEvent{
			Event: "response.done",
			Data: relaymodel.ResponseDoneEvent{
				Response: struct {
					ID     string            `json:"id"`
					Object string            `json:"object"`
					Status string            `json:"status"`
					Usage  *relaymodel.Usage `json:"usage,omitempty"`
				}{
					ID:     chatResp.Id,
					Object: "response",
					Status: "completed",
					Usage:  chatResp.Usage,
				},
			},
		}
		return event, nil
	}

	// Handle content delta
	if deltaText := anyToString(choice.Delta.Content); deltaText != "" {
		event := &relaymodel.ResponsesStreamEvent{
			Event: "response.output_text.delta",
			Data: relaymodel.OutputTextDeltaEvent{
				ItemID:       fmt.Sprintf("resp_item_%d", choice.Index),
				OutputIndex:  0,
				ContentIndex: 0,
				Delta:        deltaText,
			},
		}
		return event, nil
	}

	// Handle tool call delta
	if len(choice.Delta.ToolCalls) > 0 {
		tc := choice.Delta.ToolCalls[0]
		if argsText := anyToString(tc.Function.Arguments); argsText != "" {
			event := &relaymodel.ResponsesStreamEvent{
				Event: "response.function_call_arguments.delta",
				Data: relaymodel.FunctionCallArgumentsDeltaEvent{
					ItemID: tc.Id,
					Delta:  argsText,
				},
			}
			return event, nil
		}
	}

	return nil, nil
}

// BuildResponsesStreamEvent builds Responses stream events from chat stream response
// Note: Does NOT include response.created event - caller controls when to send it
func BuildResponsesStreamEvent(chatResp *relaymodel.ChatCompletionsStreamResponse) ([]*relaymodel.ResponsesStreamEvent, error) {
	events := []*relaymodel.ResponsesStreamEvent{}

	// Process deltas
	for _, choice := range chatResp.Choices {
		// Content delta
		if deltaText := anyToString(choice.Delta.Content); deltaText != "" {
			contentEvent := &relaymodel.ResponsesStreamEvent{
				Event: "response.output_text.delta",
				Data: relaymodel.OutputTextDeltaEvent{
					ItemID:       fmt.Sprintf("resp_item_%d", choice.Index),
					OutputIndex:  0,
					ContentIndex: 0,
					Delta:        deltaText,
				},
			}
			events = append(events, contentEvent)
		}

		// Tool call delta
		if len(choice.Delta.ToolCalls) > 0 {
			tc := choice.Delta.ToolCalls[0]
			// First, output_item_created event
			itemCreatedEvent := &relaymodel.ResponsesStreamEvent{
				Event: "response.output_item.created",
				Data: relaymodel.OutputItemCreatedEvent{
					Item: struct {
						ID           string `json:"id"`
						Type         string `json:"type"`
						FunctionCall *relaymodel.FunctionCall `json:"function_call,omitempty"`
					}{
						ID:   tc.Id,
						Type: "function_call",
						FunctionCall: &relaymodel.FunctionCall{
							ID:   tc.Id,
							Type: "function_call",
						},
					},
				},
			}
			events = append(events, itemCreatedEvent)

			// Then function_call_arguments.delta
			if argsText := anyToString(tc.Function.Arguments); argsText != "" {
				argsDeltaEvent := &relaymodel.ResponsesStreamEvent{
					Event: "response.function_call_arguments.delta",
					Data: relaymodel.FunctionCallArgumentsDeltaEvent{
						ItemID: tc.Id,
						Delta:  argsText,
					},
				}
				events = append(events, argsDeltaEvent)
			}
		}
	}

	return events, nil
}

// SSEFormatResponsesEvent formats a Responses stream event as SSE
func SSEFormatResponsesEvent(event *relaymodel.ResponsesStreamEvent) string {
	if event == nil {
		return ""
	}
	data, err := json.Marshal(event.Data)
	if err != nil {
		logger.Errorf(nil, "failed to marshal responses event: %v", err)
		return ""
	}
	eventStr := event.Event
	if eventStr == "" {
		return fmt.Sprintf("data: %s\n\n", string(data))
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventStr, string(data))
}

// ParseResponsesRequest parses an incoming OpenAI Responses API request
func ParseResponsesRequest(body []byte) (*relaymodel.ResponsesRequest, error) {
	var req relaymodel.ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Responses request: %w", err)
	}
	return &req, nil
}

// ValidateResponsesRequest validates a Responses request
func ValidateResponsesRequest(req *relaymodel.ResponsesRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if req.Input == nil {
		return fmt.Errorf("input is required")
	}
	if req.PreviousResponseID != "" {
		return fmt.Errorf("previous_response_id is not yet supported")
	}
	return nil
}

// InputToMessages converts Responses input to chat messages for token counting
func InputToMessages(input any) ([]relaymodel.Message, error) {
	var messages []relaymodel.Message

	switch v := input.(type) {
	case string:
		messages = append(messages, relaymodel.Message{
			Role:    "user",
			Content: v,
		})
	case []any:
		for _, item := range v {
			switch msgItem := item.(type) {
			case map[string]any:
				msgType, _ := msgItem["type"].(string)
				switch msgType {
				case "input_text", "text":
					text, _ := msgItem["text"].(string)
					messages = append(messages, relaymodel.Message{
						Role:    "user",
						Content: text,
					})
				case "message":
					role, _ := msgItem["role"].(string)
					content, _ := msgItem["content"].(string)
					messages = append(messages, relaymodel.Message{
						Role:    role,
						Content: content,
					})
				case "input_image", "image":
					if url, ok := msgItem["image_url"].(string); ok {
						messages = append(messages, relaymodel.Message{
							Role:    "user",
							Content: "[image: " + url + "]",
						})
					}
				default:
					// Try to parse as string
					if s, ok := msgItem["text"].(string); ok {
						messages = append(messages, relaymodel.Message{
							Role:    "user",
							Content: s,
						})
					}
				}
			case string:
				messages = append(messages, relaymodel.Message{
					Role:    "user",
					Content: msgItem,
				})
			}
		}
	}

	return messages, nil
}

// CountResponsesInputTokens counts tokens in a Responses input
func CountResponsesInputTokens(input any, model string) int {
	messages, err := InputToMessages(input)
	if err != nil {
		logger.Warnf(nil, "failed to convert input to messages: %v", err)
		return 0
	}

	total := 0
	for _, msg := range messages {
		tokens := CountTokenInput(msg.Content, model)
		total += tokens
	}

	return total
}

// GetResponsesInstructionTokens counts tokens in instructions
func GetResponsesInstructionTokens(instructions string, model string) int {
	if instructions == "" {
		return 0
	}
	return CountTokenInput(instructions, model)
}

// IsTextOnlyInput checks if the input contains only text (no images)
func IsTextOnlyInput(input any) bool {
	switch v := input.(type) {
	case string:
		return true
	case []any:
		for _, item := range v {
			switch msgItem := item.(type) {
			case map[string]any:
				msgType, _ := msgItem["type"].(string)
				if msgType == "input_image" || msgType == "image" {
					return false
				}
			}
		}
		return true
	}
	return false
}

// IsMultimodalInput checks if the input contains multimodal content
func IsMultimodalInput(input any) bool {
	switch v := input.(type) {
	case string:
		return false
	case []any:
		for _, item := range v {
			switch msgItem := item.(type) {
			case map[string]any:
				msgType, _ := msgItem["type"].(string)
				if msgType == "input_image" || msgType == "image" {
					return true
				}
			}
		}
	}
	return false
}

// HasTools checks if the request has tools defined
func HasTools(req *relaymodel.ResponsesRequest) bool {
	return len(req.Tools) > 0
}

// FormatToolsForLogging formats tools for structured logging
func FormatToolsForLogging(tools []relaymodel.Tool) string {
	if len(tools) == 0 {
		return "none"
	}
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t.Function.Name != "" {
			names = append(names, t.Function.Name)
			continue
		}
		names = append(names, "unknown")
	}
	return strings.Join(names, ", ")
}
