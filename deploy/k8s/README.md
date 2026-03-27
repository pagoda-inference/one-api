# One API K8s 部署指南 (私有云)

## 环境信息
- 镜像仓库: `10.1.112.238:8443`
- 命名空间: `baota`
- K8s API: 已配置默认 secret

## 部署步骤

### 1. 准备数据库

使用宝塔面板创建 PostgreSQL 数据库：
- 主机: 10.1.112.239:34001
- 用户: postgres
- 密码: (设置并记录)

### 2. 准备 Redis

使用宝塔面板创建 Redis：
- 主机: 10.1.112.239:34000
- 密码: (设置并记录)

### 3. 配置飞书 OAuth

在飞书开放平台创建应用：
1. 创建企业自建应用 → 开启「网页登录」
2. 配置回调地址：`https://你的域名/oauth/lark`
3. 获取 App ID 和 App Secret

### 4. 构建并推送镜像

```bash
# 登录镜像仓库
docker login 10.1.112.238:8443

# 构建镜像
docker build -f Dockerfile.k8s -t 10.1.112.238:8443/baota/one-api:latest .

# 推送镜像
docker push 10.1.112.238:8443/baota/one-api:latest
```

### 5. 部署服务

#### 5.1 创建 Secret (敏感信息)

```bash
# 设置环境变量
export ONEAPI_DB_PASSWORD="你的数据库密码"
export ONEAPI_REDIS_PASSWORD="你的Redis密码"
export ONEAPI_SESSION_SECRET="32位随机字符串"
export ONEAPI_LARK_CLIENT_SECRET="飞书AppSecret"

# 创建 Secret
kubectl create secret generic one-api-secret \
  --namespace=baota \
  --from-literal=ONEAPI_DB_PASSWORD=$ONEAPI_DB_PASSWORD \
  --from-literal=ONEAPI_REDIS_PASSWORD=$ONEAPI_REDIS_PASSWORD \
  --from-literal=ONEAPI_SESSION_SECRET=$ONEAPI_SESSION_SECRET \
  --from-literal=ONEAPI_LARK_CLIENT_SECRET=$ONEAPI_LARK_CLIENT_SECRET
```

或使用 Makefile:
```bash
make secret
```

#### 5.2 部署其他资源

```bash
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f ingress.yaml

# 查看状态
kubectl get pod -n baota

# 查看日志
kubectl logs -f deployment/one-api -n baota
```

或使用 Makefile:
```bash
make deploy
```

### 6. 配置 DNS

联系网络管理员配置 DNS，或在 hosts 添加：
```
10.1.112.237 one-api.your-domain.com
```

## 部署文件说明

| 文件 | 说明 |
|------|------|
| namespace.yaml | 命名空间 baota |
| configmap.yaml | 应用配置（非敏感） |
| secret.yaml | Secret 模板（需通过 kubectl create secret 创建） |
| deployment.yaml | Deployment + Service + PVC |
| ingress.yaml | Ingress 网关 |

## 验证

```bash
# 检查所有资源
kubectl get all,ingress -n baota

# 测试 API
curl http://localhost/api/status

# 浏览器访问
# http://one-api.your-domain.com
```

## 故障排除

```bash
# Pod 日志
kubectl logs -n baota -l app=one-api

# 详细描述
kubectl describe pod -n baota <pod-name>

# 重启
kubectl rollout restart deployment/one-api -n baota
```

## Secret 管理说明

为保证安全，敏感信息（密码、密钥）通过 K8s Secret 管理，不存储在 Git 中。

创建 Secret 的两种方式：
1. **环境变量方式** (推荐):
   ```bash
   export ONEAPI_DB_PASSWORD="xxx"
   kubectl create secret generic one-api-secret --from-literal=ONEAPI_DB_PASSWORD=$ONEAPI_DB_PASSWORD ...
   ```

2. **Sealed Secrets** (生产环境推荐):
   使用 bitnami-labs/sealed-secrets 将 Secret 加密存储在 Git 中。
