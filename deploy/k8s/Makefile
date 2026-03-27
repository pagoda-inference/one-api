# One API K8s 部署 Makefile (私有云)

REGISTRY := 10.1.112.238:8443
NAMESPACE := baota
IMAGE_NAME := one-api
IMAGE_TAG := latest
IMAGE := $(REGISTRY)/$(NAMESPACE)/$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build push deploy delete status logs restart secret

# 构建镜像
build:
	docker build -f ../..//Dockerfile.k8s -t $(IMAGE) ../..
	docker tag $(IMAGE) $(IMAGE):$(shell git rev-parse --short HEAD)

# 推送镜像
push:
	docker push $(IMAGE)
	docker push $(IMAGE):$(shell git rev-parse --short HEAD)

# 登录镜像仓库
login:
	docker login $(REGISTRY)

# 创建 Secret (使用环境变量)
# 先设置环境变量: export ONEAPI_DB_PASSWORD=xxx 等
secret:
	@echo "Creating secrets from environment variables..."
	@kubectl create secret generic one-api-secret \
		--namespace=$(NAMESPACE) \
		--from-literal=ONEAPI_DB_PASSWORD=$$ONEAPI_DB_PASSWORD \
		--from-literal=ONEAPI_REDIS_PASSWORD=$$ONEAPI_REDIS_PASSWORD \
		--from-literal=ONEAPI_SESSION_SECRET=$$ONEAPI_SESSION_SECRET \
		--from-literal=ONEAPI_LARK_CLIENT_SECRET=$$ONEAPI_LARK_CLIENT_SECRET \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Secrets created successfully"

# 部署所有资源
deploy:
	kubectl apply -f namespace.yaml
	kubectl apply -f configmap.yaml
	kubectl apply -f secret.yaml
	kubectl apply -f deployment.yaml
	kubectl apply -f ingress.yaml

# 查看状态
status:
	kubectl get all,ingress -n $(NAMESPACE)

# 查看日志
logs:
	kubectl logs -f deployment/one-api -n $(NAMESPACE) --tail=100

# 重启
restart:
	kubectl rollout restart deployment/one-api -n $(NAMESPACE)

# 删除所有资源
delete:
	kubectl delete -f ingress.yaml -f deployment.yaml -f secret.yaml -f configmap.yaml -f namespace.yaml

# 等待部署完成
wait:
	kubectl wait --for=condition=available deployment/one-api -n $(NAMESPACE) --timeout=300s
