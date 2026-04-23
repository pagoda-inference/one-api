# API 文档

---

## 认证方式

所有 API 请求需要在 Header 中携带认证信息：

```http
Authorization: Bearer YOUR_API_KEY
```

> [!WARNING]
> API Key 是您的访问凭证，请勿泄露给他人。

---

## 文本系列

### OpenAI-compatible 接口

#### 聊天补全

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

### Anthropic 兼容

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

---

### 文本嵌入

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

---

### 语义检索

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

---

## 图像系列

功能开发中，即将上线。

---

## 语音系列

功能开发中，即将上线。

---

## 视频系列

功能开发中，即将上线。

---

## 文件解析

**POST** `/v1/files/file_parse`

上传文件并解析其内容，支持 PDF、Word、Excel、TXT 等格式。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| file | file | ✓ | 要解析的文件 |
| model | string | ✓ | 解析模型名称 |

```bash
curl -X POST https://baotaai.bedicloud.net/v1/files/file_parse \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@/path/to/document.pdf" \
  -F "model=bedi/document-parse"
```

```json
{
  "id": "file-xxxxx",
  "object": "file_parse",
  "status": "completed",
  "content": "这是解析后的文档内容...",
  "usage": {"prompt_tokens": 1000, "total_tokens": 1000}
}
```

---

## 批量处理

### 批量推理

**POST** `/v1/batches`

提交离线批量推理任务。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | 模型名称 |
| input_file_id | string | ✓ | 输入文件 ID |
| endpoint | string | ✓ | 目标端点 (如 `/v1/chat/completions`) |
| completion_window | string | ✓ | 完成时间窗口 (如 `24h`) |

**GET** `/v1/batches`

查询批量任务列表。

**GET** `/v1/batches/:id`

查询指定批量任务状态和结果。

**POST** `/v1/batches/:id/cancel`

取消运行中的批量任务。

---

## 模型列表

**GET** `/v1/models`

获取所有可用模型列表。不同用户可用的模型可能不同。

```bash
curl https://baotaai.bedicloud.net/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

```json
{
  "object": "list",
  "data": [
    {
      "id": "bedi/glm-4.7",
      "object": "model",
      "provider": "BEDI",
      "type": "chat"
    }
  ]
}
```

---

## SDK 示例

### Python

```bash
pip install openai
```

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://baotaai.bedicloud.net/v1"
)

# 聊天
chat_response = client.chat.completions.create(
    model="bedi/glm-4.7",
    messages=[{"role": "user", "content": "你好"}]
)
print(chat_response.choices[0].message.content)

# Embedding
embedding_response = client.embeddings.create(
    model="bedi/bge-m3",
    input="Hello, world!"
)
print(embedding_response.data[0].embedding)
```

### JavaScript / TypeScript

```bash
npm install openai
```

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'YOUR_API_KEY',
  baseURL: 'https://baotaai.bedicloud.net/v1'
});

// 聊天
const chatResponse = await client.chat.completions.create({
  model: 'bedi/glm-4.7',
  messages: [{ role: 'user', content: '你好' }]
});
console.log(chatResponse.choices[0].message.content);

// Embedding
const embeddingResponse = await client.embeddings.create({
  model: 'bedi/bge-m3',
  input: 'Hello, world!'
});
console.log(embeddingResponse.data[0].embedding);
```

---

## 错误码

| 错误码 | 说明 | 处理建议 |
|:------:|:-----|:---------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 401 | API Key 无效或已过期 | 检查或重新创建 API Key |
| 403 | 无权限访问该模型 | 检查 Key 是否有该模型权限 |
| 404 | 模型不存在 | 检查模型名称是否正确 |
| 408 | 请求超时 | 减少 max_tokens 或稍后重试 |
| 413 | 请求体过大 | 减少输入文本长度 |
| 429 | 请求频率超限 (RPM/TPM) | 降低请求频率或申请更高配额 |
| 500 | 服务器内部错误 | 联系技术支持 |
| 503 | 服务暂时不可用 | 稍后重试或切换渠道 |

---

## 限流说明

| 限制类型 | 说明 |
|:---------|:-----|
| RPM | Requests Per Minute，每分钟请求数 |
| TPM | Tokens Per Minute，每分钟 Token 数 |
| 并发数 | 同时进行的最大请求数 |

> [!NOTE]
> 具体限流值取决于您的 API Key 权限设置，可在 API Keys 管理页面查看和修改。
---

## 错误码

| 错误码 | 说明 | 处理建议 |
|:------:|:-----|:---------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 401 | API Key 无效或已过期 | 检查或重新创建 API Key |
| 403 | 无权限访问该模型 | 检查 Key 是否有该模型权限 |
| 404 | 模型不存在 | 检查模型名称是否正确 |
| 408 | 请求超时 | 减少 max_tokens 或稍后重试 |
| 413 | 请求体过大 | 减少输入文本长度 |
| 429 | 请求频率超限 (RPM/TPM) | 降低请求频率或申请更高配额 |
| 500 | 服务器内部错误 | 联系技术支持 |
| 503 | 服务暂时不可用 | 稍后重试或切换渠道 |

---

## 限流说明

| 限制类型 | 说明 |
|:---------|:-----|
| RPM | Requests Per Minute，每分钟请求数 |
| TPM | Tokens Per Minute，每分钟 Token 数 |
| 并发数 | 同时进行的最大请求数 |

> [!NOTE]
> 具体限流值取决于您的 API Key 权限设置，可在 API Keys 管理页面查看和修改。

---

## 平台操作 API

以下接口供用户操作自己的账户、额度、订单等资源，使用 **访问令牌（Access Token）** 认证，而非 API Key。

### 获取 Access Token

```bash
curl -X POST https://baotaai.bedicloud.net/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"xxxxxx"}'
```

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "user@example.com",
      "quota": 100000,
      "role": 1
    }
  }
}
```

### 额度充值

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/topup/create` | POST | 创建充值订单 |
| `/api/user/topup` | GET | 充值记录列表 |
| `/api/user/topup/:id` | GET | 充值订单详情 |
| `/api/user/topup/:id/cancel` | POST | 取消订单 |

### 用量查询

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/usage/summary` | GET | 用量汇总 |
| `/api/usage/daily` | GET | 每日用量 |
| `/api/usage/by-model` | GET | 按模型统计 |
| `/api/usage/by-channel` | GET | 按渠道统计 |

### 发票管理

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/invoice` | GET | 发票列表 |
| `/api/user/invoice` | POST | 申请发票 |
| `/api/user/invoice/:id` | GET | 发票详情 |

### 通知

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/notifications` | GET | 通知列表 |
| `/api/user/notifications/unread-count` | GET | 未读数量 |
| `/api/user/notifications/:id/read` | PUT | 标记已读 |

### 签到

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/signin` | POST | 每日签到 |
| `/api/user/signin/records` | GET | 签到记录 |