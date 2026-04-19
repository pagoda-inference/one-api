# Key 安全

API Key 是访问凭证，泄露可能导致额度被盗用。

---

## 禁止硬编码

❌ **错误做法**：

```python
# 不要这样！
client = OpenAI(
    api_key="sk-xxxxxxxxxxxxx",
    base_url="https://baotaai.bedicloud.net/v1"
)
```

❌ **不要提交到 Git**：

```bash
# .gitignore 添加
*.env
config/secrets.json
```

---

## 环境变量管理

✅ **正确做法**：

```python
import os

client = OpenAI(
    api_key=os.environ.get("ONE_API_KEY"),
    base_url=os.environ.get("ONE_API_BASE", "https://baotaai.bedicloud.net/v1")
)
```

### 本地开发

```bash
# .env 文件
ONE_API_KEY=sk-xxxxxxxxxxxxx
ONE_API_BASE=https://baotaai.bedicloud.net/v1

# 加载环境变量（使用 python-dotenv）
# pip install python-dotenv
from dotenv import load_dotenv
load_dotenv()
```

### 生产环境

```bash
# Linux/Mac
export ONE_API_KEY="sk-xxxxxxxxxxxxx"

# Docker
# docker run -e ONE_API_KEY=sk-xxxx myapp

# Kubernetes
# kubectl create secret generic api-keys --from-literal=ONE_API_KEY=sk-xxxx
```

---

## 定期轮换

建议每 90 天更换一次 API Key：

```bash
# 1. 在控制台创建新 Key
# 2. 更新所有使用旧 Key 的地方
# 3. 确认新 Key 正常工作后，删除旧 Key
```

---

## 权限最小化

创建 Key 时只授权需要的模型：

```python
# 创建 Key 时设置权限
key = client.api_keys.create(
    name="production-key",
    allowed_models=["bedi/glm-4", "bedi/bge-m3"],
    rpm_limit=60,
    tpm_limit=100000
)
```

---

## 多 Key 管理

```python
import os
import random

class KeyManager:
    def __init__(self, keys: list[str]):
        self.keys = keys

    def get_key(self) -> str:
        """随机获取一个可用 Key"""
        return random.choice(self.keys)

    def get_client(self):
        """获取配置好的客户端"""
        return OpenAI(
            api_key=self.get_key(),
            base_url=os.environ.get("ONE_API_BASE")
        )

# 使用
keys = [
    os.environ.get("API_KEY_1"),
    os.environ.get("API_KEY_2"),
]
manager = KeyManager(keys)
client = manager.get_client()
```

---

## 安全检查清单

- [ ] Key 不在代码中硬编码
- [ ] Key 不提交到 Git
- [ ] 使用环境变量管理
- [ ] 生产环境使用单独的 Key
- [ ] 为不同应用创建不同的 Key
- [ ] 设置合理的 RPM/TPM 限制
- [ ] 定期检查 Key 使用情况
- [ ] 发现异常及时更换 Key

---

## 发现泄露怎么办

1. **立即停用** - 在控制台禁用该 Key
2. **检查用量** - 查看是否有异常调用
3. **创建新 Key** - 替换所有使用的地方
4. **联系支持** - 如有损失，联系技术支持
