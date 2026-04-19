# Claude Code 接入

Claude Code 是 Anthropic 官方推出的 CLI 工具，可直接在终端使用 Claude 模型进行编码辅助。本节介绍如何将其接入 BEDI API。

---

## 前提条件

- macOS 或 Linux 系统
- Node.js 18+
- BEDI API Key

---

## 安装配置

### 方式一：一键安装脚本

```bash
# 下载并执行安装脚本
curl -fsSL https://raw.githubusercontent.com/your-repo/config-scripts/main/claude-code-setup.sh | bash
```

安装过程会要求：
1. 输入 BEDI API Key
2. 选择默认模型
3. 自动配置环境变量

### 方式二：手动配置

#### 1. 全局安装 Claude Code

```bash
npm install -g @anthropic-ai/claude-code
```

#### 2. 配置环境变量

```bash
export ANTHROPIC_API_KEY="YOUR_BEDI_API_KEY"
export ANTHROPIC_API_BASE="https://baotaai.bedicloud.net/v1"
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

## 配置说明

### 模型选择

在 `~/.claudeirc.json` 中配置：

```json
{
  "model": "bedi/claude-3.5-sonnet",
  "api_key": "YOUR_API_KEY",
  "api_base": "https://baotaai.bedicloud.net/v1"
}
```

### 可用模型

| 模型 ID | 说明 |
|---------|------|
| bedi/claude-3.5-sonnet | Claude 3.5 Sonnet（推荐）|
| bedi/claude-3-opus | Claude 3 Opus |
| bedi/claude-3-haiku | Claude 3 Haiku（快速）|

### 代理设置

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
1. 使用 `claude-3-haiku` 模型（更快）
2. 检查网络延迟
3. 选择物理距离更近的接入点

**Q: 如何退出？**

A: 输入 `exit` 或按 `Ctrl+C`。
