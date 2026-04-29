package controller

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestAnthropicAdapterCompatibility(t *testing.T) {
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
