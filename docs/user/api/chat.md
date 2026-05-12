# 文本系列

## OpenAI 兼容

**POST** `/v1/chat/completions`

发送对话请求，获取 AI 生成的回答。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | 模型名称 |
| messages | array | ✓ | 对话消息数组 |
| temperature | float | - | 采样温度 (0-2) |
| top_p | float | - | 核采样概率 |
| max_tokens | int | - | 最大生成 Token 数 |
| stream | bool | - | 开启流式输出 (SSE) |
| stop | string/array | - | 停止生成标记 |
| presence_penalty | float | - | 话题新鲜度 (-2~2) |
| frequency_penalty | float | - | 频率惩罚 (-2~2) |
| user | string | - | 用户标识 |

#### 请求示例

```bash
curl https://baotaai.bedicloud.net/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/glm-4.7",
    "messages": [
      {"role": "system", "content": "你是一个有用的助手"},
      {"role": "user", "content": "解释一下什么是量子计算"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

#### 响应示例

```json
{
  "id": "chatcmpl-xxxxx",
  "object": "chat.completion",
  "created": 1704067200,
  "model": "bedi/glm-4.7",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "量子计算是一种利用量子力学原理进行信息处理的计算方式..."},
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 150,
    "total_tokens": 200
  }
}
```

#### 流式响应示例

```bash
curl -N https://baotaai.bedicloud.net/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/glm-4.7",
    "messages": [{"role": "user", "content": "讲一个笑话"}],
    "stream": true
  }'
```

## Anthropic 兼容

**POST** `/v1/messages`

Claude Code 等默认使用 Anthropic 格式调用此接口。平台自动识别上游渠道类型，将请求转发至对应上游：

- 若上游为 OpenAI-compatible → 自动转换为 `/v1/chat/completions` 格式
- 若上游为原生 Anthropic → 保持 Anthropic 格式转发

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | 模型名称 |
| messages | array | ✓ | 对话消息数组 |
| system | string | - | 系统提示 |
| max_tokens | int | ✓ | 最大生成 Token 数 |
| stream | bool | - | 开启流式输出 |

```bash
curl -X POST https://baotaai.bedicloud.net/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/claude-3.5-sonnet",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 1024
  }'
```

## OpenAI Responses API

**POST** `/v1/responses`

OpenAI 推出的新一代统一 API，支持更强的工具调用、流式输出和结构化输出。与 `/v1/chat/completions` 共用同一套渠道分发逻辑，但请求/响应格式不同。

### 功能特性

- **流式 SSE 事件**：严格对齐 OpenAI 官方事件名（`response.created`、`response.output_text.delta`、`response.done` 等）
- **工具调用**：完整支持 `tools` 和 `tool_choice`，支持流式增量参数
- **配额安全**：预扣费 + 降级回滚 + 实际用量结算

### 兼容性目标

| 上游 | 支持状态 | 说明 |
|------|---------|------|
| OpenAI官方 | ✓ 完全兼容 | 原生透传 |
| Anthropic 原生 | ✗ 不支持 | 请使用 `/v1/messages` |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | 模型名称 |
| input | string / array | ✓ | 输入内容，见下方说明 |
| instructions | string | - | 系统指令，转换为 `messages[0].system` |
| tools | array | - | 工具定义数组 |
| tool_choice | string / object | - | 工具选择策略 (`auto`/`required`/`none`) |
| temperature | float | - | 采样温度 (0-2) |
| top_p | float | - | 核采样概率 |
| max_output_tokens | int | - | 最大生成 Token 数 |
| stream | bool | - | 开启流式输出 (SSE) |
| stream_options | object | - | 流式选项，`include_usage: true` 可在最后包含用量 |
| metadata | object | - | 元数据，透传至日志 |
| reasoning | object | - | 推理配置（`effort`/`summary`），按上游能力决定是否传递 |
| previous_response_id | string | - | **暂不支持**，返回错误 |

#### input 格式说明

```json
// 简单文本
"input": "你好，请介绍一下自己"

// 多段数组
"input": [
  {"type": "input_text", "text": "这张图片里有什么？"},
  {"type": "input_image", "image_url": "https://example.com/image.jpg"}
]

