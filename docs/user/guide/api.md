# API 文档

---

## 快速开始

### 第一步：获取 API Key

登录后进入 **API Keys** 页面，点击「创建 API Key」，设置名称和权限后即可获取。

### 第二步：发送请求

使用获取到的 API Key，通过以下端点发送请求：

```bash
curl https://your-domain.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"model": "glm-4", "messages": [{"role": "user", "content": "你好"}]}'
```

### 第三步：查看响应

成功响应示例：

```json
{
  "id": "chatcmpl-xxxxx",
  "object": "chat.completion",
  "created": 1704067200,
  "model": "glm-4",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "你好！有什么可以帮助你的吗？"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 20, "completion_tokens": 30, "total_tokens": 50}
}
```

---

## 认证方式

所有 API 请求需要在 Header 中携带认证信息：

```http
Authorization: Bearer YOUR_API_KEY
```

> [!WARNING]
> API Key 是您的访问凭证，请勿泄露给他人。

---

## 接口列表

### 聊天补全

**POST** `/v1/chat/completions`

发送对话请求，获取 AI 生成的回答。

#### 请求参数

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
curl https://your-domain.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "glm-4",
    "messages": [
      {"role": "system", "content": "你是一个有用的助手"},
      {"role": "user", "content": "解释一下什么是量子计算"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

```python
import requests

response = requests.post(
    "https://your-domain.com/v1/chat/completions",
    headers={
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_API_KEY"
    },
    json={
        "model": "glm-4",
        "messages": [
            {"role": "system", "content": "你是一个有用的助手"},
            {"role": "user", "content": "解释一下什么是量子计算"}
        ],
        "temperature": 0.7,
        "max_tokens": 1000
    }
)
print(response.json())
```

```javascript
fetch("https://your-domain.com/v1/chat/completions", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer YOUR_API_KEY"
  },
  body: JSON.stringify({
    model: "glm-4",
    messages: [
      { role: "system", content: "你是一个有用的助手" },
      { role: "user", content: "解释一下什么是量子计算" }
    ],
    temperature: 0.7,
    max_tokens: 1000
  })
})
.then(res => res.json())
.then(data => console.log(data));
```

#### 响应示例

```json
{
  "id": "chatcmpl-xxxxx",
  "object": "chat.completion",
  "created": 1704067200,
  "model": "glm-4",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "量子计算是一种利用量子力学原理进行信息处理的计算方式..."
    },
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
curl -N https://your-domain.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "glm-4",
    "messages": [{"role": "user", "content": "讲一个笑话"}],
    "stream": true
  }'
```

---

### 文本嵌入

**POST** `/v1/embeddings`

将文本转换为向量表示，用于检索、相似度计算等场景。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | Embedding 模型名称 |
| input | string/array | ✓ | 要嵌入的文本 |
| encoding_format | string | - | 返回格式 (float/base64) |

#### 请求示例

```bash
curl https://your-domain.com/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "bge-m3",
    "input": "Hello, world!"
  }'
```

```python
import requests

response = requests.post(
    "https://your-domain.com/v1/embeddings",
    headers={
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_API_KEY"
    },
    json={
        "model": "bge-m3",
        "input": "Hello, world!"
    }
)
print(response.json())
```

```javascript
fetch("https://your-domain.com/v1/embeddings", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer YOUR_API_KEY"
  },
  body: JSON.stringify({
    model: "bge-m3",
    input: "Hello, world!"
  })
})
.then(res => res.json())
.then(data => console.log(data));
```

#### 响应示例

```json
{
  "object": "list",
  "data": [{
    "object": "embedding",
    "embedding": [0.123, -0.456, 0.789, ...],
    "index": 0
  }],
  "model": "bge-m3",
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

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | Rerank 模型名称 |
| query | string | ✓ | 查询文本 |
| documents | array | ✓ | 文档列表 |

#### 请求示例

```bash
curl https://your-domain.com/v1/rerank \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "rerank",
    "query": "什么是量子计算",
    "documents": [
      "量子计算是一种基于量子力学原理的计算方式",
      "传统计算机使用二进制比特",
      "人工智能技术发展迅速"
    ]
  }'
```

```python
import requests

response = requests.post(
    "https://your-domain.com/v1/rerank",
    headers={
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_API_KEY"
    },
    json={
        "model": "rerank",
        "query": "什么是量子计算",
        "documents": [
            "量子计算是一种基于量子力学原理的计算方式",
            "传统计算机使用二进制比特",
            "人工智能技术发展迅速"
        ]
    }
)
print(response.json())
```

```javascript
fetch("https://your-domain.com/v1/rerank", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer YOUR_API_KEY"
  },
  body: JSON.stringify({
    model: "rerank",
    query: "什么是量子计算",
    documents: [
      "量子计算是一种基于量子力学原理的计算方式",
      "传统计算机使用二进制比特",
      "人工智能技术发展迅速"
    ]
  })
})
.then(res => res.json())
.then(data => console.log(data));
```

#### 响应示例

```json
{
  "object": "list",
  "results": [
    {
      "index": 0,
      "relevance_score": 0.95,
      "document": "量子计算是一种基于量子力学原理的计算方式"
    },
    {
      "index": 1,
      "relevance_score": 0.30,
      "document": "传统计算机使用二进制比特"
    },
    {
      "index": 2,
      "relevance_score": 0.15,
      "document": "人工智能技术发展迅速"
    }
  ],
  "model": "rerank",
  "usage": {
    "prompt_tokens": 15,
    "total_tokens": 15
  }
}
```

---

### 文件解析

**POST** `/v1/files/file_parse`

上传文件并解析其内容。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| file | file | ✓ | 要解析的文件 (PDF/Word/Excel/TXT) |
| model | string | ✓ | 解析模型名称 |

#### 请求示例

```bash
curl https://your-domain.com/v1/files/file_parse \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@/path/to/document.pdf" \
  -F "model=document-parse"
