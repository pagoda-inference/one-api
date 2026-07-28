# 视频系列（Video）

本平台提供两套等价的视频生成 API：

1. **OpenAI 兼容接口（推荐）**：`/v1/videos`、`/v1/videos/sync`，以及 `GET/DELETE /v1/videos/:id`、`GET /v1/videos/:id/content`（同源视频代理）、`GET /v1/videos`。
   行为与 OpenAI 风格一致，支持 `Bearer` 鉴权、按模型自动选择渠道，并在 `POST /v1/videos`
   上具备与文本模型相同的失败重试能力（429/5xx 自动换渠道重试）。
2. **Ark 风格接口（兼容保留）**：`/api/v1/contents/generations/tasks`，用于已有客户端的平滑迁移。

两套接口共享同一套任务存储、配额结算与渠道调度逻辑，任务 ID 互通。

---

## 鉴权

所有视频接口均使用 OpenAI 风格的 API Key：

```
Authorization: Bearer sk-xxxxxxxx
```

API Key 可在「令牌」页面创建。令牌可绑定可用模型范围；调用 `POST /v1/videos` 时，
请求体中的 `model` 必须在令牌允许的模型列表内（与文本模型一致）。

---

## 一、OpenAI 兼容接口

### 1. 创建视频生成任务（异步）

```
POST /v1/videos
```

异步提交一个视频生成任务，立即返回任务 ID。任务完成后，通过 `GET /v1/videos/:id`
轮询获取视频地址，或改用同步接口 `POST /v1/videos/sync`。

**请求体**

支持两种写法：

- **简洁形式**（文生视频）：`{"model","prompt"}`
- **简洁形式**（图生视频）：`{"model","image_url"}`
- **完整形式**（多模态输入）：`{"model","content":[{"type":"text",...},{"type":"image_url",...}]}`

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 必填 | 模型名称，如 `wan2.2-t2v`、`wan2.2-i2v`。未提供时按输入类型自动推断 |
| `prompt` | string | - | 文本提示词（t2v）。当未提供 `content` 时使用 |
| `image_url` | string | - | 参考图片 URL（i2v）。当未提供 `content` 时使用 |
| `content` | array | - | 多模态输入数组，元素 `type` 为 `text` 或 `image_url`；提供后将忽略 `prompt`/`image_url` |
| `content[].type` | string | - | `text` 或 `image_url` |
| `content[].text` | string | - | `type=text` 时的文本提示词 |
| `content[].image_url.url` | string | - | `type=image_url` 时的图片 URL |
| `resolution` | string | `720p` | 分辨率：`480p`、`720p`、`1080p` |
| `ratio` | string | `16:9` | 宽高比：`16:9`、`4:3`、`1:1` |
| `duration` | int | `5` | 视频时长（秒），需 > 0 |
| `fps` | int | `16` | 帧率，需 > 0 |
| `seed` | int | `42` | 随机种子 |

> 文生视频（t2v）与图生视频（i2v）的判定：只要 `content`（或 `image_url`）中存在图片输入即判定为 i2v，
> 同时存在文本与图片的混合输入也视为 i2v（上游支持“图片 + 文本描述”）。

**示例：文生视频（简洁形式）**

```bash
YOUR_API_KEY=sk-xxx
curl -s https://your-host/v1/videos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY" \
  -d '{
    "model": "wan2.2-t2v",
    "prompt": "写实风格，一只猫咪在玩耍",
    "ratio": "16:9",
    "resolution": "720p",
    "duration": 5,
    "fps": 16,
    "seed": 42
  }'
```

**示例：图生视频（完整 content 形式）**

```bash
curl -s https://your-host/v1/videos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY" \
  -d '{
    "model": "wan2.2-i2v",
    "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/cat.png"}},
      {"type": "text", "text": "让猫咪转头看向镜头"}
    ],
    "ratio": "1:1",
    "resolution": "720p"
  }'
```

