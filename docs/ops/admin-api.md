# 运维 API

运维人员可通过 API 调用获取系统运营数据、管理用户、渠道、模型等资源。

> [!NOTE]
> 以下 API 需要管理员权限（Admin Token 或 Admin Role >= 10）。

---

## 认证方式

### Admin Token（推荐）

可用于所有 Admin API（包括计量、用户、渠道、模型管理等）：

```http
Authorization: Bearer {your_admin_token}
```

### Admin Role

请求 header 中携带登录态，或通过 session 认证。

---

## 用户管理

### 获取用户列表

```
GET /api/admin/
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| limit | int | 每页数量（默认 20） |
| offset | int | 偏移量 |

**返回：**

```json
{
  "success": true,
  "data": {
    "users": [...],
    "total": 100,
    "limit": 20,
    "offset": 0
  }
}
```

### 搜索用户

```
GET /api/admin/search
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| keyword | string | 搜索关键词 |

### 获取单个用户

```
GET /api/admin/:id
```

### 创建用户

```
POST /api/admin/
```

### 更新用户

```
PUT /api/admin/:id
```

### 删除用户

```
DELETE /api/admin/:id
```

---

## 运营统计

### 运营概览

```
GET /api/admin/ops/stats
```

**返回：**

```json
{
  "success": true,
  "data": {
    "today_revenue": 1234.56,
    "today_usage_tokens": 987654,
    "today_tokens": 987654,
    "active_users": 42,
    "channel_health_rate": 95.5,
    "total_users": 1000,
    "total_channels": 10,
    "total_tokens": 98765432,
    "total_quota": 1234567890,
    "revenue_by_day": { "2026-04-01": 123.45 },
    "usage_by_model": { "glm-4": 50000 },
    "topup_by_day": { "2026-04-01": 123.45 }
  }
}
```

### 营收统计

```
GET /api/admin/ops/revenue
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| start | string | 开始时间 |
| end | string | 结束时间 |

**返回：**

```json
{
  "success": true,
  "data": {
    "daily": [
      { "date": "2026-04-01", "amount": 123.45, "count": 5, "quota": 10000 }
    ],
    "total_amount": 1234.56,
    "total_quota": 100000,
    "total_count": 50
  }
}
```

### 用量统计

```
GET /api/admin/ops/usage
```

**返回：**

```json
{
  "success": true,
  "data": {
    "by_model": [
      { "model_name": "glm-4", "prompt_tokens": 50000, "completion_tokens": 30000 }
    ]
  }
}
```

### 用户列表（运营视图）

```
GET /api/admin/ops/users
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| limit | int | 每页数量 |
| offset | int | 偏移量 |

**返回：**

```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": 1,
        "username": "zhangsan",
        "display_name": "张三",
        "email": "zhangsan@example.com",
        "quota": 123456,
        "role": 10,
        "status": 1,
        "usage_quota": 50000
      }
    ],
    "total": 100,
    "limit": 20,
    "offset": 0
  }
}
```

---

## 渠道管理

### 获取渠道列表

```
GET /api/channel/
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| limit | int | 每页数量 |
| offset | int | 偏移量 |

### 获取渠道健康状态

```
GET /api/admin/channels/health
```

**返回：**

```json
{
  "success": true,
  "data": {
    "channels": [
      {
        "id": 1,
        "name": "BEDI-集群1",
        "status": "1",
        "type": 50,
        "type_name": "OpenAI兼容",
        "provider": "bedi",
        "base_url": "https://api.bedicloud.net/v1",
        "success_rate": 99.5,
        "avg_latency": 150,
        "priority": 100,
        "is_enabled": true
      }
    ],
    "overall_health": 98.5,
    "enabled_count": 8,
    "total_count": 10
  }
}
```

### 测试渠道

```
GET /api/channel/test
```

### 测试单个渠道

```
GET /api/channel/test/:id
```

### 更新渠道余额

```
GET /api/channel/update_balance/:id
```

### 添加渠道

```
POST /api/channel/
```

### 更新渠道

```
PUT /api/channel/
```

### 删除渠道

```
DELETE /api/channel/:id
```

---

## 告警配置

### 获取告警配置

```
GET /api/admin/alerts/config
```

**返回：**

```json
{
  "success": true,
  "data": {
    "channel_failure_threshold": 10,
    "queue_utilization_alert": 80,
    "error_rate_alert": 1,
    "latency_threshold": 5000,
    "alert_email": "admin@example.com",
    "alert_webhook": "https://example.com/webhook",
    "enabled": true,
    "max_concurrent_requests": 100,
    "request_queue_timeout": 120,
    "health_check_interval": 60,
    "health_check_fail_threshold": 5,
    "circuit_breaker_threshold": 10,
    "circuit_breaker_timeout": 60,
    "relay_timeout": 120
  }
}
```

### 更新告警配置

```
PUT /api/admin/alerts/config
```

**请求体：**

```json
{
  "alert_email": "admin@example.com",
  "alert_webhook": "https://example.com/webhook",
  "enabled": true,
  "max_concurrent_requests": 100,
  "request_queue_timeout": 120
}
```

---

## 系统健康

### 获取系统健康状态

```
GET /api/admin/system/health
```

**返回：**

```json
{
  "success": true,
  "data": {
    "uptime": 86400,
    "queue": {
      "size": 50,
      "capacity": 200
    },
    "channels": [...],
    "circuit_breakers": [...],
    "config": {
      "max_concurrent": 100,
      "health_interval": 60,
      "cb_threshold": 10
    }
  }
}
```

---

## 模型管理

### 获取模型列表

```
GET /api/admin/models
```

### 获取模型类型

```
GET /api/admin/models/types
```

### 获取模型状态

```
GET /api/admin/models/statuses
```

### 获取单个模型

```
GET /api/admin/models/model
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| id | int | 模型 ID |