```

```python
import requests

response = requests.post(
    "https://your-domain.com/v1/files/file_parse",
    headers={
        "Authorization": "Bearer YOUR_API_KEY"
    },
    files={
        "file": open("/path/to/document.pdf", "rb")
    },
    data={
        "model": "document-parse"
    }
)
print(response.json())
```

```javascript
const formData = new FormData();
formData.append("file", fileInput.files[0]);
formData.append("model", "document-parse");

fetch("https://your-domain.com/v1/files/file_parse", {
  method: "POST",
  headers: {
    "Authorization": "Bearer YOUR_API_KEY"
  },
  body: formData
})
.then(res => res.json())
.then(data => console.log(data));
```

#### 响应示例

```json
{
  "id": "file-xxxxx",
  "object": "file_parse",
  "status": "completed",
  "content": "这是解析后的文档内容...",
  "usage": {
    "prompt_tokens": 1000,
    "total_tokens": 1000
  }
}
```

---

### 模型列表

**GET** `/v1/models`

获取所有可用模型的列表。

#### 请求示例

```bash
curl https://your-domain.com/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

```python
import requests

response = requests.get(
    "https://your-domain.com/v1/models",
    headers={
        "Authorization": "Bearer YOUR_API_KEY"
    }
)
print(response.json())
```

```javascript
fetch("https://your-domain.com/v1/models", {
  headers: {
    "Authorization": "Bearer YOUR_API_KEY"
  }
})
.then(res => res.json())
.then(data => console.log(data));
```

