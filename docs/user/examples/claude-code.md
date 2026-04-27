# Claude Code 接入

Claude Code 是 Anthropic 官方推出的 CLI 工具，可直接在终端使用 Claude 模型进行编码辅助。本节介绍如何将其接入 BEDI API。

---

## 前提条件

- macOS 或 Linux 系统
- Node.js 18+
- BEDI API Key

---

## 安装配置

### 方式一：手动配置

#### 1. 全局安装 Claude Code

```bash
npm install -g @anthropic-ai/claude-code
```

#### 2. 配置环境变量

```bash
export ANTHROPIC_API_KEY="YOUR_BEDI_API_KEY"
export ANTHROPIC_API_BASE="https://baotaai.bedicloud.net/anthropic"
```

#### 3. 验证配置

```bash
claude --version
```

---

## 使用示例

### 基础对话

```bash
# 启动交互式对话
claude

# 单次请求
claude "解释这段代码的作用"
```

### 代码审查

```bash
# 审查单个文件
claude --file src/utils.py "检查这个文件有什么问题"

# 审查变更
claude --diff "优化这段代码的性能"
```

### 代码生成

```bash
# 生成单元测试
claude --task "为这个函数生成单元测试" --file src/calculator.py

# 重构代码
claude --task "将这个类重构为函数式风格" --file src/legacy.py
```

---

## 支持的功能

### 基础对话 ✅

Claude Code 通过 BEDI API 的 Anthropic 兼容接口（`/v1/messages`）工作，支持以下核心功能：

| 功能 | 状态 | 说明 |
|------|------|------|
| 非流式对话 | ✅ | 返回标准 JSON |
| 流式对话（SSE） | ✅ | 完整 Anthropic SSE 事件格式 |
| 工具调用（Tools） | ✅ | 支持 function calling |
| 流式工具调用 | ✅ | `tool_use` + `input_json_delta` |
| 工具结果回调 | ✅ | 自动转换为 OpenAI tool 格式 |
| Token 计数 | ✅ | `/v1/messages/count_tokens` |
| System Array | ✅ | 支持 `system: [{type:"text", text:"..."}]` |
| Content Block Array | ✅ | 支持 `content: [{type:"text", text:"..."}]` |

### 模型配置

在 `~/.claude/settings.json` 中配置：

```json
{
  "model": "bedi/glm-4.7",
  "api_key": "YOUR_API_KEY",
  "api_base": "https://baotaai.bedicloud.net/anthropic",
  "ANTHROPIC_DEFAULT_HAIKU_MODEL": "bedi/glm-4.7",
  "ANTHROPIC_DEFAULT_SONNET_MODEL": "bedi/glm-4.7",
  "ANTHROPIC_DEFAULT_OPUS_MODEL": "bedi/glm-4.7",
  "CLAUDE_CODE_SUBAGENT_MODEL": "bedi/glm-4.7",
  "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1",
  "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
}
```

配置说明：
- `model`：默认使用的模型
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`：轻量快速任务
- `ANTHROPIC_DEFAULT_SONNET_MODEL`：日常编码任务（推荐主力模型）
- `ANTHROPIC_DEFAULT_OPUS_MODEL`：复杂重构、代码审查
- `CLAUDE_CODE_SUBAGENT_MODEL`：子代理使用的模型

### 可用模型

| 模型 ID | 说明 | 推荐场景 |
|---------|------|---------|
| bedi/glm-4.7 | 智谱 GLM-4.7（当前主力）| 默认主力模型 |
| bedi/kimi-k2.6 | 月之暗面 Kimi | 主力模型、通用对话 |
| bedi/minimax-m2.5 | MiniMax M2.5 | 主力模型 |
| bedi/deepseek-v4-flash | DeepSeek V4 Flash | 主力模型 |
| bedi/minimax-m2.7 | MiniMax M2.7 | 主力模型 |
| bedi/qwen3-32b | 通义千问 32B | 日常编码 |
| bedi/qwen3-235b-a22b | 通义千问 235B（昇腾）| 轻量任务（能力受限）|

> 注意：Claude Code 本身为 Claude 系列模型设计，接入 BEDI 后使用国产大模型替代，功能可能存在差异。

---

## 使用示例

### 基础对话

```bash
# 启动交互式对话
claude

# 单次请求
claude "解释这段代码的作用"
```

### 代码审查

```bash
# 审查单个文件
claude --file src/utils.py "检查这个文件有什么问题"

# 审查变更
claude --diff "优化这段代码的性能"
```

### 代码生成

```bash
# 生成单元测试
claude --task "为这个函数生成单元测试" --file src/calculator.py

# 重构代码
claude --task "将这个类重构为函数式风格" --file src/legacy.py
```

---

## 代理设置

如需通过代理访问：

```bash
export HTTPS_PROXY="http://proxy.example.com:8080"
claude
```

---

## 常见问题

**Q: 连接失败？**

A: 检查：
1. API Key 是否正确
2. 网络是否能访问 `baotaai.bedicloud.net`
3. API Key 是否有额度

**Q: 速度慢？**

A: 尝试：
1. 使用 `bedi/qwen3-32b`
2. 检查网络延迟
3. 选择物理距离更近的接入点

**Q: 如何退出？**

A: 输入 `exit` 或按 `Ctrl+C`。
