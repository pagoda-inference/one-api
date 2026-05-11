package model

import (
	"fmt"
	"time"
)

// OpenAI Responses API Request structures

// ResponsesRequest represents an incoming OpenAI Responses API request
type ResponsesRequest struct {
	Model               string                  `json:"model"`
	Input               any                     `json:"input"` // string or []InputContent
	Instructions        string                  `json:"instructions,omitempty"`
	Tools               []ResponsesTool         `json:"tools,omitempty"`
	ToolChoice          any                     `json:"tool_choice,omitempty"`
	Temperature         *float64                `json:"temperature,omitempty"`
	TopP                *float64                `json:"top_p,omitempty"`
	MaxOutputTokens     *int                    `json:"max_output_tokens,omitempty"`
	Stream              bool                    `json:"stream,omitempty"`
	StreamOptions       *ResponsesStreamOptions `json:"stream_options,omitempty"`
	Metadata            map[string]any          `json:"metadata,omitempty"`
	PreviousResponseID  string                  `json:"previous_response_id,omitempty"`
	Reasoning           *ReasoningConfig        `json:"reasoning,omitempty"`
	ResponseFormat      any                     `json:"response_format,omitempty"`
	Seed                *int                    `json:"seed,omitempty"`
	ServiceTier         string                  `json:"service_tier,omitempty"`
}

// InputContent represents a item in the input array
type InputContent struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	ImageURL  string `json:"image_url,omitempty"`
	Name      string `json:"name,omitempty"`
}

// ResponsesStreamOptions controls streaming behavior
type ResponsesStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ResponsesTool represents a tool in the OpenAI Responses API format
type ResponsesTool struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
}

// ReasoningConfig configures reasoning behavior
type ReasoningConfig struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// ResponsesStreamEvent represents a server-sent event in streaming mode
type ResponsesStreamEvent struct {
	Event string `json:"event,omitempty"`
	Data  any    `json:"data"`
}

// OpenAI Responses API Response structures

// ResponsesResponse represents the non-streaming response
type ResponsesResponse struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`
	CreatedAt int64         `json:"created_at"`
	Status    string        `json:"status"`
	Error     *Error        `json:"error,omitempty"`
	Output    []OutputItem  `json:"output,omitempty"`
	Usage     *Usage        `json:"usage,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// OutputItem represents a single output item in the response
type OutputItem struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"` // "message" | "function_call" | "reasoning" | "refusal"
	Message      *ResponseMessage  `json:"message,omitempty"`
	FunctionCall *FunctionCall     `json:"function_call,omitempty"`
	Reasoning    *Reasoning        `json:"reasoning,omitempty"`
	Refusal      string            `json:"refusal,omitempty"`
}

// ResponseMessage represents a message output item in Responses format
type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"` // or []ContentBlock for multimodal
}

// FunctionCall represents a function call output item
type FunctionCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Reasoning represents a reasoning output item
type Reasoning struct {
	Summary []ReasoningSummary `json:"summary,omitempty"`
}

// ReasoningSummary represents a summary of reasoning
type ReasoningSummary struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// Stream event data structures

// ResponseCreatedEvent is sent when the response starts
type ResponseCreatedEvent struct {
	Response struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Status    string `json:"status"`
	} `json:"response"`
}

// OutputItemCreatedEvent is sent when an output item is created
type OutputItemCreatedEvent struct {
	Item struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		FunctionCall *FunctionCall `json:"function_call,omitempty"`
	} `json:"item"`
}

// OutputTextDeltaEvent is sent for text content deltas
type OutputTextDeltaEvent struct {
	ItemID      string `json:"item_id"`
	OutputIndex int    `json:"output_index"`
	ContentIndex int   `json:"content_index"`
	Delta       string `json:"delta"`
}

// OutputTextDoneEvent is sent when text output is complete
type OutputTextDoneEvent struct {
	Item struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Text      string `json:"text"`
	} `json:"item"`
}

// FunctionCallArgumentsDeltaEvent is sent for function call argument deltas
type FunctionCallArgumentsDeltaEvent struct {
	ItemID      string `json:"item_id"`
	Delta       string `json:"delta"`
}

// FunctionCallArgumentsDoneEvent is sent when function call is complete
type FunctionCallArgumentsDoneEvent struct {
	Item *FunctionCall `json:"item"`
}

// ResponseDoneEvent is sent when the response is complete
type ResponseDoneEvent struct {
	Response struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Status    string `json:"status"`
		Usage     *Usage `json:"usage,omitempty"`
	} `json:"response"`
}

