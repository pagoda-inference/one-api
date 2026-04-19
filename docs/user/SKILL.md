---
name: create-agent
description: Build modular AI agents using One-API with OpenAI-compatible SDK
metadata:
  version: 0.0.1
  homepage: https://github.com/pagoda-inference/one-api
---

# Build AI Agents with One-API

This skill helps you create **modular AI agents** using One-API as a self-hosted API gateway.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Your Application                  │
├─────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│  │   CLI/TUI  │  │  HTTP API   │  │   Python   │  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │
│         │                │                │         │
│         └────────────────┼────────────────┘         │
│                          ▼                          │
│              ┌───────────────────────┐              │
│              │      One-API          │              │
│              │  (OpenAI-compatible)   │              │
│              └───────────┬───────────┘              │
│                          ▼                          │
│              ┌───────────────────────┐              │
│              │   Channel Router     │              │
│              │   (Multi-model)       │              │
│              └───────────────────────┘              │
└─────────────────────────────────────────────────────┘
```

## Capabilities

- **OpenAI-compatible API** - Use standard OpenAI SDK
- **Function Calling** - Tool/function calling support
- **Streaming** - SSE stream responses
- **Multi-model Routing** - Single endpoint for multiple providers
- **Token Management** - Built-in quota tracking

---

## Quick Start

### Step 1: Get Your API Key

1. Log in to One-API
2. Go to **API Keys** → Create new key
3. Set permissions and expiration

### Step 2: Install SDK

```bash
pip install openai
npm install openai
```

### Step 3: Create Agent

```typescript
// agent.ts
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: process.env.ONE_API_KEY,
  baseURL: 'https://your-domain.com/v1'  // One-API base URL
});

// Simple agent with function calling
async function chat(message: string, tools?: any[]) {
  const response = await client.chat.completions.create({
    model: 'glm-4',
    messages: [{ role: 'user', content: message }],
    tools,
    stream: false
  });

  const choice = response.choices[0];
  if (choice.finish_reason === 'tool_calls') {
    const toolCall = choice.message.tool_calls![0];
    console.log(`Calling function: ${toolCall.function.name}`);
    console.log(`Arguments: ${toolCall.function.arguments}`);
    return { toolCall, arguments: JSON.parse(toolCall.function.arguments) };
  }

  return choice.message.content;
}
```

```python
# agent.py
from openai import OpenAI
import json

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://your-domain.com/v1"
)

def chat(message: str, tools: list = None):
    response = client.chat.completions.create(
        model="glm-4",
        messages=[{"role": "user", "content": message}],
        tools=tools,
        stream=False
    )

    choice = response.choices[0]
    if choice.finish_reason == "tool_calls":
        tool_call = choice.message.tool_calls[0]
        return {
            "tool": tool_call.function.name,
            "arguments": json.loads(tool_call.function.arguments)
        }

    return choice.message.content
```

---

## Function Calling (Tools)

One-API supports OpenAI's function calling via the `tools` parameter.

### Define Tools

```typescript
const tools = [
  {
    type: "function",
    function: {
      name: "get_weather",
      description: "Get current weather for a location",
      parameters: {
        type: "object",
        properties: {
          location: {
            type: "string",
            description: "City name, e.g. 'Beijing'"
          },
          unit: {
            type: "string",
            enum: ["celsius", "fahrenheit"]
          }
        },
        required: ["location"]
      }
    }
  }
];
```

### Handle Tool Calls

```typescript
async function handleToolCall(toolCall: any) {
  const { name, arguments: argsStr } = toolCall.function;
  const args = JSON.parse(argsStr);

  switch (name) {
    case "get_weather":
      return await getWeather(args.location, args.unit);
    case "calculate":
      return await calculate(args.expression);
    default:
      return { error: `Unknown tool: ${name}` };
  }
}

