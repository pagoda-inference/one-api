# 批量推理

批量推理允许您上传包含多个请求的 JSONL 文件，后台异步执行，适合离线批处理场景。

---

## 功能入口

登录后，点击左侧菜单 **批量推理** 进入。（仅普通用户可见，管理员除外）

---

## 前置准备

### JSONL 文件格式

每行一个 JSON 对象，遵循 OpenAI Batch API 规范：

```jsonl
{"custom_id": "request-1", "method": "POST", "url": "/v1/chat/completions", "body": {"model": "bedi/glm-4", "messages": [{"role": "user", "content": "你好"}], "max_tokens": 1000}}
{"custom_id": "request-2", "method": "POST", "url": "/v1/chat/completions", "body": {"model": "bedi/glm-4", "messages": [{"role": "user", "content": "今天天气如何"}], "max_tokens": 1000}}
```

> [!NOTE]
> 每行是一个独立请求，`custom_id` 用于标识结果对应关系。

### 字段说明

| 字段 | 必填 | 说明 |
|:-----|:-----|:-----|
| custom_id | 是 | 自定义请求标识，结果中用于对应 |
| method | 是 | 目前仅支持 `POST` |
| url | 是 | `/v1/chat/completions` |
| body.model | 是 | 模型 ID |
| body.messages | 是 | 消息列表 |
| body.max_tokens | 否 | 最大输出 Token 数 |
| body.temperature | 否 | 采样温度 |

### 文件要求

| 限制 | 说明 |
|:-----|:-----|
| 最大文件大小 | 100 MB |
| 最大行数 | 无明确限制，受文件大小约束 |
| 编码 | UTF-8 |
| 格式 | JSONL（每行一个 JSON）|

---

## 操作流程

### 第一步：上传文件

1. 点击 **上传文件** 按钮
2. 选择本地 JSONL 文件
3. 等待上传完成
4. 文件出现在文件列表中

> [!NOTE]
> 文件 purpose 为 `batch`，上传后可在任务中选用。

### 第二步：创建批量任务

1. 选择已上传的输入文件
2. 选择要使用的模型
3. 填写任务描述（可选）
4. 点击 **创建任务**

### 第三步：查看任务状态

任务创建后自动进入执行，状态流转：

```
validating → in_progress → finalizing → completed
                                    ↘ cancelled
                                    ↘ expired
```

| 状态 | 说明 |
|:-----|:-----|
| validating | 正在验证请求格式 |
| in_progress | 执行中 |
| finalizing | 结果写入中 |
| completed | 已完成，可下载结果 |
| cancelled | 已手动取消 |
| expired | 已过期（超过最大有效期）|

### 第四步：下载结果

任务完成后：

- 点击 **下载结果** 获取输出文件
- 输出文件同样为 JSONL 格式
- 每行对应输入请求的 `custom_id`

---

## API 调用说明

### 上传文件

```bash
curl https://baotaai.bedicloud.net/v1/files \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@requests.jsonl" \
  -F "purpose=batch"
```

响应示例：

```json
{
  "id": "file-abc123",
  "object": "file",
  "filename": "requests.jsonl",
  "purpose": "batch",
  "created_at": 1713000000,
  "bytes": 1024000
}
```

### 创建批量任务

```bash
curl https://baotaai.bedicloud.net/v1/batches \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "input_file_id": "file-abc123",
    "endpoint": "/v1/chat/completions",
    "model": "bedi/glm-4",
    "description": "测试批量任务"
  }'
```

响应示例：

```json
{
  "id": "batch-abc123",
  "object": "batch",
  "status": "validating",
  "input_file_id": "file-abc123",
  "completion_window": "24h",
  "created_at": 1713000000
}
```

### 查询任务状态

```bash
curl https://baotaai.bedicloud.net/v1/batches/batch-abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 取消任务

```bash
curl https://baotaai.bedicloud.net/v1/batches/batch-abc123/cancel \
  -X POST \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 列出所有任务

```bash
curl "https://baotaai.bedicloud.net/v1/batches?limit=10" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

---

## 结果文件格式

输出文件为 JSONL，每行对应一个请求的结果：

```jsonl
{"id": "batchreq_abc123", "custom_id": "request-1", "response": {"status": 200, "body": {"id": "chatcmpl-xxx", "choices": [{"message": {"role": "assistant", "content": "你好"}}]}}}, "status": "success"}
{"id": "batchreq_def456", "custom_id": "request-2", "response": {"status": 200, "body": {"id": "chatcmpl-yyy", "choices": [{"message": {"role": "assistant", "content": "今天天气晴朗"}}]}}}, "status": "success"}
```

---

## 错误处理

请求级别的错误会记录在结果文件中，`status` 字段为 `failed`，`response.body.error` 包含错误信息。

> [!NOTE]
> 单个请求失败不会中止其他请求，所有请求都会执行完毕。

---

## 有效期限制

| 限制 | 默认值 | 最大值 |
|:-----|:-------|:-------|
| 任务有效期 | 24 小时 | 7 天 |
| 输入文件有效期 | 24 小时 | 7 天 |
| 输出文件有效期 | 24 小时 | 7 天 |

超过有效期后，文件和任务将自动删除。