#### 响应示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "glm-4",
      "object": "model",
      "created": 1704067200,
      "name": "GLM-4 对话模型",
      "provider": "zhipuai",
      "type": "chat"
    },
    {
      "id": "glm-4v",
      "object": "model",
      "created": 1704067200,
      "name": "GLM-4V 视觉模型",
      "provider": "zhipuai",
      "type": "vision"
    }
  ]
}
```

---

## 模型分类

### 对话模型 (Chat)

用于对话交互、文本生成等场景。

| 模型 | 说明 | 上下文长度 |
|:-----|:-----|:----------:|
| glm-4 | GLM-4 对话模型 | 128K |
| glm-4-flash | GLM-4 快速响应 | 128K |
| claude-3.5-sonnet | Claude 3.5 对话模型 | 200K |
| gpt-4o | GPT-4o 对话模型 | 128K |

### 视觉模型 (Vision)

支持图像理解的多模态模型。

| 模型 | 说明 | 上下文长度 |
|:-----|:-----|:----------:|
| glm-4v | GLM-4V 视觉模型 | 2K 图像 |
| gpt-4o-mini | GPT-4o-mini 多模态 | 128K |
| claude-3.5-haiku | Claude 3.5 Haiku | 200K |

### Embedding 模型

用于文本向量表示。

| 模型 | 说明 | 维度 |
|:-----|:-----|:----:|
| bge-m3 | BGE-M3 多语言 | 1024 |
| text-embedding-3-small | OpenAI 小型 | 1536 |
| text-embedding-3-large | OpenAI 大型 | 3072 |

### Reranker 模型

用于语义检索排序。

| 模型 | 说明 |
|:-----|:-----|
| rerank | 通用排序模型 |
| cohere-rerank | Cohere 排序模型 |

### OCR 模型

用于文档识别和文字提取。

| 模型 | 说明 |
|:-----|:-----|
| ocr | 通用 OCR 识别 |
| document-parse | 文档解析 |

---

## 错误码

| 错误码 | 说明 | 处理建议 |
|:------:|:-----|:---------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 401 | API Key 无效或已过期 | 在设置中检查或重新创建 API Key |
| 403 | 无权限访问该模型 | 检查 Key 是否有该模型权限 |
| 404 | 模型不存在 | 检查模型名称是否正确 |
| 408 | 请求超时 | 减少 max_tokens 或稍后重试 |
| 413 | 请求体过大 | 减少输入文本长度 |
| 429 | 请求频率超限 (RPM/TPM) | 降低请求频率或申请更高配额 |
| 500 | 服务器内部错误 | 联系技术支持 |
| 503 | 服务暂时不可用 | 稍后重试或切换渠道 |
| 10000 | 额度不足 | 充值或等待配额重置 |
| 10001 | Key 被禁用 | 检查 Key 状态 |
| 10002 | Key 未启用 | 启用 Key |
| 10003 | 模型已下架 | 选择其他可用模型 |

---

## SDK 示例

### Python SDK

```python
pip install openai
```

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://your-domain.com/v1"
)

# 聊天
chat_response = client.chat.completions.create(
    model="glm-4",
    messages=[{"role": "user", "content": "你好"}]
)
print(chat_response.choices[0].message.content)

# Embedding
embedding_response = client.embeddings.create(
    model="bge-m3",
    input="Hello, world!"
)
print(embedding_response.data[0].embedding)
```

### JavaScript/TypeScript SDK

```bash
npm install openai
```

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'YOUR_API_KEY',
  baseURL: 'https://your-domain.com/v1'
});

// 聊天
const chatResponse = await client.chat.completions.create({
  model: 'glm-4',
  messages: [{ role: 'user', content: '你好' }]
});
console.log(chatResponse.choices[0].message.content);

// Embedding
const embeddingResponse = await client.embeddings.create({
  model: 'bge-m3',
  input: 'Hello, world!'
});
console.log(embeddingResponse.data[0].embedding);
```

### Go SDK

```bash
go get github.com/sashabaranov/go-openai
```

```go
package main

import (
    "context"
    "fmt"
    openai "github.com/sashabaranov/go-openai"
)

func main() {
    client := openai.NewClient("YOUR_API_KEY")
    client.BaseURL = "https://your-domain.com/v1"

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: "glm-4",
            Messages: []openai.ChatCompletionMessage{
                {Role: "user", Content: "你好"},
            },
        },
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(resp.Choices[0].Message.Content)
}
```

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

以下 API 供用户操作自己的账户、额度、订单等资源。

### 认证

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/login` | POST | 用户登录 |
| `/api/user/register` | POST | 用户注册 |
| `/api/user/logout` | GET | 退出登录 |
| `/api/user/self` | GET | 获取当前用户信息 |
| `/api/user/self` | PUT | 更新用户信息 |
| `/api/user/token` | GET | 生成访问令牌 |

#### 登录

