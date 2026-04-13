# One API K8s 部署指南 (私有云)

## 环境信息
- 镜像仓库: `10.1.112.238:8443`
- 命名空间: `baota`
- K8s API: 已配置默认 secret

## 快速部署

### 1. 设置环境变量

```bash
# 数据库密码 (PostgreSQL)
export ONEAPI_DB_PASSWORD="你的数据库密码"

# Redis 密码
export ONEAPI_REDIS_PASSWORD="你的Redis密码"

# Session 密钥 (32位随机字符串)
export ONEAPI_SESSION_SECRET="你的随机字符串"

# 飞书 OAuth App Secret
export ONEAPI_LARK_CLIENT_SECRET="飞书AppSecret"
```

### 2. 构建并推送镜像

```bash
cd deploy/k8s

# 登录镜像仓库 (如果未登录)
docker login 10.1.112.238:8443

# 构建镜像
docker build -f ../../Dockerfile.k8s -t 10.1.112.238:8443/baota/one-api:latest ../../

# 推送镜像
docker push 10.1.112.238:8443/baota/one-api:latest
```

或使用 Makefile:

```bash
cd deploy/k8s
make login
make build
make push
```

### 3. 创建 Secret

```bash
cd deploy/k8s

# 创建 Secret (从环境变量)
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

### 4. 部署服务

```bash
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f ingress.yaml
```

或使用 Makefile:

```bash
make deploy
```

### 5. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n baota

# 查看日志
kubectl logs -f deployment/one-api -n baota

# 查看所有资源
kubectl get all,ingress -n baota
```

### 6. 重启服务

```bash
kubectl -n baota rollout restart deployment one-api
```

或使用 Makefile:

```bash
make restart
```

## 部署文件说明

| 文件 | 说明 |
|------|------|
| namespace.yaml | 命名空间 baota |
| configmap.yaml | 应用配置（非敏感），包含 Server、Database、Redis、OAuth 等配置 |
| secret.yaml | Secret 模板（包含占位符，需通过 kubectl create secret 创建） |
| deployment.yaml | Deployment + Service + PVC |
| ingress.yaml | Ingress 网关 |
| Makefile | 自动化部署脚本 |

## 配置说明

### 环境变量 (Secret)

| 变量 | 说明 |
|------|------|
| ONEAPI_DB_PASSWORD | PostgreSQL 数据库密码 |
| ONEAPI_REDIS_PASSWORD | Redis 密码 |
| ONEAPI_SESSION_SECRET | Session 加密密钥 |
| ONEAPI_LARK_CLIENT_SECRET | 飞书 OAuth App Secret |

### ConfigMap 配置

| 配置项 | 说明 |
|------|------|
| PORT | 服务端口 (默认 3000) |
| SERVER_ADDRESS | 服务器地址 (用于回调) |
| SQL_DSN | PostgreSQL 连接字符串 |
| REDIS_CONN_STRING | Redis 连接字符串 |
| LarkClientId | 飞书 OAuth App ID |
| THEME | 前端主题 |

## 故障排除

```bash
# Pod 日志
kubectl logs -n baota -l app=one-api

# 详细描述
kubectl describe pod -n baota <pod-name>

# 删除并重新创建 Secret
kubectl delete secret one-api-secret -n baota
kubectl create secret generic one-api-secret \
  --namespace=baota \
  --from-literal=ONEAPI_DB_PASSWORD=$ONEAPI_DB_PASSWORD \
  --from-literal=ONEAPI_REDIS_PASSWORD=$ONEAPI_REDIS_PASSWORD \
  --from-literal=ONEAPI_SESSION_SECRET=$ONEAPI_SESSION_SECRET \
  --from-literal=ONEAPI_LARK_CLIENT_SECRET=$ONEAPI_LARK_CLIENT_SECRET

# 强制重启
kubectl rollout restart deployment/one-api -n baota
```

## 数据库和 Redis

- PostgreSQL: `10.1.112.239:34001`
- Redis: `10.1.112.239:34000`

使用宝塔面板创建和管理数据库和 Redis 实例。

---

## 文档站部署 (docs-site)

文档使用 Docsify 构建，部署在 `/guide` 路径下。

### 构建并推送

```bash
# 构建 amd64 镜像 (Mac 需要指定平台)
docker build --platform=linux/amd64 -f Dockerfile.docs -t 10.1.112.238:8443/baota/docs-site:latest .

# 推送
docker push 10.1.112.238:8443/baota/docs-site:latest
```

### 部署

```bash
kubectl apply -f docs-site.yaml
```

### 验证

```bash
# 查看 Pod
kubectl get pods -n baota -l app=docs-site

# 查看日志
kubectl logs -n baota -l app=docs-site
```

### 访问

- 文档地址: https://baotaai.bedicloud.net/guide
- 源文件目录: `docs/`

### 更新流程

1. 修改 `docs/` 下的文档
2. 重新构建并推送镜像
3. 执行 `kubectl apply -f docs-site.yaml` 或 `kubectl rollout restart deployment/docs-site -n baota`
