# 多模型对比

同一问题让不同模型回答，方便对比效果和成本。

---

## 为什么对比

| 考量 | 说明 |
|------|------|
| 效果 | 不同模型在不同任务上表现各异 |
| 成本 | 小模型便宜，大模型贵 |
| 速度 | 小模型响应更快 |
| 能力 | 复杂推理需要大模型 |

---

## 对比示例

```python
from openai import OpenAI
import json

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://baotaai.bedicloud.net/v1"
)

# 需要对比的模型
models = [
    "bedi/glm-4",
    "bedi/qwen2.5-72b-instruct",
    "bedi/deepseek-v3",
]

# 测试问题
question = "解释什么是量子计算，用外行人能懂的方式"

print(f"问题: {question}\n")

results = []
for model in models:
    print(f"正在测试 {model}...")
    response = client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": question}],
        temperature=0.7,
        max_tokens=500
    )

    answer = response.choices[0].message.content
    usage = response.usage

    results.append({
        "model": model,
        "answer": answer,
        "input_tokens": usage.prompt_tokens,
        "output_tokens": usage.completion_tokens,
    })

    print(f"✓ 完成 (输入:{usage.prompt_tokens} 输出:{usage.completion_tokens})\n")

# 打印对比结果
print("=" * 60)
for r in results:
    print(f"【{r['model']}】")
    print(f"Token: 输入={r['input_tokens']} 输出={r['output_tokens']}")
    print(f"回答:\n{r['answer']}")
    print("-" * 60)
```

---

## 成本对比

```python
# 模型价格（元/千token）
prices = {
    "bedi/glm-4": {"input": 0.001, "output": 0.001},
    "bedi/qwen2.5-72b-instruct": {"input": 0.004, "output": 0.008},
    "bedi/deepseek-v3": {"input": 0.001, "output": 0.002},
}

def calculate_cost(model: str, input_tokens: int, output_tokens: int) -> float:
    """计算单次调用成本"""
    if model not in prices:
        return 0
    p = prices[model]
    return (input_tokens / 1000 * p["input"] +
            output_tokens / 1000 * p["output"])

print("\n成本估算:")
for r in results:
    cost = calculate_cost(r["model"], r["input_tokens"], r["output_tokens"])
    print(f"  {r['model']}: ¥{cost:.6f}")
```

---

## 选择建议

| 场景 | 推荐模型 |
|------|---------|
| 简单问答、客服 | glm-4, qwen2.5-7b |
| 代码生成 | claude-3.5-sonnet, glm-4 |
| 复杂推理 | deepseek-v3, qwen2.5-72b |
| 快速响应 | glm-4-flash, qwen2.5-7b |

---

## 批量对比工具

```python
class ModelComparator:
    def __init__(self, api_key: str, base_url: str):
        self.client = OpenAI(api_key=api_key, base_url=base_url)

    def compare(self, question: str, models: list[str], **kwargs):
        """批量对比多个模型"""
        results = []
        for model in models:
            try:
                resp = self.client.chat.completions.create(
                    model=model,
                    messages=[{"role": "user", "content": question}],
                    **kwargs
                )
                results.append({
                    "model": model,
                    "answer": resp.choices[0].message.content,
                    "usage": resp.usage,
                    "success": True
                })
            except Exception as e:
                results.append({
                    "model": model,
                    "error": str(e),
                    "success": False
                })
        return results

# 使用
comparator = ModelComparator("YOUR_API_KEY", "https://baotaai.bedicloud.net/v1")
results = comparator.compare(
    "解释什么是微服务架构",
    ["bedi/glm-4", "bedi/qwen2.5-72b-instruct"]
)
```