**响应（202 Accepted）**

```json
{
  "id": "cgt-20250327-a1b2c3d4",
  "object": "video",
  "model": "wan2.2-t2v",
  "status": "queued",
  "created_at": 1711536000
}
```

### 2. 同步创建视频生成任务

```
POST /v1/videos/sync
```

请求体与异步接口相同，额外支持以下同步控制参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `timeout` | int | `300` | 同步等待最大秒数，超时返回 408 |
| `poll_interval` | int | `5` | 轮询上游的间隔秒数，最小 3 |

行为：创建任务后阻塞轮询，直到任务进入 `succeeded`/`failed` 或到达 `timeout`。
到达终态返回完整视频对象（含 `video.url`）；超时返回 408 并附带当前任务状态。

**响应（200 OK，成功）**

```json
{
  "id": "cgt-20250327-a1b2c3d4",
  "object": "video",
  "model": "wan2.2-t2v",
  "status": "succeeded",
  "created_at": 1711536000,
  "updated_at": 1711539600,
  "seed": 42,
  "resolution": "720p",
  "ratio": "16:9",
  "duration": 5,
  "frames_per_second": 16,
  "video": {
    "url": "https://your-host/files/a1b2c3d4e5f67890.mp4"
  },
  "usage": {
    "completion_tokens": 0,
    "total_tokens": 0
  }
}
```

### 3. 查询单个任务

```
GET /v1/videos/:id
```

返回当前任务状态（若上游渠道可用，会自动刷新并结算配额）。

**响应**：同「同步创建」的成功响应对象。

### 3.1 获取视频内容（同源代理）

```
GET /v1/videos/:id/content
```

通过同源端点流式代理生成的视频字节流，避免浏览器混合内容 / CORS 问题。任务状态为
`succeeded` 后可用。支持以下能力：

- `Range` 请求（HTTP 分段加载/拖动播放）
- `?download=1` 触发下载（自动设置 `Content-Disposition`，扩展名按 Content-Type 推断，默认 `.mp4`）
- 自动透传 `Content-Type` / `Content-Length` / `Accept-Ranges` / `Content-Range`

**响应**：直接返回视频字节流（`Content-Type: video/mp4` 等），非 JSON。

| 状态码 | 说明 |
| --- | --- |
| 200 | 成功，返回视频字节流 |
| 400 | 视频未就绪（`video not ready`）或 URL 非法 |
| 404 | 任务不存在或不属于当前用户 |
| 502 | 拉取上游视频失败 |

### 4. 列出任务

```
GET /v1/videos?page=1&page_size=20
```

支持 `page`/`page_size` 或 OpenAI 风格的 `limit` 参数（`page_size` 优先），最大 100。

**响应（200 OK）**

```json
{
  "object": "list",
  "data": [
    { "id": "...", "object": "video", "model": "...", "status": "succeeded", ... }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

### 5. 删除 / 取消任务

```
DELETE /v1/videos/:id
```

- 任务处于 `running` 时不可取消（返回 400）。
- 其余状态下结算配额后从列表删除。

**响应（200 OK）**

```json
{
  "id": "cgt-20250327-a1b2c3d4",
  "object": "video",
  "deleted": true,
  "status": "succeeded"
}
```

---

## 二、Ark 风格接口（兼容保留）

> 与 OpenAI 兼容接口共享任务存储，任务 ID 互通。新接入建议使用 `/v1/videos`。

### 1. 创建任务

```
POST /api/v1/contents/generations/tasks
```

**请求体**：使用 `content[]` 多模态形式（与 OpenAI 接口的 `content[]` 相同）：

```bash
curl -s https://your-host/api/v1/contents/generations/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $YOUR_API_KEY" \
  -d '{
    "model": "wan2.2-t2v",
    "content": [{"type": "text", "text": "写实风格，一只猫咪在玩耍"}],
    "ratio": "1:1",
    "resolution": "480p",
    "duration": 5,
    "fps": 16,
    "seed": 42
  }'
