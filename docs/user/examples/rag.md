# RAG 检索系统

RAG（Retrieval-Augmented Generation）通过检索增强生成，结合知识库和 LLM 提供更准确的回答。

---

## 架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   文档      │ ──▶ │  Embedding  │ ──▶ │  向量数据库  │
│  (分段)     │     │    API      │     │   (存储)    │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                                 │
                                                 ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   用户问题   │ ──▶ │   Rerank   │ ◀── │   相似检索   │
└─────────────┘     │    API      │     └─────────────┘
                    └──────┬──────┘
                           ▼
                    ┌─────────────┐
                    │    LLM     │
                    │   (生成)    │
                    └─────────────┘
```

---

## Step 1: 文档Embedding

将文档分割后转为向量存储：

```python
from openai import OpenAI
import json

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://baotaai.bedicloud.net/v1"
)

def embed_documents(documents: list[str]) -> list[list[float]]:
    """将文档列表转为向量"""
    response = client.embeddings.create(
        model="bedi/bge-m3",
        input=documents
    )
    return [item.embedding for item in response.data]

# 示例文档
docs = [
    "One-API 是一个开源的 AI 模型网关。",
    "它支持 OpenAI、Anthropic 等多种模型。",
    "可以用于构建 RAG 系统。"
]

# 获取向量
embeddings = embed_documents(docs)
print(f"生成了 {len(embeddings)} 个向量，每个向量 {len(embeddings[0])} 维")
```

---

## Step 2: 语义检索

根据用户问题检索相关文档：

```python
def search_documents(query: str, documents: list[str], top_k: int = 3) -> list[dict]:
    """语义检索相关文档"""
    # 1. 将问题转为向量
    query_embedding = client.embeddings.create(
        model="bedi/bge-m3",
        input=query
    ).data[0].embedding

    # 2. 计算相似度（简化版，实际可用余弦相似度）
    similarities = []
    for i, doc_emb in enumerate(embeddings):
        # 这里用点积作为相似度度量
        sim = sum(q * d for q, d in zip(query_embedding, doc_emb))
        similarities.append((i, sim))

    # 3. 排序取 Top-K
    similarities.sort(key=lambda x: x[1], reverse=True)

    results = []
    for idx, score in similarities[:top_k]:
        results.append({
            "document": documents[idx],
            "score": score
        })

    return results

# 检索
query = "One-API 支持哪些模型？"
results = search_documents(query, docs)
for r in results:
    print(f"[{r['score']:.4f}] {r['document']}")
```

---

## Step 3: Rerank 重排序（可选）

对初步检索结果进行精细排序：

```python
def rerank_results(query: str, documents: list[str], top_k: int = 3) -> list[dict]:
    """使用 Rerank API 精细排序"""
    response = client.rerank.create(
        model="bedi/bge-reranker",
        query=query,
        documents=documents,
        top_n=top_k
    )

    results = []
    for item in response.results:
        results.append({
            "document": documents[item.index],
            "rerank_score": item.relevance_score
        })

    return results

# 重排
reranked = rerank_results(query, docs)
for r in reranked:
    print(f"[{r['rerank_score']:.4f}] {r['document']}")
```

---

## Step 4: 构建 RAG 回答

将检索结果注入 Prompt：

```python
def rag_answer(query: str, documents: list[str]) -> str:
    """RAG 增强回答"""
    # 1. 检索相关文档
    docs = search_documents(query, documents, top_k=3)

    # 2. 构建上下文
    context = "\n".join([f"- {d['document']}" for d in docs])

    # 3. 构建 Prompt
    prompt = f"""基于以下参考资料回答问题。如果资料不足，请如实说明。

参考资料：
{context}

问题：{query}

回答："""

    # 4. 调用 LLM
    response = client.chat.completions.create(
        model="bedi/glm-4",
        messages=[
            {"role": "user", "content": prompt}
        ],
        temperature=0.7
    )

    return response.choices[0].message.content

# 测试
answer = rag_answer("One-API 是什么？", docs)
print(answer)
```

---

## 完整示例

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://baotaai.bedicloud.net/v1"
)

# 1. 文档列表
documents = [
    "One-API 是一个开源的 AI 模型网关项目。",
    "它支持 OpenAI、Anthropic、Google、Cohere 等多种模型。",
    "One-API 提供 OpenAI 兼容的 API 接口。",
    "可以用于构建 RAG、Agent 等应用。",
    "支持多租户、配额管理、渠道管理等功能。"
]

# 2. 获取 Embedding
print("正在生成文档向量...")
doc_embeddings = client.embeddings.create(
    model="bedi/bge-m3",
    input=documents
).data

# 3. 用户问题
query = "One-API 能做什么？"
print(f"\n问题: {query}")

# 4. 检索
query_emb = client.embeddings.create(
    model="bedi/bge-m3",
    input=query
).data[0].embedding

# 简化相似度计算
scores = []
for i, doc_emb in enumerate(doc_embeddings):
    sim = sum(q * d for q, d in zip(query_emb, doc_emb.embedding))
    scores.append((i, sim, documents[i]))

scores.sort(key=lambda x: x[1], reverse=True)

print("\n检索结果:")
for idx, score, doc in scores[:3]:
    print(f"  [{score:.4f}] {doc}")

# 5. RAG 回答
context = "\n".join([f"- {d[2]}" for d in scores[:3]])
prompt = f"""基于以下参考资料回答问题：

{context}

问题：{query}

回答："""

response = client.chat.completions.create(
    model="bedi/glm-4",
    messages=[{"role": "user", "content": prompt}]
)

print(f"\nRAG 回答：{response.choices[0].message.content}")
```

---

## 常见问题

**Q: 向量维度不一致怎么办？**

A: 确保使用相同的 Embedding 模型。不同模型的向量维度不同，无法直接比较。

**Q: 检索结果不相关？**

A: 可以尝试：
1. 调整 chunk 大小（文档分段大小）
2. 使用 Rerank 进行二次排序
3. 尝试不同的 Embedding 模型
