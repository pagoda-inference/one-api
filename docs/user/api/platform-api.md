# 平台操作 API

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