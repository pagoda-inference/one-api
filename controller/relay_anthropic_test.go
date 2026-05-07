package controller

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestAnthropicAdapterCompatibility(t *testing.T) {
	t.Run("anthropic request converts tools to openai required tool call", func(t *testing.T) {
		req := &AnthropicRequest{
			Model:     "bedi/deepseek-v4-flash",
			MaxTokens: 1024,
			Stream:    true,
			Messages: []AnthropicMessage{
				{
					Role: "user",
					Content: []any{
						map[string]any{"type": "text", "text": "请调用 Write 工具写 test.txt，内容 下午好。"},
					},
				},
			},
			Tools: []AnthropicTool{
				{
					Name:        "Write",
					Description: "Write a file to disk",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"file_path": map[string]any{"type": "string"},
							"content":   map[string]any{"type": "string"},
						},
						"required": []any{"file_path", "content"},
					},
				},
			},
			ToolChoice: map[string]any{"type": "any"},
		}

		openaiReq := ConvertAnthropicToOpenAI(req)
		if openaiReq.ToolChoice != "required" {
			t.Fatalf("expected required tool choice, got %#v", openaiReq.ToolChoice)
		}
		if len(openaiReq.Tools) != 1 || openaiReq.Tools[0].Function.Name != "Write" {
			t.Fatalf("expected converted Write tool, got %#v", openaiReq.Tools)
		}
		if len(openaiReq.Messages) != 1 || openaiReq.Messages[0].Role != "user" {
			t.Fatalf("expected one user message, got %#v", openaiReq.Messages)
		}
		if _, exists := openaiReq.ExtraFields["thinking_budget_tokens"]; exists {
			t.Fatalf("did not expect thinking_budget_tokens in extra fields: %#v", openaiReq.ExtraFields)
		}
		if openaiReq.ChatTemplateKwargs["enable_thinking"] != false {
			t.Fatalf("expected chat_template_kwargs.enable_thinking=false, got %#v", openaiReq.ChatTemplateKwargs)
		}
	})

	t.Run("anthropic auto tool choice stays auto", func(t *testing.T) {
		req := &AnthropicRequest{
			Model:     "bedi/deepseek-v4-flash",
			MaxTokens: 1024,
			Stream:    true,
			Messages: []AnthropicMessage{
				{Role: "user", Content: "hi"},
			},
			Tools: []AnthropicTool{
				{
					Name:        "Read",
					Description: "Read a file",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"file_path": map[string]any{"type": "string"}},
						"required":   []any{"file_path"},
					},
				},
			},
			ToolChoice: map[string]any{"type": "auto"},
		}

		openaiReq := ConvertAnthropicToOpenAI(req)
		if openaiReq.ToolChoice != "auto" {
			t.Fatalf("expected auto tool choice, got %#v", openaiReq.ToolChoice)
		}
	})

	t.Run("anthropic missing tool choice defaults to auto", func(t *testing.T) {
		req := &AnthropicRequest{
			Model:     "bedi/deepseek-v4-flash",
			MaxTokens: 1024,
			Stream:    true,
			Messages: []AnthropicMessage{
				{Role: "user", Content: "hi"},
			},
			Tools: []AnthropicTool{
				{
					Name:        "Read",
					Description: "Read a file",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"file_path": map[string]any{"type": "string"}},
						"required":   []any{"file_path"},
					},
				},
			},
		}

		openaiReq := ConvertAnthropicToOpenAI(req)
		if openaiReq.ToolChoice != "auto" {
			t.Fatalf("expected default auto tool choice, got %#v", openaiReq.ToolChoice)
		}
	})

	t.Run("anthropic tool_result converts to openai tool message", func(t *testing.T) {
		req := &AnthropicRequest{
			Model: "bedi/deepseek-v4-flash",
			Messages: []AnthropicMessage{
				{
					Role: "user",
					Content: []any{
						map[string]any{"type": "text", "text": "write a file"},
					},
				},
				{
					Role: "assistant",
					Content: []any{
						map[string]any{
							"type":  "tool_use",
							"id":    "toolu_write_1",
							"name":  "Write",
							"input": map[string]any{"file_path": "test.txt", "content": "hello"},
						},
					},
				},
				{
					Role: "user",
					Content: []any{
						map[string]any{
							"type":        "tool_result",
							"tool_use_id": "toolu_write_1",
							"content":     "File written successfully",
						},
					},
				},
			},
		}

		openaiReq := ConvertAnthropicToOpenAI(req)
		toolMsgIndex := -1
		for i, msg := range openaiReq.Messages {
			if msg.Role == "tool" {
				toolMsgIndex = i
				break
			}
		}
		if toolMsgIndex < 0 {
			t.Fatalf("expected converted OpenAI tool message, got %#v", openaiReq.Messages)
		}
		toolMsg := openaiReq.Messages[toolMsgIndex]
		if toolMsg.ToolCallId != "toolu_write_1" {
			t.Fatalf("expected tool_call_id preserved, got %q", toolMsg.ToolCallId)
		}
		if toolMsg.Content != "File written successfully" {
			t.Fatalf("expected tool result content preserved, got %#v", toolMsg.Content)
		}
	})

	t.Run("non-stream tool arguments object preserved", func(t *testing.T) {
		resp := map[string]any{
			"id":    "chatcmpl-test",
			"model": "bedi/deepseek-v4-flash",
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": "",
						"tool_calls": []any{
							map[string]any{
								"id":   "call_read_1",
								"type": "function",
								"function": map[string]any{
									"name": "Read",
									"arguments": map[string]any{
										"file_path": "/tmp/a.ts",
										"limit":     100,
									},
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		b, _ := json.Marshal(resp)

		out, err := convertOpenAITextToAnthropic(b, "bedi/deepseek-v4-flash", true)
		if err != nil {
			t.Fatalf("convertOpenAITextToAnthropic returned error: %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("failed to unmarshal output: %v", err)
		}

		content, ok := got["content"].([]any)
		if !ok || len(content) == 0 {
			t.Fatalf("expected non-empty content blocks, got: %#v", got["content"])
		}

		var toolUse map[string]any
		for _, c := range content {
			m, _ := c.(map[string]any)
			if m["type"] == "tool_use" {
				toolUse = m
				break
			}
		}
		if toolUse == nil {
			t.Fatalf("expected tool_use block in content, got: %#v", content)
		}
		if toolUse["name"] != "Read" {
			t.Fatalf("expected tool_use name Read, got: %#v", toolUse["name"])
		}
		input, ok := toolUse["input"].(map[string]any)
		if !ok {
			t.Fatalf("expected tool_use.input object, got: %#v", toolUse["input"])
		}
		if input["file_path"] != "/tmp/a.ts" {
			t.Fatalf("expected file_path preserved, got: %#v", input["file_path"])
		}
	})

	t.Run("stream delayed tool name and merged args", func(t *testing.T) {
		chunk1 := map[string]any{
			"id":    "chatcmpl-stream",
			"model": "bedi/deepseek-v4-flash",
			"choices": []any{
				map[string]any{
					"delta": map[string]any{
						"tool_calls": []any{
							map[string]any{
								"index": 0,
								"id":    "call_read_1",
								"type":  "function",
								"function": map[string]any{
									"arguments": "{\"file_path\":\"/tmp/a.ts\",",
								},
							},
						},
					},
				},
			},
		}
		chunk2 := map[string]any{
			"id":    "chatcmpl-stream",
			"model": "bedi/deepseek-v4-flash",
			"choices": []any{
				map[string]any{
					"delta": map[string]any{
						"tool_calls": []any{
							map[string]any{
								"index": 0,
								"id":    "call_read_1",
								"type":  "function",
								"function": map[string]any{
									"name":      "Read",
									"arguments": "\"limit\":100}",
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
		}
		b1, _ := json.Marshal(chunk1)
		b2, _ := json.Marshal(chunk2)
		sse := "data: " + string(b1) + "\n" + "data: " + string(b2) + "\n" + "data: [DONE]\n"

		out, err := convertOpenAIStreamToAnthropic([]byte(sse), "bedi/deepseek-v4-flash", true)
		if err != nil {
			t.Fatalf("convertOpenAIStreamToAnthropic returned error: %v", err)
		}
		outStr := string(out)

		if strings.Contains(outStr, `"name":""`) {
			t.Fatalf("unexpected empty tool name in stream output: %s", outStr)
		}
		if !strings.Contains(outStr, `"name":"Read"`) {
			t.Fatalf("expected tool name Read in stream output: %s", outStr)
		}
		if !strings.Contains(outStr, `"partial_json":"{\"file_path\":\"/tmp/a.ts\",\"limit\":100}"`) {
			t.Fatalf("expected merged tool args in stream output, got: %s", outStr)
		}
		if !strings.Contains(outStr, `"type":"input_json_delta"`) {
			t.Fatalf("expected Anthropic input_json_delta for streamed tool input, got: %s", outStr)
		}
	})

	t.Run("stream tool id omitted in later chunks still merges by index", func(t *testing.T) {
		chunk1 := map[string]any{
			"id":    "chatcmpl-stream",
			"model": "bedi/deepseek-v4-flash",
			"choices": []any{
				map[string]any{
					"delta": map[string]any{
						"tool_calls": []any{
							map[string]any{
								"index": 0,
								"id":    "call_write_1",
								"type":  "function",
								"function": map[string]any{
									"name":      "Write",
									"arguments": "{",
								},
							},
						},
					},
				},
			},
		}
		chunk2 := map[string]any{
			"id":    "chatcmpl-stream",
			"model": "bedi/deepseek-v4-flash",
			"choices": []any{
				map[string]any{
					"delta": map[string]any{
						"tool_calls": []any{
							map[string]any{
								"index": 0,
								"function": map[string]any{
									"arguments": "\"file_path\":\"test.txt\",\"content\":\"hello\"}",
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		b1, _ := json.Marshal(chunk1)
		b2, _ := json.Marshal(chunk2)
		sse := "data: " + string(b1) + "\n" + "data: " + string(b2) + "\n" + "data: [DONE]\n"

		out, err := convertOpenAIStreamToAnthropic([]byte(sse), "bedi/deepseek-v4-flash", true)
		if err != nil {
			t.Fatalf("convertOpenAIStreamToAnthropic returned error: %v", err)
		}
		outStr := string(out)

		if !strings.Contains(outStr, `"name":"Write"`) {
			t.Fatalf("expected tool name Write in stream output: %s", outStr)
		}
		if !strings.Contains(outStr, `"partial_json":"{\"file_path\":\"test.txt\",\"content\":\"hello\"}"`) {
			t.Fatalf("expected merged args across id-less chunks, got: %s", outStr)
		}
		if !strings.Contains(outStr, `"stop_reason":"tool_use"`) {
			t.Fatalf("expected tool_use stop reason, got: %s", outStr)
		}
	})

	t.Run("stream block index uniqueness", func(t *testing.T) {
		chunk := map[string]any{
			"id":    "chatcmpl-stream-mixed",
			"model": "bedi/deepseek-v4-flash",
			"choices": []any{
				map[string]any{
					"delta": map[string]any{
						"reasoning_content": "let me think",
						"content":           "final answer",
						"tool_calls": []any{
							map[string]any{
								"index": 0,
								"id":    "call_bash_1",
								"type":  "function",
								"function": map[string]any{
									"name":      "Bash",
									"arguments": "{\"command\":\"pwd\"}",
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
		}
		b, _ := json.Marshal(chunk)
		sse := "data: " + string(b) + "\n" + "data: [DONE]\n"

		out, err := convertOpenAIStreamToAnthropic([]byte(sse), "bedi/deepseek-v4-flash", true)
		if err != nil {
			t.Fatalf("convertOpenAIStreamToAnthropic returned error: %v", err)
		}
		outStr := string(out)

		re := regexp.MustCompile(`"content_block_start","index":([0-9]+).*?"type":"([^"]+)"`)
		matches := re.FindAllStringSubmatch(outStr, -1)
		if len(matches) < 3 {
			t.Fatalf("expected at least 3 block starts (thinking/text/tool), got: %s", outStr)
		}

		indexToType := make(map[string]string)
		for _, m := range matches {
			idx := m[1]
			typ := m[2]
			if prev, ok := indexToType[idx]; ok && prev != typ {
				t.Fatalf("mismatched block type for same index %s: %s vs %s. output=%s", idx, prev, typ, outStr)
			}
			indexToType[idx] = typ
		}
	})
}
