# 模型列表

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