```bash
curl -X POST "https://baotaai.bedicloud.net/api/user/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"xxxxxx"}'
```

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "user@example.com",
      "email": "user@example.com",
      "quota": 100000,
      "role": 1
    }
  }
}
```

#### 注册

```bash
curl -X POST "https://baotaai.bedicloud.net/api/user/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"xxxxxx","username":"用户名"}'
```

### 额度管理

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/topup` | GET | 获取充值记录列表 |
| `/api/user/topup/create` | POST | 创建充值订单 |
| `/api/user/topup/:id` | GET | 获取充值订单详情 |
| `/api/user/topup/:id/cancel` | POST | 取消充值订单 |
| `/api/user/self` | GET | 获取账户余额 |

#### 创建充值订单

```bash
curl -X POST "https://baotaai.bedicloud.net/api/user/topup/create" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100}'
```

```json
{
  "success": true,
  "data": {
    "order_id": "TP20260419001",
    "amount": 100,
    "status": "pending",
    "pay_url": "https://pay.example.com/..."
  }
}
```

### 用量查询

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/usage/summary` | GET | 用量汇总 |
| `/api/usage/by-token` | GET | 按 Token 统计 |
| `/api/usage/by-model` | GET | 按模型统计 |
| `/api/usage/by-channel` | GET | 按渠道统计 |
| `/api/usage/by-hour` | GET | 按小时统计 |
| `/api/usage/daily` | GET | 按天统计 |

#### 查询参数

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| start | string | 开始时间 (YYYY-MM-DD) |
| end | string | 结束时间 (YYYY-MM-DD) |
| model | string | 按模型筛选 |
| channel | int | 按渠道筛选 |

#### 用量汇总

```bash
curl -X GET "https://baotaai.bedicloud.net/api/usage/summary" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

```json
{
  "success": true,
  "data": {
    "total_tokens": 987654,
    "total_quota": 1234567,
    "period": {
      "start": "2026-04-01",
      "end": "2026-04-19"
    }
  }
}
```

### 发票管理

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/invoice` | GET | 获取发票列表 |
| `/api/user/invoice` | POST | 申请发票 |
| `/api/user/invoice/:id` | GET | 获取发票详情 |

#### 申请发票

```bash
curl -X POST "https://baotaai.bedicloud.net/api/user/invoice" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100,"title":"公司名称","tax_id":"911101xx"}'
```

### 签到

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/signin` | POST | 每日签到 |
| `/api/user/signin/records` | GET | 签到记录 |

#### 签到

```bash
curl -X POST "https://baotaai.bedicloud.net/api/user/signin" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

```json
{
  "success": true,
  "data": {
    "quota": 100,
    "total_days": 7
  }
}
```

### 通知

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/notifications` | GET | 获取通知列表 |
| `/api/user/notifications/unread-count` | GET | 未读数量 |
| `/api/user/notifications/:id/read` | PUT | 标记已读 |
| `/api/user/notifications/read-all` | PUT | 全部已读 |

### 模型市场

| 接口 | 方法 | 说明 |
|:-----|:-----|:-----|
| `/api/user/market/models` | GET | 获取模型列表 |
| `/api/user/market/models/:id` | GET | 获取模型详情 |
| `/api/user/market/models/:id/pricing` | GET | 获取模型定价 |
| `/api/user/market/models/:id/trial` | GET/POST | 模型试用 |
| `/api/user/market/providers` | GET | 获取 Provider 列表 |
| `/api/user/market/groups/:id/models` | GET | 获取分组下模型 |
| `/api/user/market/stats` | GET | 获取市场统计 |
| `/api/user/market/calculate` | GET | 计算价格 |

#### 模型试用

```bash
curl -X POST "https://baotaai.bedicloud.net/api/user/market/models/glm-4/trial" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

```json
{
  "success": true,
  "data": {
    "trial_quota": 10000,
    "expires_at": "2026-04-20T00:00:00Z"
  }
}
```

---

## 响应格式

所有 API 响应遵循统一格式：

```json
{
  "success": true,
  "message": "操作成功",
  "data": {}
}
```

错误响应：

```json
{
  "success": false,
  "message": "错误描述",
  "code": 10000
}
```

### 常见错误码

| code | 说明 |
|:----:|:-----|
| 10000 | 额度不足 |
| 10001 | Key 被禁用 |
| 10002 | Key 未启用 |
| 10003 | 模型已下架 |
