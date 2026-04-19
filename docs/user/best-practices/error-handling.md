# 错误处理与重试

---

## 错误码

| HTTP 状态码 | 错误码 | 说明 | 处理建议 |
|-------------|--------|------|---------|
| 400 | - | 请求参数错误 | 检查请求格式 |
| 401 | - | API Key 无效或过期 | 检查或重新创建 Key |
| 403 | - | 无权限访问该模型 | 检查 Key 权限设置 |
| 404 | - | 模型不存在 | 确认模型 ID 正确 |
| 408 | - | 请求超时 | 减少 max_tokens 或稍后重试 |
| 413 | - | 请求体过大 | 减少输入文本长度 |
| 429 | - | 限流 (RPM/TPM) | 降低请求频率 |
| 500 | - | 服务器内部错误 | 联系技术支持 |
| 503 | - | 服务暂时不可用 | 稍后重试 |
| - | 10000 | 额度不足 | 充值 |
| - | 10001 | Key 被禁用 | 检查 Key 状态 |
| - | 10002 | Key 未启用 | 启用 Key |
| - | 10003 | 模型已下架 | 选择其他可用模型 |

---

## Python 重试示例

```python
import time
from openai import APIError, RateLimitError

def chat_with_retry(messages, model="bedi/glm-4", max_retries=3):
    """带指数退避的重试机制"""
    for attempt in range(max_retries):
        try:
            response = client.chat.completions.create(
                model=model,
                messages=messages
            )
            return response

        except RateLimitError:
            wait_time = 2 ** attempt  # 1s, 2s, 4s
            print(f"限流，等待 {wait_time}s...")
            time.sleep(wait_time)

        except APIError as e:
            if e.status_code >= 500:
                # 服务器错误，重试
                wait_time = 2 ** attempt
                print(f"服务器错误 ({e.status_code})，等待 {wait_time}s...")
                time.sleep(wait_time)
            else:
                # 客户端错误，不再重试
                print(f"客户端错误: {e}")
                raise

    raise Exception(f"达到最大重试次数 ({max_retries})")
```

---

## JavaScript 重试示例

```javascript
async function chatWithRetry(messages, model = 'bedi/glm-4', maxRetries = 3) {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      const response = await client.chat.completions.create({
        model,
        messages
      });
      return response;

    } catch (error) {
      if (error.status === 429) {
        // 限流 - 指数退避
        const waitTime = Math.pow(2, attempt) * 1000;
        console.log(`限流，等待 ${waitTime}ms...`);
        await new Promise(r => setTimeout(r, waitTime));

      } else if (error.status >= 500) {
        // 服务器错误 - 重试
        const waitTime = Math.pow(2, attempt) * 1000;
        console.log(`服务器错误 (${error.status})，等待 ${waitTime}ms...`);
        await new Promise(r => setTimeout(r, waitTime));

      } else {
        // 客户端错误 - 不重试
        console.error(`错误: ${error.message}`);
        throw error;
      }
    }
  }
  throw new Error(`达到最大重试次数 (${maxRetries})`);
}
```

---

## 死信处理

```python
def process_with_dead_letter(messages, fallback_model="bedi/glm-4"):
    """
    主模型失败时降级到备用模型
    """
    models = ["bedi/glm-4", "bedi/qwen2.5-7b-instruct"]

    last_error = None
    for model in models:
        try:
            response = client.chat.completions.create(
                model=model,
                messages=messages
            )
            return response
        except Exception as e:
            last_error = e
            print(f"{model} 失败: {e}")
            continue

    # 所有模型都失败
    print(f"所有模型均失败，最后错误: {last_error}")
    return None
```

---

## 超时处理

```python
from openai import Timeout

# 设置超时（单位：秒）
response = client.chat.completions.create(
    model="bedi/glm-4",
    messages=[{"role": "user", "content": "Hello"}],
    timeout=30  # 30 秒超时
)

# 或使用配置
client.timeout = httpx.Timeout(30.0, connect=10.0)
```

---

## 最佳实践总结

| 场景 | 建议 |
|------|------|
| 限流 (429) | 指数退避，不要立即重试 |
| 服务器错误 (5xx) | 可以重试，建议 3 次以内 |
| 客户端错误 (4xx) | 不要重试，检查请求 |
| 超时 | 设置合理 timeout，配合重试 |
| 全部失败 | 实现降级策略 |
