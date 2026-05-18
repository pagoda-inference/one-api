# 文件解析

异步任务接口：先提交任务，再查询状态和结果。

## 1. 提交任务

**POST** `/v1/file_parse`

上传文件并创建解析任务，返回 task_id。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| file | file | ✓ | 要解析的文件（PDF、Word、Excel、TXT 等） |
| model | string | ✓ | 解析模型名称，如 `bedi/mineru` |

```bash
curl -X POST https://baotaai.bedicloud.net/v1/file_parse \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "model=bedi/mineru" \
  -F "file=@/path/to/document.pdf"
```

**响应：**

```json
{
  "id": "task-xxxxx",
  "object": "file_parse.task",
  "status": "pending"
}
```

---

## 2. 查询任务状态

**GET** `/v1/file_parse/tasks/<task_id>`

```bash
curl -X GET https://baotaai.bedicloud.net/v1/file_parse/tasks/<task_id> \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**响应：**

```json
{
  "id": "task-xxxxx",
  "object": "file_parse.task",
  "status": "pending|running|succeeded|failed"
}
```

状态说明：
- `pending`: 等待处理
- `running`: 处理中
- `succeeded`: 处理完成
- `failed`: 处理失败

---

## 3. 获取解析结果

**GET** `/v1/file_parse/tasks/<task_id>/result`

```bash
curl -X GET https://baotaai.bedicloud.net/v1/file_parse/tasks/<task_id>/result \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**响应：**

```json
{
  "id": "task-xxxxx",
  "object": "file_parse.result",
  "status": "succeeded",
  "content": "这是解析后的文档内容...",
  "usage": {
    "prompt_tokens": 1000,
    "total_tokens": 1000
  }
}
```

---

## 错误处理

| 状态码 | 说明 |
|:------:|:-----|
| 400 | 参数错误，如缺少 file 或 model |
| 401 | 未提供或无效的 API Key |
| 404 | 任务不存在 |
| 500 | 服务器内部错误 |