// 消息数组（兼容性格式）
"input": [
  {"type": "message", "role": "user", "content": "Hello"}
]
```

#### tools 格式

```json
"tools": [
  {
    "type": "function",
    "name": "get_weather",
    "description": "获取天气信息",
    "parameters": {
      "type": "object",
      "properties": {
        "city": {"type": "string", "description": "城市名称"}
      },
      "required": ["city"]
    }
  }
]
```

### 请求示例

**非流式**
```bash
curl https://baotaai.bedicloud.net/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/gpt-4o",
    "input": "解释一下量子计算的基本原理",
    "instructions": "你是一位物理学家，用通俗易懂的语言解释",
    "temperature": 0.7,
    "max_output_tokens": 1000
  }'
```

**带工具调用**
```bash
curl https://baotaai.bedicloud.net/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/gpt-4o",
    "input": "北京今天天气怎么样？",
    "tools": [
      {
        "type": "function",
        "name": "get_weather",
        "description": "获取天气信息",
        "parameters": {
          "type": "object",
          "properties": {
            "city": {"type": "string"}
          },
          "required": ["city"]
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

**流式**
```bash
curl -N https://baotaai.bedicloud.net/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/gpt-4o",
    "input": "讲一个关于AI的笑话",
    "stream": true,
    "stream_options": {"include_usage": true}
  }'
```

### 响应结构

#### 非流式响应

**文本消息**
```json
{
  "id": "resp_abc123xyz",
  "object": "response",
  "created_at": 1704067200,
  "status": "completed",
  "output": [
    {
      "id": "resp_item_0",
      "type": "message",
      "message": {
        "role": "assistant",
        "content": "量子计算是一种利用量子力学原理进行信息处理的计算方式..."
      }
    }
  ],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 150,
    "total_tokens": 200
  }
}
```

**工具调用**
```json
{
  "id": "resp_abc123xyz",
  "object": "response",
  "created_at": 1704067200,
  "status": "completed",
  "output": [
    {
      "id": "resp_item_0",
      "type": "function_call",
      "function_call": {
        "id": "call_abc123",
        "type": "function_call",
        "name": "get_weather",
        "arguments": "{\"city\":\"北京\"}"
      },
      "message": {
        "role": "assistant",
        "content": ""
      }
    }
  ],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 30,
    "total_tokens": 80
  }
}
```

**错误响应**
```json
{
  "id": "resp_abc123xyz",
  "object": "response",
  "created_at": 1704067200,
  "status": "failed",
  "error": {
    "type": "invalid_request_error",
    "message": "model is required"
  }
}
```

### 流式 SSE 事件

流式响应使用 Server-Sent Events (SSE)，每个事件格式为：

```
event: <事件名>
data: <JSON数据>

```

#### 事件类型

| 事件名 | 说明 | 关键字段 |
|--------|------|---------|
| `response.created` | 流开始 | `id`, `object`, `status` |
| `response.output_item.created` | 输出项创建 | `id`, `type` |
| `response.output_text.delta` | 文本增量 | `item_id`, `delta` |
| `response.output_text.done` | 文本输出完成 | `item_id`, `text` |
| `response.function_call_arguments.delta` | 工具参数增量 | `item_id`, `delta` |
| `response.function_call_arguments.done` | 工具调用完成 | `item` |
| `response.done` | **流结束（必须发送）** | `usage`, `status` |

#### 流式响应示例

```
event: response.created
data: {"response":{"id":"resp_abc123","object":"response","status":"in_progress"}}

event: response.output_item.created
data: {"item":{"id":"resp_item_0","type":"message"}}

event: response.output_text.delta
data: {"item_id":"resp_item_0","output_index":0,"content_index":0,"delta":"量子"}

event: response.output_text.delta
data: {"item_id":"resp_item_0","output_index":0,"content_index":0,"delta":"计算"}

event: response.output_text.done
data: {"item":{"id":"resp_item_0","type":"message","text":"量子计算是一种..."}}

event: response.done
data: {"response":{"id":"resp_abc123","object":"response","status":"completed","usage":{"prompt_tokens":50,"completion_tokens":150,"total_tokens":200}}}
```


### 工具调用说明

#### 非流式工具调用流程

1. 客户端发送带 `tools` 的请求
2. 服务器返回 `type: "function_call"` 的 output item
3. 客户端执行工具，携带 `previous_response_id` 继续对话（**暂不支持**）

#### 流式工具调用流程

```
response.created
  └─> response.output_item.created (type: "function_call")
        └─> response.function_call_arguments.delta (增量参数)
              └─> ... (多个 delta 事件)
                    └─> response.function_call_arguments.done
                          └─> response.done
```

### 配额与计费

#### 配额生命周期

1. **预扣费 (Pre-consume)**：请求进入时按 `max_output_tokens` 预估用量扣减配额
2. **降级回滚 (Rollback)**：若发生降级或错误，预扣费用返还
3. **实际结算 (Post-consume)**：响应完成后按实际 `usage` 精算，多退少补

#### 用量来源

| 来源 | 说明 |
|------|------|
| `exact` | 上游返回实际用量，按真实 token 数结算 |
| `fallback` | 上游未返回用量，按 `ResponsesUsageFallbackMultiplier` 系数估算 |

### 与其他接口的关系

| 接口 | 适用场景 | 工具调用 | 流式输出 |
|------|---------|---------|---------|
| `/v1/responses` | OpenAI 新 API，结构化输出强 | ✓ 完整支持 | ✓ SSE |
| `/v1/chat/completions` | 通用对话，兼容性最好 | ✓ Function calling | ✓ SSE |
| `/anthropic/v1/messages` | Claude 原生接口 | ✗ 不支持 | ✓ SSE |

### 已知限制

- `previous_response_id` 暂不支持，连续对话功能待实现
- 多模态输入（图片）暂不支持完整透传，降级为文本描述
- `reasoning` 配置按 `ResponsesPassReasoning` 开关控制，默认不传递
- 部分上游（如 vLLM）不支持 Responses API，会自动降级

### 故障排查

**问题：请求返回 400 "input is required"**
```
原因：上游不支持 /v1/responses 格式
解决：平台会自动降级，无需人工干预。若未自动降级，请检查上游版本。
```

**问题：流式响应卡住不结束**
```
原因：上游未发送 response.done 事件
解决：平台实现了超时保护，超时后强制结束流并结算配额。
```

**问题：工具调用参数不完整**
```
原因：流式传输中增量参数未完整接收
解决：确保客户端正确处理 response.function_call_arguments.delta 事件，
      并在 response.function_call_arguments.done 后再解析完整参数。
```

## 文本嵌入

**POST** `/v1/embeddings`

将文本转换为向量表示，用于检索、相似度计算等场景。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | Embedding 模型名称 |
| input | string/array | ✓ | 要嵌入的文本 |
| encoding_format | string | - | 返回格式 (float/base64) |

```bash
curl https://baotaai.bedicloud.net/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/bge-m3",
    "input": "Hello, world!"
  }'
```

```json
{
  "object": "list",
  "data": [{
    "object": "embedding",
    "embedding": [0.123, -0.456, 0.789],
    "index": 0
  }],
  "model": "bedi/bge-m3",
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 5
  }
}
```

## 语义检索

**POST** `/v1/rerank`

对查询与文档列表进行语义相关性排序。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | Rerank 模型名称 |
| query | string | ✓ | 查询文本 |
| documents | array | ✓ | 文档列表 |
| top_n | int | - | 返回前 N 条结果 |

```bash
curl -X POST https://baotaai.bedicloud.net/v1/rerank \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bedi/rerank",
    "query": "什么是量子计算",
    "documents": [
      "量子计算是一种基于量子力学原理的计算方式",
      "传统计算机使用二进制比特",
      "人工智能技术发展迅速"
    ],
    "top_n": 2
  }'
```

```json
{
  "object": "list",
  "results": [
    {"index": 0, "relevance_score": 0.95, "document": "量子计算是一种基于量子力学原理的计算方式"},
    {"index": 2, "relevance_score": 0.15, "document": "人工智能技术发展迅速"}
  ],
  "model": "bedi/rerank",
  "usage": {"prompt_tokens": 15, "total_tokens": 15}
}
```