// ChatCompletionsResponse is a minimal chat-completions response shape used by responses conversion.
type ChatCompletionsResponse struct {
	Id      string                         `json:"id"`
	Object  string                         `json:"object,omitempty"`
	Created int64                          `json:"created,omitempty"`
	Model   string                         `json:"model,omitempty"`
	Choices []ChatCompletionsResponseChoice `json:"choices"`
	Usage   *Usage                         `json:"usage,omitempty"`
	Error   *Error                         `json:"error,omitempty"`
}

type ChatCompletionsResponseChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// ChatCompletionsStreamResponse is a minimal stream chunk shape used by responses conversion.
type ChatCompletionsStreamResponse struct {
	Id      string                               `json:"id"`
	Object  string                               `json:"object,omitempty"`
	Created int64                                `json:"created,omitempty"`
	Model   string                               `json:"model,omitempty"`
	Choices []ChatCompletionsStreamResponseChoice `json:"choices"`
	Usage   *Usage                               `json:"usage,omitempty"`
}

type ChatCompletionsStreamResponseChoice struct {
	Index        int      `json:"index"`
	Delta        Message  `json:"delta"`
	FinishReason *string  `json:"finish_reason,omitempty"`
}

// ConvertResponsesToChatRequest converts an OpenAI Responses request to a unified Chat request
func ConvertResponsesToChatRequest(req *ResponsesRequest) *GeneralOpenAIRequest {
	// Build messages from input
	messages := []Message{}

	// Add instructions as system message if present
	if req.Instructions != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: req.Instructions,
		})
	}

	// Convert input to messages
	switch v := req.Input.(type) {
	case string:
		messages = append(messages, Message{
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
					messages = append(messages, Message{
						Role:    "user",
						Content: text,
					})
				case "message":
					role, _ := msgItem["role"].(string)
					content, _ := msgItem["content"].(string)
					messages = append(messages, Message{
						Role:    role,
						Content: content,
					})
				case "input_image", "image":
					// For now, extract text part and log
					if url, ok := msgItem["image_url"].(string); ok {
						messages = append(messages, Message{
							Role:    "user",
							Content: "[image: " + url + "]",
						})
					}
				}
			case string:
				// Plain string item
				messages = append(messages, Message{
					Role:    "user",
					Content: msgItem,
				})
			}
		}
	}

	// Convert tools
	var tools []Tool
	if len(req.Tools) > 0 {
		for _, t := range req.Tools {
			tools = append(tools, Tool{
				Id:   fmt.Sprintf("tool_%d", len(tools)),
				Type: t.Type,
				Function: Function{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			})
		}
	}

	// Build request
	chatReq := &GeneralOpenAIRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.Temperature != nil {
		chatReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		chatReq.TopP = req.TopP
	}
	if req.MaxOutputTokens != nil {
		chatReq.MaxTokens = *req.MaxOutputTokens
	}
	if tools != nil {
		chatReq.Tools = tools
	}
	if req.ToolChoice != nil {
		chatReq.ToolChoice = req.ToolChoice
	}
	if req.Stream {
		chatReq.StreamOptions = &StreamOptions{
			IncludeUsage: true,
		}
	}

	return chatReq
}

// ConvertChatResponseToResponses converts a unified Chat response to OpenAI Responses format
func ConvertChatResponseToResponses(chatResp *ChatCompletionsResponse, requestID string) *ResponsesResponse {
	response := &ResponsesResponse{
		ID:        "resp_" + requestID,
		Object:    "response",
		CreatedAt: time.Now().Unix(),
		Status:    "completed",
		Output:    []OutputItem{},
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		response.Error = chatResp.Error
		response.Status = "failed"
		return response
	}

	// Convert choices to output items
	for i, choice := range chatResp.Choices {
		item := OutputItem{
			ID:   fmt.Sprintf("resp_item_%d", i),
			Type: "message",
			Message: &ResponseMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.StringContent(),
			},
		}

		// Handle tool calls
		if len(choice.Message.ToolCalls) > 0 {
			item.Type = "function_call"
			// Take the first tool call
			tc := choice.Message.ToolCalls[0]
			item.FunctionCall = &FunctionCall{
				ID:        tc.ID,
				Type:      "function_call",
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
			item.Message = &ResponseMessage{
				Role:    "assistant",
				Content: choice.Message.StringContent(),
			}
		}

		response.Output = append(response.Output, item)
	}

	if chatResp.Usage != nil {
		response.Usage = chatResp.Usage
	}

	return response
}
