# Token 成本优化

---

## 成本构成

```
总成本 = (输入 Token 数 × 输入价格 + 输出 Token 数 × 输出价格) / 1000
```

---

## 模型选择策略

| 任务类型 | 推荐模型 | 原因 |
|---------|---------|------|
| 简单问答 | 小模型 (7B) | 速度快，成本低 |
| 文本格式化 | 小模型 | 不需要推理能力 |
| 代码生成 | 中等模型 (72B) | 需要更好的理解 |
| 复杂推理 | 大模型 | 效果好 |

---

## 上下文压缩

### 1. 摘要旧对话

```python
def summarize_if_needed(messages: list, max_history: int = 10):
    """如果对话历史过长，先摘要再继续"""
    if len(messages) <= max_history:
        return messages

    # 用模型摘要前面的对话
    older_messages = messages[:-max_history]
    summary_prompt = f"""请简要总结以下对话的主要内容：

{[m['content'] for m in older_messages]}

简洁摘要："""

    summary_response = client.chat.completions.create(
        model="bedi/glm-4",
        messages=[{"role": "user", "content": summary_prompt}]
    )

    summary = summary_response.choices[0].message.content

    return [
        {"role": "system", "content": f"之前的对话摘要: {summary}"}
    ] + messages[-max_history:]
```

### 2. 去除冗余

```python
def optimize_messages(messages: list) -> list:
    """优化消息列表"""
    optimized = []

    for msg in messages:
        # 跳过空消息
        if not msg.get('content'):
            continue

        # 跳过纯 system 重复
        if msg['role'] == 'system' and optimized and optimized[-1]['role'] == 'system':
            continue

        optimized.append(msg)

    return optimized
```

---

## 缓存技巧

### 相似问题缓存

```python
import hashlib

cache = {}

def cached_chat(prompt: str, model: str = "bedi/glm-4") -> str:
    """对相同问题使用缓存"""
    cache_key = hashlib.md5(f"{model}:{prompt}".encode()).hexdigest()

    if cache_key in cache:
        print("(使用缓存)")
        return cache[cache_key]

    response = client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": prompt}]
    )

    result = response.choices[0].message.content
    cache[cache_key] = result

    return result
```

---

## max_tokens 控制

```python
# 场景 1: 短回答
short_response = client.chat.completions.create(
    model="bedi/glm-4",
    messages=[{"role": "user", "content": "北京是哪个国家的？"}],
    max_tokens=50  # 简短回答
)

# 场景 2: 长回答
long_response = client.chat.completions.create(
    model="bedi/glm-4",
    messages=[{"role": "user", "content": "写一篇 500 字作文"}],
    max_tokens=800  # 足够长
)
```

---

## 成本监控

```python
def track_cost(response, model: str):
    """追踪单次请求成本"""
    prices = {
        "bedi/glm-4": {"input": 0.001, "output": 0.001},
        "bedi/qwen2.5-72b-instruct": {"input": 0.004, "output": 0.008},
    }

    usage = response.usage
    price = prices.get(model, {"input": 0, "output": 0})

    input_cost = usage.prompt_tokens / 1000 * price["input"]
    output_cost = usage.completion_tokens / 1000 * price["output"]
    total = input_cost + output_cost

    print(f"Token: {usage.prompt_tokens} in / {usage.completion_tokens} out")
    print(f"成本: ¥{total:.6f}")

    return total
```

---

## 最佳实践总结

| 方法 | 节省比例 | 适用场景 |
|------|---------|---------|
| 使用小模型 | 50-90% | 简单任务 |
| 设置 max_tokens | 20-50% | 可以预估长度时 |
| 上下文压缩 | 30-70% | 长对话 |
| 结果缓存 | 0-100% | 重复问题 |
| 批量处理 | 10-30% | 减少请求 overhead |
