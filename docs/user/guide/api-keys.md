# API Keys

API Key 是调用 API 的凭证，请妥善保管，切勿泄露。

---

## 创建 API Key

1. 进入 **API Keys** 页面
2. 点击 **创建新的 Key** 按钮
3. 填写 Key 名称
4. （可选）设置权限限制：

| 设置项 | 说明 | 默认值 |
|:------|:-----|:------|
| 允许模型 | 限制可访问的模型列表 | 全部 |
| RPM | 每分钟请求数上限 | 无限制 |
| TPM | 每分钟 Token 上限 | 无限制 |
| 并发数 | 同时存在的请求数 | 无限制 |

5. 点击 **确定** 完成创建

> [!WARNING]
> 创建后请**立即复制** Key，系统不会再显示完整 Key。

---

## 查看与管理

在 API Keys 列表中，每行显示一个 Key 的信息：

- **名称** - 自定义的 Key 名称
- **状态** - 启用（绿色）/ 禁用（红色）
- **已用额度** - 该 Key 累计消耗
- **剩余额度** - 剩余可用额度
- **创建时间** - 创建日期
- **最近使用** - 最后调用时间

### 操作按钮

| 操作 | 说明 |
|:----|:-----|
| 👁 显示/隐藏 | 查看或隐藏完整 Key |
| 📋 复制 | 一键复制 Key 到剪贴板 |
| ✏️ 编辑 | 修改 Key 名称、权限、状态 |
| 🗑 删除 | 删除该 Key（需确认） |

---

## 认证方式

在调用 API 时，在 HTTP Header 中携带认证信息：

```http
Authorization: Bearer YOUR_API_KEY
```

**示例：**

```bash
curl https://api.example.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"model": "glm-4", "messages": [{"role": "user", "content": "Hello"}]}'
```

---

## 限流说明

| 限制类型 | 全称 | 说明 |
|:--------|:-----|:-----|
| RPM | Requests Per Minute | 每分钟允许的请求次数 |
| TPM | Tokens Per Minute | 每分钟允许的 Token 数量 |
| 并发 | Concurrent Requests | 同时存在的请求数量上限 |

> [!DANGER]
> 超过限流时，API 返回 `429 Too Many Requests` 错误。

---

## 最佳实践

> [!TIP]
> - 每个应用使用独立的 Key，便于管理权限和监控
> - 定期轮换 Key，保障安全
> - 生产环境不要使用有无限额度的 Key
> - 设置合理的 RPM/TPM 防止意外消耗