# 视频系列

## 文生视频

### **API 端点**

#### **1. 创建视频生成任务**

**请求：**

```bash
YOUR_API_KEY=xxx
curl -s https://baotaai.bedicloud.net/api/v1/contents/generations/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY" \
  -d '{
    "model": "bedi/wan2.2-t2v-a14b",
    "content": [
      {
        "type": "text",
        "text": "写实风格，一只猫咪在玩耍"
      }
    ],
    "ratio": "1:1",
    "resolution": "480p",
    "duration": 1,
    "fps": 16,
    "seed": 42
  }'
```

**响应：**

```JSON
{
  "id": "cgt-20250327-a1b2c3d4",
  "status": "queued"
}
```

#### **2. 查询任务状态**

**请求：**

```bash
curl -X GET https://baotaai.bedicloud.net/api/v1/contents/generations/tasks/cgt-20260417-a0937bb7 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY"
```

**响应：**

```JSON
{
  "id": "cgt-20250327-a1b2c3d4",
  "model": "wan2.2-t2v",
  "status": "succeeded",
  "created_at": 1711536000,
  "updated_at": 1711539600,
  "seed": 42,
  "resolution": "480p",
  "ratio": "16:9",
  "duration": 5,
  "framespersecond": 16,
  "content": {
    "video_url": "http://localhost:8091/files/a1b2c3d4e5f67890.mp4"
  },
  "usage": {
    "completion_tokens": 0,
    "total_tokens": 0
  }
}
```

状态值：`queued`、`running`、`succeeded`、`failed`

#### **3. 列出所有任务**

**请求：**

```bash
curl -X GET 'https://baotaai.bedicloud.net/api/v1/contents/generations/tasks?page_num=1&page_size=10' \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY"
```

#### **4. 删除/取消任务**

**请求：**

```bash
curl -X DELETE https://baotaai.bedicloud.net/api/v1/contents/generations/tasks/cgt-20260417-a0937bb7 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY"
```

#### **5. 健康检查**

**请求：**

```HTTP
GET /health
```

### **请求参数**

| 参数             | 类型   | 默认值   | 说明                            |
| ---------------- | ------ | -------- | ------------------------------- |
| `content[].text` | string | 必填     | 文本提示词                      |
| `resolution`     | string | `720p`   | 分辨率：`480p`、`720p`、`1080p` |
| `ratio`          | string | `16:9`   | 宽高比：`16:9`、`4:3`、`1:1`    |
| `duration`       | int    | `5`      | 视频时长（秒）                  |
| `fps`            | int    | `16`     | 帧率                            |
| `seed`           | int    | `42`     | 随机种子，`-1` 表示随机生成     |
| `model`          | string | 模型名称 |

**分辨率与尺寸**

| 分辨率 | 16:9      | 4:3       | 1:1       |
| ------ | --------- | --------- | --------- |
| 480p   | 864×480   | 736×544   | 640×640   |
| 720p   | 1248×740  | 1120×832  | 960×960   |
| 1080p  | 1920×1088 | 1664×1248 | 1440×1440 |

### **错误处理**

常见错误码：

| 状态码 | 错误信息                      | 说明                   |
| ------ | ----------------------------- | ---------------------- |
| 401    | Missing ARK API key           | 缺少 API Key           |
| 401    | Invalid API key               | API Key 格式或签名无效 |
| 403    | Forbidden                     | 无权访问该任务         |
| 404    | Task not found                | 任务不存在             |
| 400    | Cannot cancel running request | 无法取消正在运行的任务 |