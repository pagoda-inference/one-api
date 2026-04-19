# 流式输出处理

流式输出（Server-Sent Events）让内容实时显示，无需等待完整响应。

---

## 原理

```
请求 ──▶ 服务器 ──▶ [chunk1] ──▶ [chunk2] ──▶ [chunk3] ──▶ [DONE]
                              └── 边生成边返回 ──┘
```

服务器持续返回数据帧，前端实时渲染。

---

## Python 示例

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://baotaai.bedicloud.net/v1"
)

stream = client.chat.completions.create(
    model="bedi/glm-4",
    messages=[{"role": "user", "content": "写一个快速排序"}],
    stream=True
)

print("生成中: ", end="", flush=True)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
print()
```

---

## JavaScript/Node.js 示例

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: process.env.ONE_API_KEY,
  baseURL: 'https://baotaai.bedicloud.net/v1'
});

async function streamChat() {
  const stream = await client.chat.completions.create({
    model: 'bedi/glm-4',
    messages: [{ role: 'user', content: '写一个快速排序' }],
    stream: true
  });

  for await (const chunk of stream) {
    const content = chunk.choices[0]?.delta?.content;
    if (content) {
      process.stdout.write(content);
    }
  }
  console.log();
}

streamChat();
```

---

## SSE 原始格式

如果不使用 SDK，直接处理 SSE：

```javascript
const response = await fetch('https://baotaai.bedicloud.net/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${apiKey}`
  },
  body: JSON.stringify({
    model: 'bedi/glm-4',
    messages: [{ role: 'user', content: 'Hello' }],
    stream: true
  })
});

// 读取流
const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const chunk = decoder.decode(value);
  console.log('Received:', chunk);

  // 解析 SSE 格式: data: {"id":...,"choices":[...]}
  // 最后一行是: data: [DONE]
}
```

---

## 断连重连

```python
import time

def stream_with_retry(messages, max_retries=3):
    """带重试的流式请求"""
    for attempt in range(max_retries):
        try:
            stream = client.chat.completions.create(
                model="bedi/glm-4",
                messages=messages,
                stream=True
            )

            full_content = ""
            for chunk in stream:
                if chunk.choices[0].delta.content:
                    full_content += chunk.choices[0].delta.content
                    print(chunk.choices[0].delta.content, end="", flush=True)

            return full_content

        except Exception as e:
            print(f"\n连接断开 (attempt {attempt + 1}/{max_retries}): {e}")
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt  # 指数退避
                print(f"等待 {wait_time}s 后重试...")
                time.sleep(wait_time)
            else:
                print("已达到最大重试次数")
                raise

# 使用
messages = [{"role": "user", "content": "解释量子计算"}]
stream_with_retry(messages)
```

---

## 注意事项

1. **流式不支持并行 tool_calls** - 需要等流结束才能处理
2. **网络波动会导致断连** - 建议实现重试机制
3. **stream=True 时 usage 可能为 None** - 需要单独获取