// In your chat function:
const result = await chat("What's the weather in Beijing?", tools);
if (result.toolCall) {
  const toolResult = await handleToolCall(result.toolCall);

  // Send result back to continue conversation
  const continued = await client.chat.completions.create({
    model: 'glm-4',
    messages: [
      { role: "user", content: "What's the weather in Beijing?" },
      { role: "assistant", tool_calls: [result.toolCall] },
      {
        role: "tool",
        tool_call_id: result.toolCall.id,
        content: JSON.stringify(toolResult)
      }
    ]
  });
}
```

### Tool Response Format

```json
{
  "messages": [
    {"role": "user", "content": "What's the weather in Beijing?"},
    {
      "role": "assistant",
      "tool_calls": [
        {
          "id": "call_123",
          "type": "function",
          "function": {
            "name": "get_weather",
            "arguments": "{\"location\": \"Beijing\"}"
          }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_123",
      "content": "{\"temperature\": 22, \"condition\": \"Sunny\"}"
    }
  ]
}
```

---

## Streaming Responses

```typescript
// Streaming chat
const stream = await client.chat.completions.create({
  model: 'glm-4',
  messages: [{ role: 'user', content: 'Tell me a story' }],
  stream: true,
  stream_options: { include_usage: true }
});

for await (const chunk of stream) {
  const content = chunk.choices[0]?.delta?.content;
  if (content) {
    process.stdout.write(content);
  }

  // Tool calls can also stream in chunks
  const toolCall = chunk.choices[0]?.delta?.tool_calls?.[0];
  if (toolCall) {
    console.log(`Streaming tool call: ${toolCall.function?.name}`);
  }
}
```

### SSE Stream Format

One-API uses Server-Sent Events (SSE) for streaming:

```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"glm-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"glm-4","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: [DONE]
```

---

## Multi-Model Routing

One-API routes requests by model name. Use different model names to route to different providers:

```typescript
// Route to different providers via model name
const models = {
  chat: 'glm-4',
  vision: 'glm-4v',
  embedding: 'bge-m3',
  rerank: 'rerank'
};

// One-API automatically routes based on model name
const response = await client.chat.completions.create({
  model: models.chat,
  messages: [{ role: 'user', content: 'Hello' }]
});
```

### Get Available Models

```bash
curl https://your-domain.com/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

```json
{
  "object": "list",
  "data": [
    {"id": "glm-4", "object": "model", "name": "GLM-4", "provider": "zhipuai"},
    {"id": "gpt-4o", "object": "model", "name": "GPT-4o", "provider": "openai"},
    {"id": "bge-m3", "object": "model", "name": "BGE-M3", "type": "embedding"}
  ]
}
```

---

## Embedding & Rerank

### Generate Embeddings

```typescript
const embedding = await client.embeddings.create({
  model: 'bge-m3',
  input: 'Hello, world!'
});

console.log(embedding.data[0].embedding); // Float32Array
```

### Semantic Rerank

```typescript
const rerank = await client.rerank.create({
  model: 'rerank',
  query: 'What is quantum computing?',
  documents: [
    'Quantum computing uses quantum mechanics',
    'Classical computers use binary bits',
    'AI is developing rapidly'
  ]
});

console.log(rerank.results);
// [{ index: 0, relevance_score: 0.95 }, ...]
```

---

## Error Handling

```typescript
try {
  const response = await client.chat.completions.create({
    model: 'glm-4',
    messages: [{ role: 'user', content: 'Hello' }]
  });
} catch (error) {
  if (error.status === 401) {
    console.error('Invalid API key');
  } else if (error.status === 429) {
    console.error('Rate limited - try again later');
  } else if (error.status === 403) {
    console.error('No permission for this model');
  } else {
    console.error(`Error: ${error.message}`);
  }
}
```

### Error Codes

| Code | Meaning |
|------|---------|
| 401 | Invalid API key |
| 403 | No permission for this model |
| 404 | Model not found |
| 429 | Rate limited (RPM/TPM exceeded) |
| 500 | Internal server error |
| 10000 | Insufficient quota |
| 10001 | API key disabled |

---

## Best Practices

### 1. Use Streaming for Better UX

```typescript
// Stream responses for real-time feedback
const stream = await client.chat.completions.create({
  model: 'glm-4',
  messages: [{ role: 'user', content: 'Write a story' }],
  stream: true
});

for await (const chunk of stream) {
  const text = chunk.choices[0]?.delta?.content;
  if (text) process.stdout.write(text);
}
```

### 2. Handle Tool Calls Properly

```typescript
// Always validate tool arguments before execution
const args = JSON.parse(toolCall.function.arguments);
if (!args.location) {
  return { error: 'Missing required parameter: location' };
}
```

### 3. Implement Retry Logic

```typescript
async function withRetry(fn, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (error.status === 429 || error.status >= 500) {
        await sleep(1000 * (i + 1));
        continue;
      }
      throw error;
    }
  }
}
```

---

## Resources

- [One-API GitHub](https://github.com/pagoda-inference/one-api)
- [API Documentation](./api.md)
- [Admin API](../ops/admin-api.md)