```

**响应（200 OK）**

```json
{
  "id": "cgt-20250327-a1b2c3d4",
  "status": "queued"
}
```

### 2. 查询任务

```
GET /api/v1/contents/generations/tasks/:id
```

```json
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

### 3. 列出任务

```
GET /api/v1/contents/generations/tasks?page_num=1&page_size=10
```

### 4. 删除 / 取消任务

```
DELETE /api/v1/contents/generations/tasks/:id
```

### 5. 同源视频代理

```
GET /api/v1/contents/generations/tasks/:id/content
GET /api/user/market/video/tasks/:id/content
```

通过同源端点代理视频字节流，避免浏览器混合内容/CORS 问题。支持 `Range` 请求与 `?download=1` 下载。

---

## 通用说明

### 状态值

| 状态 | 说明 |
| --- | --- |
| `queued` | 任务已入队，尚未开始 |
| `running` | 生成中 |
| `succeeded` | 生成成功，可取 `video.url` |
| `failed` | 生成失败 |

> 视频文件（`video.url`）通常保留 3 小时，之后链接失效，请及时下载或使用同源代理端点。

### 分辨率与尺寸矩阵

| 分辨率 | 16:9 | 4:3 | 1:1 |
| --- | --- | --- | --- |
| 480p | 864×480 | 736×544 | 640×640 |
| 720p | 1248×704 | 1120×832 | 960×960 |
| 1080p | 1920×1088 | 1664×1248 | 1440×1440 |

### 渠道选择与重试

- `/v1/videos` 与 `/v1/videos/sync` 走标准的 `Distribute` 中间件：按 `model` 在所有启用的
  渠道（OpenAI 兼容、火山方舟视频、自定义视频）中自动选择。
- `POST /v1/videos`（async）与 `POST /v1/videos/sync`（sync）经 `controller.Relay` 调度，
  对上游 429/5xx 自动换渠道重试，与文本模型行为一致。
- `GET/DELETE/LIST` 操作已持久化的任务，渠道固定，不重试。

---

## 错误处理

所有错误均以 OpenAI 风格返回：

```json
{
  "error": {
    "message": "错误描述",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

### 错误码

| 状态码 | type | code | 说明 | 处理建议 |
| --- | --- | --- | --- | --- |
| 400 | `invalid_request_error` | `invalid_request` | 请求参数错误（缺 content、非法 resolution/ratio 等） | 检查请求体格式与参数 |
| 400 | `invalid_request_error` | `model_required` | 未提供 `model` | 提供 `model` 字段 |
| 400 | `invalid_request_error` | `cannot_cancel_running` | 任务运行中无法取消 | 等待任务结束后再删除 |
| 401 | `authentication_error` | - | API Key 无效或缺失 | 检查/重建 API Key |
| 403 | `forbidden_error` | - | 令牌无权使用该模型 / 用户被封禁 | 检查令牌模型权限 |
| 403 | `forbidden_error` | `insufficient_quota` | 配额不足 | 充值或降低分辨率/时长 |
| 404 | `not_found_error` | `task_not_found` | 任务不存在或不属于当前用户 | 核对任务 ID |
| 408 | `invalid_request_error` | `video_sync_timeout` | 同步等待超时 | 增大 `timeout` 或改用异步轮询 |
| 429 | `rate_limit_error` | `upstream_429` | 上游限流（已自动重试） | 稍后重试 |
| 500 | `server_error` | `persist_task_failed` 等 | 服务器内部错误 | 联系技术支持 |
| 502 | `server_error` | `upstream_5xx` / `invalid_upstream_response` | 上游异常或响应非法 | 稍后重试或切换渠道 |
| 503 | `server_error` | `no_available_channel` | 无可用视频渠道 | 配置对应模型渠道并启用 |
