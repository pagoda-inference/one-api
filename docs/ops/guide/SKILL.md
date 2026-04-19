---
name: one-api-ops
description: Operate and manage a One-API AI gateway deployment
metadata:
  version: 0.0.1
  homepage: https://github.com/pagoda-inference/one-api
---

# One-API Operations Guide

This skill helps operators manage and maintain a One-API deployment.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   One-API Gateway                    │
├─────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│  │   Model    │  │  Channel   │  │   Token    │  │
│  │  Management│  │  Management│  │  Management│  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │
│         │                │                │         │
│         └────────────────┼────────────────┘         │
│                          ▼                          │
│              ┌───────────────────────┐              │
│              │   Channel Router     │              │
│              │   (Provider Routing)  │              │
│              └───────────────────────┘              │
└─────────────────────────────────────────────────────┘
```

## Capabilities

- **Multi-tenant** - Platform → Company → Department → Team hierarchy
- **Multi-provider** - OpenAI, Anthropic, local models, and more
- **Quota Management** - Per-user, per-token, per-team allocation
- **Model Routing** - Priority-based channel selection
- **Observability** - Logs, metrics, health checks

---

## Key Concepts

### Model Visibility

Models can be restricted to specific teams via `visible_to_teams`:

```sql
-- Public model (all users can see)
visible_to_teams = ""

-- Restricted to teams 1, 2
visible_to_teams = ",1,2,"
```

### Channel vs Provider

- **Channel** - A configured API endpoint with credentials
- **Provider** - A logical grouping of channels (e.g., "openai", "zhipuai")

Channels are tagged with a `provider` field for organization.

### Routing Flow

```
Request → ListModels() → visible_to_teams filtering
                ↓
v1/chat/completions → CacheGetRandomSatisfiedChannelByModel()
                ↓
         Match by model name
                ↓
         Select channel by priority
```

---

## Admin API

### Authentication

All admin APIs support API Key authentication:

```http
Authorization: Bearer {admin_token}
```

### Channel Management

```bash
# List channels
curl https://your-domain.com/api/channel/ \
  -H "Authorization: Bearer {admin_token}"

# Add channel
curl -X POST https://your-domain.com/api/channel/ \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My OpenAI Channel",
    "type": 50,
    "key": "sk-...",
    "base_url": "https://api.openai.com/v1",
    "models": "gpt-4,gpt-3.5-turbo",
    "provider": "openai"
  }'

# Test channel
curl https://your-domain.com/api/channel/test/:id \
  -H "Authorization: Bearer {admin_token}"

# Update channel balance
curl https://your-domain.com/api/channel/update_balance/:id \
  -H "Authorization: Bearer {admin_token}"
```

### Model Management

```bash
# List models
curl https://your-domain.com/api/admin/models \
  -H "Authorization: Bearer {admin_token}"

# Create model
curl -X POST https://your-domain.com/api/admin/models \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "glm-4",
    "name": "GLM-4",
    "provider": "zhipuai",
    "model_type": "chat",
    "status": "active"
  }'

# Update model (e.g., set trial availability)
curl -X PUT https://your-domain.com/api/admin/models/model \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "glm-4",
    "is_trial": true
  }'

# Set model visibility (visible_to_teams)
curl -X PUT https://your-domain.com/api/admin/models/model \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "glm-4",
    "visible_to_teams": ",1,2,"
  }'
```

### Usage Statistics

```bash
# Usage summary
curl "https://your-domain.com/api/admin/usage/summary" \
  -H "Authorization: Bearer {admin_token}"

# By users
curl "https://your-domain.com/api/admin/usage/by-users?start=2026-04-01&end=2026-04-09" \
  -H "Authorization: Bearer {admin_token}"

# By models
curl "https://your-domain.com/api/admin/usage/by-models" \
  -H "Authorization: Bearer {admin_token}"

# Operations stats
curl "https://your-domain.com/api/admin/ops/stats" \
  -H "Authorization: Bearer {admin_token}"

# Channel health
curl "https://your-domain.com/api/admin/channels/health" \
  -H "Authorization: Bearer {admin_token}"
```

### Tenant Management

```bash
# Create tenant
curl -X POST https://your-domain.com/api/tenant/ \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Team Alpha",
    "quota": 1000000
  }'

# Allocate quota to user
curl -X POST https://your-domain.com/api/tenant/:id/quota \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123,
    "quota": 500000
  }'
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_RETENTION_DAYS` | 90 | Days to keep logs (0 = no cleanup) |
| `SERVER_ADDRESS` | - | Public server address |
| `SYNC_FREQUENCY` | 600 | Channel balance sync interval (seconds) |

### Database Cleanup

Logs are automatically cleaned after `LOG_RETENTION_DAYS`. Before deletion, usage totals are archived to preserve historical statistics.

---

## Monitoring

### Health Checks

```bash
# System health
curl "https://your-domain.com/api/admin/system/health" \
  -H "Authorization: Bearer {admin_token}"

# Channel health status in response
{
  "data": {
    "channels": [
      {
        "id": 1,
        "name": "My Channel",
        "status": 1,
        "success_rate": 99.5,
        "avg_latency": 150
      }
    ]
  }
}
```

### Alerts

Configure alerts via `PUT /api/admin/alerts/config`:

```json
{
  "enabled": true,
  "alert_email": "ops@example.com",
  "alert_webhook": "https://hooks.example.com/alert",
  "channel_failure_threshold": 10,
  "error_rate_alert": 1,
  "latency_threshold": 5000
}
```

---

## Troubleshooting

### Channel Not Routing

1. Check channel status is `enabled` (status = 1)
2. Verify channel has the model in its `models` list
3. Test with `POST /api/channel/test/:id`
4. Check ability cache: restart may be needed after adding new models

### Model Not Visible

1. Check `status` is `active`
2. If `visible_to_teams` is set, ensure user belongs to that team
3. Check `model_info` table directly

### High Latency

1. Check channel health rates in `/api/admin/channels/health`
2. Verify upstream provider status
3. Consider adjusting `RelayTimeout` configuration
4. Check for channel balance depletion

---

## Resources

- [One-API GitHub](https://github.com/pagoda-inference/one-api)
- [Admin API Documentation](./admin-api.md)
- [User API Documentation](../user/api.md)
