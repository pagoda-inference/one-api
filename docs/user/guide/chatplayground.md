# 体验中心

体验中心提供交互式对话界面，无需编写代码即可测试模型效果。

---

## 功能入口

登录后，点击左侧菜单 **体验中心** 进入。

---

## 界面布局

体验中心采用三栏布局：

| 区域 | 说明 |
|:-----|:-----|
| 左侧面板 | 模型选择、参数配置 |
| 右侧对话 | 消息展示、流式输出 |
| 对比模式 | 双模型同时回复，便于对比效果 |

---

## 模型选择

体验中心提供以下 Trial 模型供免费测试：

| 模型 | 类型 | 说明 |
|:-----|:-----|:-----|
| bedi/qwen3-14b | 对话 | 基础文本对话 |
| bedi/qwen3-32b | 对话 + 推理 | 支持思维链（Enable Thinking）|
| bedi/qwen3-vl-8b | 图文 | 支持图片理解 |

> [!NOTE]
> 体验中心使用专属 Trial API Key，不消耗您的个人额度。

---

## 参数配置

### 通用参数

| 参数 | 说明 | 范围 |
|:-----|:-----|:-----|
| max_tokens | 最大输出 Token 数 | 256 ~ 32768 |
| temperature | 采样温度，影响随机性 | 0 ~ 2 |
| top_p | Nucleus 采样阈值 | 0 ~ 1 |
| top_k | Top-K 采样词数 | 1 ~ 100 |
| frequency_penalty | 频率惩罚（减少重复）| -2 ~ 2 |
| presence_penalty | 存在惩罚（鼓励新话题）| -2 ~ 2 |

### Qwen3-32B 专属参数

| 参数 | 说明 | 范围 |
|:-----|:-----|:-----|
| Enable Thinking | 开启思维链模式 | 开关 |
| Thinking Budget | 思维链最大 Token 数 | 512 ~ 32768 |

---

## 对比模式

点击 **+ 对比模式**，可同时选择两个模型进行对话，便于比较不同模型的回复效果。

- 左栏显示第一个模型的回复
- 右栏显示第二个模型的回复
- 两条回复同时生成，实时流式输出

---

## 使用示例

### 基础对话

1. 选择模型（如 `bedi/qwen3-14b`）
2. 调整参数（可使用默认参数）
3. 在输入框输入问题
4. 点击发送按钮或按 `Enter`
5. 等待流式回复完成

### 思维链演示（Qwen3-32B）

1. 选择 `bedi/qwen3-32b` 模型
2. 开启 **Enable Thinking** 开关
3. 调整 **Thinking Budget**（如 8192）
4. 发送复杂推理问题
5. 观察模型先输出思考过程，再输出最终答案

---

## 技术说明

体验中心通过 SSE（Server-Sent Events）实现流式输出，消息逐字显示，模拟打字机效果。

请求示例：

```bash
curl https://baotaai.bedicloud.net/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "bedi/qwen3-14b",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```
