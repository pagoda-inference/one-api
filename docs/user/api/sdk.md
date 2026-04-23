# SDK 示例

### Python

```bash
pip install openai
```

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://baotaai.bedicloud.net/v1"
)

# 聊天
chat_response = client.chat.completions.create(
    model="bedi/glm-4.7",
    messages=[{"role": "user", "content": "你好"}]
)
print(chat_response.choices[0].message.content)

# Embedding
embedding_response = client.embeddings.create(
    model="bedi/bge-m3",
    input="Hello, world!"
)
print(embedding_response.data[0].embedding)
```

### JavaScript / TypeScript

```bash
npm install openai
```

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'YOUR_API_KEY',
  baseURL: 'https://baotaai.bedicloud.net/v1'
});

// 聊天
const chatResponse = await client.chat.completions.create({
  model: 'bedi/glm-4.7',
  messages: [{ role: 'user', content: '你好' }]
});
console.log(chatResponse.choices[0].message.content);

// Embedding
const embeddingResponse = await client.embeddings.create({
  model: 'bedi/bge-m3',
  input: 'Hello, world!'
});
console.log(embeddingResponse.data[0].embedding);
```