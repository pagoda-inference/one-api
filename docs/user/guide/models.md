# 模型广场

模型广场展示所有可用的 AI 模型，支持浏览、搜索和试用。

---

## 获取模型列表

模型列表通过 API 动态获取：

```bash
curl https://your-domain.com/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

响应示例：

```json
{
  "object": "list",
  "data": [
    {
      "id": "bedi/glm-4",
      "object": "model",
      "name": "GLM-4 对话模型",
      "provider": "bedi",
      "type": "chat"
    },
    {
      "id": "bedi/bge-m3",
      "object": "model",
      "name": "BGE-M3 向量化模型",
      "provider": "bedi",
      "type": "embedding"
    }
  ]
}
```

> [!NOTE]
> 具体模型列表由管理员配置，请以实际 API 返回为准。

---

## 筛选模型

### 按类型筛选

| 类型 | 说明 |
|:-----|:-----|
| chat | 文本对话生成 |
| vlm | 视觉模型（图文理解）|
| embedding | 文本向量化 |
| reranker | 文本重排序 |
| ocr | 文字识别/文档解析 |

### 按供应商筛选

支持按 **Provider（供应商）** 分组浏览。

---

## 搜索模型

在搜索框中输入关键词搜索：

- 模型名称
- 模型 ID
- 供应商名称

---

## 查看模型详情

点击模型卡片打开详情面板：

### 基本信息

| 字段 | 说明 |
|:-----|:-----|
| 模型 ID | 唯一标识，API 调用时使用 |
| 供应商 | 模型提供方 |
| 类型 | 模型分类 |
| 上下文长度 | 最大输入 Token 数 |
| 输入价格 | 元 / 千 Token |
| 输出价格 | 元 / 千 Token |

### 能力标签

模型支持的能力会以标签展示，如：`chat` `vision` `function_call` 等。

---

## 价格计算器

在模型详情页使用价格计算器：

1. 输入 **输入 Token 数**（如：1000）
2. 输入 **输出 Token 数**（如：500）
3. 系统自动计算预计消耗额度

**计算公式：**

```
消耗额度 = (输入 Token × 输入价格 + 输出 Token × 输出价格) / 1000
```

---

## 试用模型

部分模型支持免费试用：

| 试用条件 | 说明 |
|:---------|:-----|
| 试用额度 | 每个模型有限试用 Token 数量 |
| 试用次数 | 每个用户只能开启一次试用 |
| 有效期 | 试用额度有使用期限 |

### 试用流程

1. 点击模型卡片上的 **试用** 按钮
2. 阅读试用说明
3. 点击 **开启试用**
4. 使用试用额度进行测试

> [!NOTE]
> 试用额度用完或过期后，如需继续使用请充值。

---

## 使用示例

```python
import openai

openai.api_key = "your-api-key"
openai.api_base = "https://baotaai.bedicloud.net/v1"

response = openai.ChatCompletion.create(
    model="bedi/glm-4",
    messages=[
        {"role": "system", "content": "你是一个有用的助手"},
        {"role": "user", "content": "你好"}
    ],
    temperature=0.7
)

print(response.choices[0].message.content)
```
