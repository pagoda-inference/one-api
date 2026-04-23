# One-API (BEDI 宝塔) 部署流程

## 部署环境
- **集群**: 10.1.112.238:8443
- **命名空间**: baota
- **镜像**: 10.1.112.238:8443/baota/one-api:latest

## 计费模型重大更新 (2026-04-12)

### 变更说明

**废弃 ModelRatio 机制，改用 model_info 表中的 InputPrice/OutputPrice 直接计费**

#### 旧机制（已废弃）
- 计费使用 `options` 表中的 `ModelRatio` (JSON map)
- 模型名称需手动添加到 ModelRatio map 中
- 如果模型不在 map 中，使用默认值 ratio=30

#### 新机制
- 计费直接使用 `model_info` 表中的 `InputPrice` 和 `OutputPrice`（元/千token）
- 模型管理页面设置的价格自动用于计费
- 不再需要手动维护 ModelRatio map

#### 计费公式
```
quota = (promptTokens * InputPrice / 1000 + completionTokens * OutputPrice / 1000) * groupRatio
```

#### 受影响的文件
- `relay/controller/helper.go` - preConsumeQuota/postConsumeQuota 改为使用 InputPrice/OutputPrice
- `relay/controller/text.go` - 文本请求计费
- `relay/controller/audio.go` - 音频请求计费
- `relay/controller/image.go` - 图像请求计费
- `relay/billing/billing.go` - PostConsumeQuota 改为使用 InputPrice/OutputPrice
- `model/market.go` - 修复 CalculateQuota 的除以 1000 bug

#### 注意事项
1. 如果模型在 `model_info` 表中不存在，价格默认为 0
2. `groupRatio` 仍然作为全局调整因子保留
3. `ModelRatio` 配置项不再用于计费（但仍存在于 options 表中，可保留或清空）

## 完整部署命令

```bash
# 1. 构建并推送镜像
docker build --no-cache -f Dockerfile.k8s -t 10.1.112.238:8443/baota/one-api:latest . 2>&1 && docker push 10.1.112.238:8443/baota/one-api:latest 2>&1

# 2. 重启 deployment
kubectl rollout restart deployment -n baota one-api 2>&1

# 3. 等待新 pod 运行
sleep 50 && kubectl get pods -n baota 2>&1 | grep "one-api-"
```

## 一键部署脚本

```bash
# 完整构建+推送+部署
docker build --no-cache -f Dockerfile.k8s -t 10.1.112.238:8443/baota/one-api:latest . && docker push 10.1.112.238:8443/baota/one-api:latest && kubectl rollout restart deployment -n baota one-api && sleep 60 && kubectl get pods -n baota | grep "one-api-"
```

## 快速检查命令

```bash
# 查看 one-api pods
kubectl get pods -n baota 2>&1 | grep one-api

# 查看 deployment 状态
kubectl get deployment -n baota one-api

# 查看 pod 详细信息（如果有问题）
kubectl describe pod -n baota <pod-name>

# 查看 pod 日志
kubectl logs -n baota <pod-name> --tail=100

# 查看最近日志（实时）
kubectl logs -n baota <pod-name> -f --tail=50
```

## PVC 挂载问题处理

如果新 pod 一直是 `ContainerCreating` 状态，报 `Multi-Attach error for volume "pvc-xxx"`，说明旧 pod 持有 PVC：

```bash
# 查看所有 one-api 相关 pod
kubectl get pods -n baota 2>&1 | grep one-api

# 删除卡住的旧 pod（让它释放 PVC）
kubectl delete pod -n baota <old-pod-name> 2>&1

# 等待新 pod Running
sleep 30 && kubectl get pods -n baota 2>&1 | grep "one-api-" | grep -v "baota-one-api"
```

## 验证部署成功

```bash
# 检查 pod 状态
kubectl get pods -n baota 2>&1 | grep "one-api-" | grep -v "baota-one-api"

# 期望看到类似输出：
# one-api-xxxxx-xxxxx   1/1     Running            0                  <time>

# 健康检查
curl -s https://baotaai.bedicloud.net/api/status | head -20
```

## 回滚操作

```bash
# 查看历史版本
kubectl rollout history deployment -n baota one-api

# 回滚到上一个版本
kubectl rollout undo deployment -n baota one-api

# 回滚到指定版本
kubectl rollout undo deployment -n baota one-api --to-revision=<revision-number>

# 等待回滚完成
sleep 60 && kubectl get pods -n baota | grep "one-api-"
```

## 常见问题

### 新 pod 一直是 ContainerCreating
原因：PVC 被旧 pod 占用
解决：删除旧 pod 释放 PVC
```bash
kubectl delete pod -n baota <pod-name> 2>&1
```

### pod 一直 Terminating
```bash
# 强制删除
kubectl delete pod -n baota <pod-name> --grace-period=0 --force 2>&1
```

### 镜像拉取失败
检查镜像是否成功推送：
```bash
docker images | grep one-api
```

### 应用启动失败
检查日志：
```bash
kubectl logs -n baota <pod-name> --previous --tail=100
```

### 健康检查失败
```bash
# 手动测试健康端点
kubectl exec -n baota <pod-name> -- curl -s http://localhost:3000/api/status

# 检查配置
kubectl get configmap -n baota one-api-config -o yaml
kubectl get secret -n baota one-api-secret -o yaml
```

## 配置更新

修改配置后不需要重新构建镜像，直接更新 ConfigMap/Secret 然后重启：

```bash
# 更新 ConfigMap
kubectl apply -f deploy/k8s/configmap.yaml

# 更新 Secret
kubectl apply -f deploy/k8s/secret.yaml

# 重启 pod 使配置生效
kubectl rollout restart deployment -n baota one-api
```

## 资源调整

修改 deployment.yaml 中的 resources 字段后重新应用：
```bash
kubectl apply -f deploy/k8s/deployment.yaml
kubectl rollout restart deployment -n baota one-api
```