### 创建模型

```
POST /api/admin/models
```

### 更新模型

```
PUT /api/admin/models/model
```

### 删除模型

```
DELETE /api/admin/models/model
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| id | int | 模型 ID |

### 批量删除模型

```
POST /api/admin/models/batch-delete
```

---

## Provider 管理

### 获取 Provider 列表

```
GET /api/admin/providers
```

### 获取 Provider 状态

```
GET /api/admin/providers/statuses
```

### 获取单个 Provider

```
GET /api/admin/providers/:id
```

### 创建 Provider

```
POST /api/admin/providers
```

### 更新 Provider

```
PUT /api/admin/providers/:id
```

### 删除 Provider

```
DELETE /api/admin/providers/:id
```

---

## 通知管理

### 获取通知列表

```
GET /api/admin/notifications
```

### 创建通知

```
POST /api/admin/notifications
```

**请求体：**

```json
{
  "title": "系统维护通知",
  "content": "系统将于今晚22:00进行维护..."
}
```

### 删除通知

```
DELETE /api/admin/notifications/:id
```

---

## 报表导出

### 导出报表

```
GET /api/admin/reports/export
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| type | string | 报表类型：daily, weekly, monthly |
| start | string | 开始时间 |
| end | string | 结束时间 |

---

## Root 专属 API

> [!WARNING]
> 以下 API 需要 Root 权限（Role = 100）。

### 获取系统选项

```
GET /api/option/
```

### 更新系统选项

```
PUT /api/option/
```

**请求体：**

```json
{
  "key": "QuotaForNewUser",
  "value": "1000"
}
```

---

## 计量 API（Admin Token）

### 用量汇总

```
GET /api/admin/usage/summary
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| start | string | 开始时间（可选） |
| end | string | 结束时间（可选） |
| model | string | 按模型筛选（可选） |
| token | string | 按 Token 筛选（可选） |
| channel | int | 按渠道筛选（可选） |

**返回：**

```json
{
  "success": true,
  "data": {
    "total_quota": 123456789,
    "total_tokens": 98765432,
    "total_prompt_tokens": 60000000,
    "total_completion_tokens": 38765432,
    "period": {
      "start": 1711929600,
      "end": 1712256000
    }
  }
}
```

### 按用户统计

```
GET /api/admin/usage/by-users
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| start | string | 开始时间（可选） |
| end | string | 结束时间（可选） |

**返回：**

```json
{
  "success": true,
  "data": {
    "users": [
      {
        "user_id": 2,
        "username": "zhangsan",
        "display_name": "张三",
        "request_count": 150,
        "quota": 1234567,
        "prompt_tokens": 500000,
        "completion_tokens": 300000
      }
    ],
    "period": {
      "start": "2026-04-01",
      "end": "2026-04-09"
    }
  }
}
```

### 按模型统计

```
GET /api/admin/usage/by-models
```

**参数：**

| 参数 | 类型 | 说明 |
|:-----|:-----|:-----|
| start | string | 开始时间（可选） |
| end | string | 结束时间（可选） |

**返回：**

```json
{
  "success": true,
  "data": {
    "models": [
      {
        "model_name": "glm-4",
        "request_count": 200,
        "quota": 2345678,
        "prompt_tokens": 800000,
        "completion_tokens": 600000
      }
    ],
    "period": {
      "start": 0,
      "end": 0
    }
  }
}
```

---

## curl 示例

```bash
# 获取运营概览
curl -X GET "https://baotaai.bedicloud.net/api/admin/ops/stats" \
  -H "Authorization: Bearer {admin_token}"

# 按用户统计用量
curl -X GET "https://baotaai.bedicloud.net/api/admin/usage/by-users?start=2026-04-01&end=2026-04-09" \
  -H "Authorization: Bearer {admin_token}"

# 获取渠道健康状态
curl -X GET "https://baotaai.bedicloud.net/api/admin/channels/health" \
  -H "Authorization: Bearer {admin_token}"

# 获取系统选项（需 Root 权限）
curl -X GET "https://baotaai.bedicloud.net/api/option/" \
  -H "Authorization: Bearer {root_token}"
```

---

## 字段说明

| 字段 | 说明 |
|:-----|:-----|
| quota | 消耗额度（1 quota ≈ 1 token） |
| prompt_tokens | 输入 token 数 |
| completion_tokens | 输出 token 数 |
| total_tokens | prompt + completion |
| request_count | 请求次数 |
| success_rate | 请求成功率 (%) |
| avg_latency | 平均延迟 (ms) |
