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