# One API 二次开发 DEV PLAN

模型 API 聚合服务网关 · 技术实施方案

**版本 v1.3 · 2026 年 3 月 · 对标 n1n.ai**

---

## 1. 项目背景与目标

### 1.1 背景

当前团队拥有三个 GPU 云平台的 API-Key 资源（H800 北广 AI 云、910B3 移动平台、910B4 佛山 AI 云），后端均基于 vLLM 引擎，提供 OpenAI 兼容接口及 Anthropic 格式接口。

目标是以 One API 为基础进行二次开发，对外提供统一的模型 API 聚合服务，客户无需在各云平台注册，统一使用团队分发的 Key 接入。

### 1.2 总体架构

```
架构层级：
客户 → One API（对外层：Key 分发 / 限流 / 计费 / 格式标准化）
One API → Mass 平台（内部路由层：透传 + 内部 Token 统计）
Mass 平台 → vLLM × 3（H800 / 910B3 / 910B4）
One API → 三方 API（OpenAI 官方 / Anthropic 官方 / 其他）
```

### 1.3 核心交付目标

- 对外暴露统一 API 端点，兼容 OpenAI 格式与 Anthropic 格式
- 多平台渠道路由、负载均衡、自动故障切换
- 客户 Key 分发、额度管理、RPM/TPM 精细限流
- Token 级用量统计，支持按 Key / 模型 / 时间维度查询
- 批量推理 Batch API（独立服务，复用鉴权体系）
- 接入三方 API 渠道（OpenAI、Anthropic 官方等）

### 1.4 竞品分析：n1n.ai

**n1n.ai 核心功能**：

| 功能模块 | n1n.ai | 我们的实现 | 状态 |
|----------|--------|------------|------|
| LLM API 网关 | ✅ | One API | ✅ 已完成 |
| 全球500+模型 | ✅ | 50+ | ⚠️ 需扩展 |
| 人民币直付（1元=1美元） | ✅ | ❌ | 🔴 P4 |
| 模型市场展示 | ✅ | ❌ | 🔴 P5 |
| 用户Dashboard | ✅ | ⚠️ 基础 | ⚠️ P5 |
| 运营看板 | ✅ | ⚠️ 基础 | ⚠️ P7 |
| 多租户/子账号 | ✅ | ❌ | 🔴 P8 |
| 支付接入（支付宝/微信） | ✅ | ❌ | 🔴 P4 |
| 充值系统 | ✅ | ❌ | 🔴 P4 |
| 发票管理 | ✅ | ❌ | 🔴 P4 |
| 监控告警 | ✅ | ✅ | ✅ P3 |
| 用量统计 | ✅ | ✅ | ✅ P3 |
| 批量推理 | ✅ | ✅ | ✅ P2 |

**我们的差异化优势**：

1. **私有部署** - 数据不出企业，适合敏感行业
2. **成本可控** - 无平台抽成，自建渠道成本更低
3. **定制灵活** - 开源可控，可按需定制
4. **自有渠道** - 已有三平台 vLLM 资源

---

## 2. One API 现有能力评估

### 2.1 原生支持（开箱即用）

| 功能 | 文件位置 | 状态 |
|------|----------|------|
| POST /v1/chat/completions | `router/relay.go:25` | ✅ 原生支持 |
| GET /v1/models | `router/relay.go:14-18` | ✅ 原生支持 |
| POST /v1/embeddings | `router/relay.go:30` | ✅ 原生支持 |
| 虚拟 Key 管理 | `model/token.go` | ✅ 原生支持 |
| 多渠道路由（优先级） | `model/cache.go:227-255` | ⚠️ 仅优先级，无权重 |
| Token 用量统计 | `model/log.go` | ✅ 原生支持 |
| 三方 API 接入 | `relay/adaptor/*.go` | ✅ 原生支持（50+渠道） |
| 失败自动重试 | `controller/relay.go:70-91` | ✅ 原生支持 |
| 渠道自动禁用（基于错误） | `monitor/manage.go:11-44` | ⚠️ 部分支持 |

### 2.2 需要二次开发的部分

| 功能 | 说明 | 复杂度 | 工期估算 |
|------|------|--------|----------|
| **并发Bug修复** | `controller/relay.go:97` 竞态条件 | 低 | 0.5天 |
| **HTTP连接池优化** | 提升并发吞吐 | 低 | 0.5天 |
| POST /v1/messages 透传 | Anthropic 格式透传路由 | 中 | 3~4天 |
| **熔断器实现** | 快速故障隔离 | 中 | 2天 |
| 主动健康检查 | 定时探测 + 恢复后自动加回 | 高 | 5~7天 |
| **加权负载均衡** | 动态权重 + 成功率感知 | 中 | 2天 |
| RPM/TPM 精细限流 | 按 Key/模型维度的限流 | 高 | 5~7天 |
| **请求队列（背压控制）** | 防止系统过载 | 中 | 1天 |
| POST /v1/files | 批量推理文件上传 | 中 | - |
| POST /v1/batches | 创建批量推理任务 | 高 | P2阶段 |
| GET /v1/batches/:id | 查询批量任务状态 | 低 | P2阶段 |
| GET /v1/files/:id/content | 下载批量推理结果 | 低 | P2阶段 |

### 2.3 已知代码问题（需修复）

#### 🔴 P0 - 并发安全问题

```go
// controller/relay.go:97
// BUG: bizErr is in race condition
bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, requestId)
```

**影响**：高并发场景下可能导致 panic 或数据竞争
**修复方案**：使用值拷贝而非指针共享

#### 🟡 P1 - 路由调度缺陷

```go
// model/cache.go:248
idx := rand.Intn(endIdx)  // 简单随机，未使用 Weight 字段
```

**影响**：无法按权重分配流量，无法感知渠道健康状态
**修复方案**：实现加权调度 + 动态权重计算

---

## 3. 分期开发计划

### 3.1 P0 第一期（约 2 周）

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 并发Bug修复 | 0.5天 | P0 | 后端 | 无 |
| HTTP连接池优化 | 0.5天 | P0 | 后端 | 无 |
| One API 渠道配置：三平台接入 | 2天 | P0 | 后端 | 无 |
| 新增 /v1/messages Anthropic 透传路由 | 3~4天 | P0 | 后端 | 无 |
| 熔断器实现 | 2天 | P0 | 后端 | 无 |
| 后端健康检查 + 故障自动摘除/恢复 | 5~7天 | P0 | 后端 | 熔断器 |
| 基础冒烟测试 + 接入文档初稿 | 2天 | P0 | 后端/文档 | 以上全部 |

**交付物**：可对外上线的在线推理服务

### 3.2 P1 第二期（约 2~3 周）

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 加权负载均衡（动态权重） | 2天 | P1 | 后端 | 健康检查 |
| 请求队列（背压控制） | 1天 | P1 | 后端 | 无 |
| RPM/TPM 精细限流（按 Key/模型） | 5~7天 | P1 | 后端 | 无 |
| 用量看板页面（Key/模型/时间维度） | 4~5天 | P1 | 全栈 | 无 |
| 三方 API 渠道验证（OpenAI/Anthropic） | 2天 | P1 | 后端 | 无 |

**交付物**：具备商业化运营基础的限流与统计能力

### 3.3 P2 第三期（约 3 周）

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| Batch Service：文件上传 + 任务管理 | 5~7天 | P2 | 后端 | 无 |
| Batch Worker：并发消费 + 结果写回 | 5~7天 | P2 | 后端 | Batch Service |
| Batch 与 One API Key 鉴权联通 | 2天 | P2 | 后端 | 以上全部 |

**交付物**：Batch API 上线，覆盖数据标注/评测场景

### 3.4 P3 第四期（已完成）

| 任务 | 工期 | 优先级 | 负责角色 |
|------|------|--------|----------|
| 最小并发数调度（长请求优化） | 2天 | P3 | 后端 |
| 监控指标暴露（Prometheus） | 3天 | P3 | 后端 |
| 用量统计API增强 | 3天 | P3 | 后端 |

**交付物**：监控告警体系，用量多维度统计

### 3.5 P4 第五期 - 商业化基础（约 2~3 周）

**目标**：对齐 n1n.ai 商业化核心功能

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 充值系统设计 | 3天 | P4 | 全栈 | 无 |
| 支付宝/微信支付接入 | 5天 | P4 | 全栈 | 充值系统 |
| 余额管理系统 | 3天 | P4 | 后端 | 充值系统 |
| 订单管理系统 | 3天 | P4 | 后端 | 支付接入 |
| 发票申请流程 | 2天 | P4 | 前端 | 订单管理 |

**交付物**：支持人民币充值、支付、发票的完整商业化流程

### 3.6 P5 第六期 - 用户门户重构（约 3~4 周）

**目标**：打造面向终端用户的全新交互界面

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 模型市场页面 | 5天 | P5 | 前端 | 无 |
| 用户Dashboard重构 | 4天 | P5 | 前端 | 无 |
| API Key管理界面 | 3天 | P5 | 前端 | 无 |
| 用量明细查询 | 3天 | P5 | 前端 | P3 |
| 充值中心页面 | 2天 | P5 | 前端 | P4 |

**交付物**：全新用户门户，支持模型浏览、API Key管理、充值

### 3.7 P7 第七期 - 运营管理后台（约 2~3 周）

**目标**：为运营团队提供高效管理工具

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 运营看板（营收/用户/用量） | 5天 | P7 | 全栈 | P3 |
| 用户管理增强 | 3天 | P7 | 后端 | 无 |
| 渠道健康监控面板 | 3天 | P7 | 前端 | P0 |
| 告警配置界面 | 2天 | P7 | 前端 | P3 |
| 财务报表导出 | 2天 | P7 | 后端 | P4 |

**交付物**：完整的运营管理后台，支持数据可视化

### 3.8 P8 第八期 - 多租户与权限（约 2~3 周）

**目标**：支持企业级多租户场景

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 子账号体系 | 4天 | P8 | 后端 | 无 |
| RBAC权限管理 | 3天 | P8 | 后端 | 子账号 |
| 部门/团队管理 | 3天 | P8 | 后端 | 子账号 |
| 额度分配策略 | 2天 | P8 | 后端 | 权限 |
| 审计日志 | 2天 | P8 | 后端 | 权限 |

**交付物**：企业级多租户支持，细粒度权限控制

### 3.9 P9 第九期 - 模型生态扩展（持续）

**目标**：扩展支持的模型范围

| 任务 | 工期 | 优先级 | 负责角色 | 依赖 |
|------|------|--------|----------|------|
| 国内模型接入（文心/通义/智谱） | 5天 | P9 | 后端 | 无 |
| 模型定价配置系统 | 3天 | P9 | 后端 | 无 |
| 模型分组与标签 | 2天 | P9 | 后端 | 定价 |
| 模型试用功能 | 2天 | P9 | 全栈 | 无 |
| 模型SLA配置 | 1天 | P9 | 后端 | 无 |

**交付物**：支持更多模型，灵活定价策略

---

## 4. 关键技术实施细节

### 4.1 并发Bug修复

**问题位置**：`controller/relay.go:97`

**修复方案**：
```go
// 修复前（有竞态）
if bizErr != nil {
    bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, requestId)
    c.JSON(bizErr.StatusCode, gin.H{"error": bizErr.Error})
}

// 修复后（值拷贝）
if bizErr != nil {
    errorResp := gin.H{
        "error": model.Error{
            Message: helper.MessageWithRequestId(bizErr.Error.Message, requestId),
            Type:    bizErr.Error.Type,
            Param:   bizErr.Error.Param,
            Code:    bizErr.Error.Code,
        },
    }
    c.JSON(bizErr.StatusCode, errorResp)
}
```

### 4.2 HTTP 连接池优化

**当前问题**：`common/client/init.go` 无连接池配置

**优化方案**：
```go
func Init() {
    transport := &http.Transport{
        Proxy: nil,
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        MaxIdleConns:          1000,    // 最大空闲连接
        MaxIdleConnsPerHost:   100,     // 每个主机最大空闲连接
        MaxConnsPerHost:       200,     // 每个主机最大连接
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
        ResponseHeaderTimeout: 10 * time.Second,
        ForceAttemptHTTP2:     true,
    }

    if config.RelayProxy != "" {
        proxyURL, _ := url.Parse(config.RelayProxy)
        transport.Proxy = http.ProxyURL(proxyURL)
    }

    HTTPClient = &http.Client{
        Transport: transport,
        Timeout:   time.Duration(config.RelayTimeout) * time.Second,
    }
}
```

### 4.3 Anthropic 格式透传

vLLM 原生支持 `/v1/messages` 端点。

**实现方案**：

1. **新增路由** (`router/relay.go`)：
```go
relayV1Router.POST("/messages", controller.RelayAnthropic)
```

2. **新增透传模式** (`relay/adaptor/anthropic/`)：
```go
// 新增 PassthroughAdaptor，不做协议转换
type PassthroughAdaptor struct{}

func (a *PassthroughAdaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
    // 直接返回原始请求体，不做转换
    return c.Request.Body, nil
}
```

3. **配置项**：Channel 表新增 `passthrough` 字段，标识是否透传模式

### 4.4 熔断器（Circuit Breaker）

**目的**：快速隔离故障渠道，防止级联故障

**新增组件** (`monitor/circuit_breaker.go`)：

```go
type CircuitBreaker struct {
    mu               sync.RWMutex
    state            State          // Closed, Open, HalfOpen
    failureCount     int
    successCount     int
    failureThreshold int           // 连续失败阈值
    successThreshold int           // 半开状态成功阈值
    timeout          time.Duration // 熔断持续时间
    lastFailureTime  time.Time
}

type State int
const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

func (cb *CircuitBreaker) AllowRequest() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        // 检查是否过了熔断期
        return time.Since(cb.lastFailureTime) > cb.timeout
    case StateHalfOpen:
        return true  // 允许探测请求
    }
    return false
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failureCount = 0
    if cb.state == StateHalfOpen {
        cb.successCount++
        if cb.successCount >= cb.successThreshold {
            cb.state = StateClosed
            cb.successCount = 0
        }
    }
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failureCount++
    cb.lastFailureTime = time.Now()

    if cb.state == StateHalfOpen {
        cb.state = StateOpen
        cb.successCount = 0
    } else if cb.failureCount >= cb.failureThreshold {
        cb.state = StateOpen
    }
}

// 全局熔断器管理
var circuitBreakers = make(map[int]*CircuitBreaker)
var cbMu sync.RWMutex

func GetCircuitBreaker(channelId int) *CircuitBreaker {
    cbMu.Lock()
    defer cbMu.Unlock()

    if cb, ok := circuitBreakers[channelId]; ok {
        return cb
    }

    cb := &CircuitBreaker{
        state:            StateClosed,
        failureThreshold: config.CircuitBreakerThreshold,
        successThreshold: config.CircuitBreakerSuccessThreshold,
        timeout:          time.Duration(config.CircuitBreakerTimeout) * time.Second,
    }
    circuitBreakers[channelId] = cb
    return cb
}
```

**配置项**：
```go
// common/config/config.go 新增
var CircuitBreakerThreshold = env.Int("CIRCUIT_BREAKER_THRESHOLD", 5)
var CircuitBreakerSuccessThreshold = env.Int("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3)
var CircuitBreakerTimeout = env.Int("CIRCUIT_BREAKER_TIMEOUT", 30)  // 秒
```

### 4.5 健康检查 + 自动摘除/恢复

**当前状态**：仅有被动禁用（基于API错误），无主动探测

**新增组件** (`monitor/health.go`)：

```go
type HealthChecker struct {
    interval    time.Duration
    timeout     time.Duration
    checkPath   string
    stopChan    chan struct{}
}

func NewHealthChecker() *HealthChecker {
    return &HealthChecker{
        interval:  time.Duration(config.HealthCheckInterval) * time.Second,
        timeout:   5 * time.Second,
        checkPath: "/v1/models",
        stopChan:  make(chan struct{}),
    }
}

func (h *HealthChecker) Start() {
    ticker := time.NewTicker(h.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            h.checkAllChannels()
        case <-h.stopChan:
            return
        }
    }
}

func (h *HealthChecker) checkAllChannels() {
    channels := getEnabledChannels()

    var wg sync.WaitGroup
    for _, ch := range channels {
        wg.Add(1)
        go func(channel *model.Channel) {
            defer wg.Done()
            h.checkChannel(channel)
        }(ch)
    }
    wg.Wait()
}

func (h *HealthChecker) checkChannel(channel *model.Channel) {
    ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
    defer cancel()

    url := fmt.Sprintf("%s%s", channel.GetBaseURL(), h.checkPath)
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))

    start := time.Now()
    resp, err := client.HTTPClient.Do(req)
    latency := time.Since(start).Milliseconds()

    cb := GetCircuitBreaker(channel.Id)

    if err != nil || resp.StatusCode >= 500 {
        cb.RecordFailure()
        recordHealthCheckResult(channel.Id, false, 0)
        return
    }
    defer resp.Body.Close()

    cb.RecordSuccess()
    recordHealthCheckResult(channel.Id, true, latency)
}

func recordHealthCheckResult(channelId int, success bool, latency int64) {
    // 更新统计信息，供调度器使用
    // 存入 Redis 或内存
    key := fmt.Sprintf("channel:stats:%d", channelId)
    stats := ChannelStats{
        LastCheckTime: time.Now().Unix(),
        LastCheckOK:   success,
        AvgLatency:    latency,
    }
    // 序列化并存储...
}
```

**配置项**：
```go
// common/config/config.go 新增
var HealthCheckInterval = env.Int("HEALTH_CHECK_INTERVAL", 30)      // 秒
var HealthCheckFailThreshold = env.Int("HEALTH_CHECK_FAIL_THRESHOLD", 3)
```

### 4.6 加权负载均衡

**当前问题**：`model/cache.go:248` 使用简单随机，未使用权重

**新增组件** (`monitor/scheduler.go`)：

```go
type Scheduler struct {
    mu            sync.RWMutex
    channelStats  map[int]*Stats   // channelId -> 统计信息
}

type Stats struct {
    SuccessCount   int64
    FailCount      int64
    AvgLatency     int64  // ms
    LastUsedTime   int64
    Concurrency    int32  // 当前并发数
}

// 加权随机选择
func (s *Scheduler) SelectChannel(channels []*Channel) *Channel {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // 先过滤掉熔断的渠道
    availableChannels := make([]*Channel, 0)
    for _, ch := range channels {
        cb := GetCircuitBreaker(ch.Id)
        if cb.AllowRequest() {
            availableChannels = append(availableChannels, ch)
        }
    }

    if len(availableChannels) == 0 {
        return nil
    }

    totalWeight := 0
    weights := make([]int, len(availableChannels))

    for i, ch := range availableChannels {
        // 动态权重 = 基础权重 × 成功率因子 × 延迟因子
        stats := s.channelStats[ch.Id]
        baseWeight := ch.GetWeight()
        if baseWeight == 0 {
            baseWeight = 1  // 默认权重
        }

        successRate := 1.0
        latencyFactor := 1.0

        if stats != nil && stats.SuccessCount+stats.FailCount > 0 {
            successRate = float64(stats.SuccessCount) / float64(stats.SuccessCount+stats.FailCount)
            // 延迟因子：延迟越低权重越高
            if stats.AvgLatency > 0 {
                latencyFactor = 1000.0 / float64(stats.AvgLatency+100)
            }
        }

        weights[i] = int(float64(baseWeight) * successRate * latencyFactor)
        if weights[i] < 1 {
            weights[i] = 1
        }
        totalWeight += weights[i]
    }

    // 加权随机
    r := rand.Intn(totalWeight)
    for i, w := range weights {
        r -= w
        if r <= 0 {
            return availableChannels[i]
        }
    }
    return availableChannels[0]
}

// 更新统计
func (s *Scheduler) RecordResult(channelId int, success bool, latency int64) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.channelStats == nil {
        s.channelStats = make(map[int]*Stats)
    }

    stats, ok := s.channelStats[channelId]
    if !ok {
        stats = &Stats{}
        s.channelStats[channelId] = stats
    }

    if success {
        stats.SuccessCount++
        // 滑动平均延迟
        stats.AvgLatency = (stats.AvgLatency + latency) / 2
    } else {
        stats.FailCount++
    }
}
```

**修改路由选择逻辑**：
```go
// model/cache.go 修改
func CacheGetRandomSatisfiedChannel(group string, model string, ignoreFirstPriority bool) (*Channel, error) {
    // ... 获取 channels 列表

    // 使用调度器选择渠道
    scheduler := monitor.GetScheduler()
    selected := scheduler.SelectChannel(channels)
    if selected == nil {
        return nil, errors.New("no available channel")
    }
    return selected, nil
}
```

### 4.7 请求队列（背压控制）

**目的**：防止系统过载，保护服务稳定性

**新增组件** (`middleware/queue.go`)：

```go
type RequestQueue struct {
    semaphore chan struct{}
    timeout   time.Duration
    stats     QueueStats
}

type QueueStats struct {
    CurrentQueueSize int64
    TotalRequests    int64
    RejectedRequests int64
}

func NewRequestQueue(maxConcurrent int, timeout time.Duration) *RequestQueue {
    return &RequestQueue{
        semaphore: make(chan struct{}, maxConcurrent),
        timeout:   timeout,
    }
}

func (q *RequestQueue) Acquire(c *gin.Context) bool {
    select {
    case q.semaphore <- struct{}{}:
        atomic.AddInt64(&q.stats.CurrentQueueSize, 1)
        atomic.AddInt64(&q.stats.TotalRequests, 1)
        return true
    case <-time.After(q.timeout):
        atomic.AddInt64(&q.stats.RejectedRequests, 1)
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": gin.H{
                "message": "Service temporarily unavailable, please retry later",
                "type":    "server_busy",
                "code":    "queue_full",
            },
        })
        c.Abort()
        return false
    }
}

func (q *RequestQueue) Release() {
    <-q.semaphore
    atomic.AddInt64(&q.stats.CurrentQueueSize, -1)
}

func (q *RequestQueue) GetStats() QueueStats {
    return q.stats
}

// 全局请求队列
var requestQueue *RequestQueue

func InitRequestQueue() {
    maxConcurrent := config.MaxConcurrentRequests
    if maxConcurrent == 0 {
        maxConcurrent = 1000  // 默认值
    }
    requestQueue = NewRequestQueue(maxConcurrent, 5*time.Second)
}

// 中间件
func QueueMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !requestQueue.Acquire(c) {
            return
        }
        defer requestQueue.Release()
        c.Next()
    }
}
```

**配置项**：
```go
// common/config/config.go 新增
var MaxConcurrentRequests = env.Int("MAX_CONCURRENT_REQUESTS", 1000)
var RequestQueueTimeout = env.Int("REQUEST_QUEUE_TIMEOUT", 5)  // 秒
```

**路由集成**：
```go
// router/relay.go 修改
relayV1Router.Use(
    middleware.RelayPanicRecover(),
    middleware.QueueMiddleware(),     // 新增：队列控制
    middleware.TokenAuth(),
    middleware.TokenRateLimit(),
    middleware.Distribute(),
)
```

### 4.8 RPM/TPM 精细限流

**当前状态**：仅支持 IP 级别限流

**实现方案**：

#### 1. Token 模型扩展

```go
// model/token.go 新增字段
type Token struct {
    // ...existing fields
    RateLimitRpm         int `json:"rate_limit_rpm" gorm:"default:0"`          // 0表示不限
    RateLimitTpm         int `json:"rate_limit_tpm" gorm:"default:0"`          // 0表示不限
    RateLimitConcurrent  int `json:"rate_limit_concurrent" gorm:"default:0"`   // 0表示不限
}
```

#### 2. 限流中间件

```go
// middleware/token-rate-limit.go
func TokenRateLimit() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenId := c.GetInt(ctxkey.TokenId)
        model := c.GetString(ctxkey.RequestModel)

        // RPM 检查
        if !checkRpm(tokenId) {
            abortWithMessage(c, 429, "RPM limit exceeded")
            return
        }

        // TPM 预检查（基于请求体估算）
        if !checkTpm(tokenId, model, estimateTokens(c)) {
            abortWithMessage(c, 429, "TPM limit exceeded")
            return
        }

        // 并发检查
        if !checkConcurrent(tokenId) {
            abortWithMessage(c, 429, "Concurrent request limit exceeded")
            return
        }

        c.Next()
    }
}

// Redis 滑动窗口实现
func checkRpm(tokenId int) bool {
    key := fmt.Sprintf("ratelimit:rpm:%d:%d", tokenId, time.Now().Unix()/60)
    count, _ := common.RDB.Incr(context.Background(), key).Result()
    common.RDB.Expire(context.Background(), key, time.Minute)

    token, _ := model.GetTokenById(tokenId)
    return token.RateLimitRpm == 0 || count <= int64(token.RateLimitRpm)
}
```

#### 3. 路由集成

```go
// router/relay.go 修改
relayV1Router.Use(
    middleware.RelayPanicRecover(),
    middleware.TokenAuth(),
    middleware.TokenRateLimit(),  // 新增
    middleware.Distribute(),
)
```

### 4.9 三平台渠道配置

**vLLM 渠道类型选择**：

推荐使用 `OpenAI` 类型（channeltype=1），配置自定义 BaseURL：

| 渠道名称 | H800-GLM4 | 910B3-GLM4 | 910B4-GLM4 |
|----------|-----------|------------|------------|
| Type | 1 (OpenAI) | 1 (OpenAI) | 1 (OpenAI) |
| BaseURL | http://mass-h800/v1 | http://mass-910b3/v1 | http://mass-910b4/v1 |
| Key | vllm-api-key | vllm-api-key | vllm-api-key |
| Models | glm-4,... | glm-4,... | glm-4,... |
| Priority | 10 | 10 | 10 |
| Weight | 按算力配比 | 按算力配比 | 按算力配比 |
| Group | default | default | default |

**负载均衡逻辑**：One API 自动在相同优先级的渠道间轮询，某渠道不可用时自动摘除。

### 4.11 充值与计费系统

**核心功能**：

```
┌─────────────────────────────────────────────────────┐
│                    用户侧                            │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐ │
│  │余额展示 │  │充值入口 │  │充值记录 │  │发票申请│ │
│  └─────────┘  └─────────┘  └─────────┘  └────────┘ │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                   支付网关层                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐              │
│  │ 支付宝  │  │ 微信支付 │  │ 银行卡  │              │
│  └─────────┘  └─────────┘  └─────────┘              │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                   业务逻辑层                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐ │
│  │充值订单 │  │余额管理 │  │计费引擎 │  │发票管理│ │
│  └─────────┘  └─────────┘  └─────────┘  └────────┘ │
└─────────────────────────────────────────────────────┘
```

**数据模型**：

```go
// 充值订单
type TopupOrder struct {
    Id            string `json:"id"`           // 订单号
    UserId        int    `json:"user_id"`
    Amount        float64 `json:"amount"`      // 充值金额(元)
    Quota         int64   `json:"quota"`        // 获得额度
    Status        string  `json:"status"`       // pending/paid/cancelled/refunded
    PayMethod     string  `json:"pay_method"`   // alipay/wechat/card
    PayOrderId    string  `json:"pay_order_id"`// 第三方订单号
    CreatedAt     int64   `json:"created_at"`
    PaidAt        *int64  `json:"paid_at"`
}

// 发票申请
type Invoice struct {
    Id            string `json:"id"`
    UserId        int    `json:"user_id"`
    OrderIds      string `json:"order_ids"`    // 关联订单，逗号分隔
    Amount        float64 `json:"amount"`       // 发票金额
    Status        string  `json:"status"`       // pending/approved/issued/rejected
    Title         string  `json:"title"`        // 发票抬头
    TaxNo         string  `json:"tax_no"`       // 税号
    Address       string  `json:"address"`       // 开票地址
    Bank          string  `json:"bank"`          // 开户行
    Account       string  `json:"account"`       // 银行账号
    CreatedAt     int64   `json:"created_at"`
    IssuedAt      *int64  `json:"issued_at"`
}
```

**计费引擎**：

```go
// 按量计费
func CalculateQuota(modelName string, promptTokens, completionTokens int) int64 {
    price := getModelPrice(modelName) // 分/千token
    quota := (int64(promptTokens) + int64(completionTokens)) * price / 1000
    return quota
}

// 包月/包年计费（未来扩展）
func CalculateSubscriptionPlan(planType string, users int) int64 {
    // ...
}
```

### 4.12 模型市场与展示

**功能要点**：

```go
// 模型元数据
type ModelInfo struct {
    Id           string   `json:"id"`            // 模型标识
    Name         string   `json:"name"`          // 显示名称
    Provider     string   `json:"provider"`       // 提供商
    Type         string   `json:"type"`           // chat/embedding/image
    Description  string   `json:"description"`
    ContextLen   int      `json:"context_len"`    // 上下文长度
    InputPrice   float64  `json:"input_price"`   // 输入价格(元/千token)
    OutputPrice  float64  `json:"output_price"`   // 输出价格(元/千token)
    Capabilities []string `json:"capabilities"`   // 支持的功能
    Status       string   `json:"status"`          // available/maintenance
    SortOrder    int      `json:"sort_order"`
}
```

**页面结构**：

```
模型市场首页
├── 搜索栏（按名称/提供商/类型筛选）
├── 模型分类Tab（全部/Chat/Embedding/图像/视频）
├── 模型卡片列表
│   ├── 模型名称 + 提供商Logo
│   ├── 定价信息
│   ├── 能力标签
│   └── "立即使用"按钮
└── 模型详情页
    ├── 基本信息
    ├── 能力说明
    ├── 价格计算器
    ├── API定价
    └── 使用文档
```

### 4.13 多租户与权限体系

**权限模型（RBAC）**：

```go
// 角色定义
const (
    RoleOwner  = 0  // 所有者（充值、删除）
    RoleAdmin  = 1  // 管理员（管理成员、额度分配）
    RoleMember = 2  // 成员（使用API）
    RoleViewer = 3  // 观察者（只读）
)

// 权限定义
const (
    PermissionTopup      = "topup"       // 充值
    PermissionManageUser = "manage_user" // 管理用户
    PermissionAllocQuota = "alloc_quota"  // 分配额度
    PermissionViewUsage  = "view_usage"   // 查看用量
    PermissionManageAPI  = "manage_api"   // 管理API Key
    PermissionViewBilling = "view_billing"// 查看账单
)

// 用户角色绑定
type UserRole struct {
    UserId     int      `json:"user_id"`
    RoleId     int      `json:"role_id"`
    TenantId   int      `json:"tenant_id"`
    Permissions []string `json:"permissions"` // 额外权限
}
```

### 4.14 运营看板

**核心指标**：

```
┌─────────────────────────────────────────────────────┐
│                    运营概览                          │
├─────────────────────────────────────────────────────┤
│  今日营收  │  今日用量  │  活跃用户  │  渠道健康   │
│  ¥12,345  │  1.2M tok │    156    │  98.5%     │
└─────────────────────────────────────────────────────┘

├── 营收趋势（折线图）
├── 用量排行（柱状图）
├── 用户增长趋势
├── 渠道负载分布
└── 告警事件列表
```

**关键实现**：

```go
// 运营统计数据
type OpsStats struct {
    TodayRevenue     float64            `json:"today_revenue"`
    TodayUsage       int64              `json:"today_usage_tokens"`
    ActiveUsers      int                `json:"active_users"`
    ChannelHealth    float64            `json:"channel_health_rate"`
    RevenueByDay     map[string]float64 `json:"revenue_by_day"`
    UsageByModel     map[string]int64   `json:"usage_by_model"`
    TopUpAmountByDay map[string]float64 `json:"topup_by_day"`
}
```

```
Batch Service 组件架构：
┌─────────────────────────────────────────────────────┐
│                   Batch Service                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│  │ /v1/files   │  │ /v1/batches │  │   Storage   │  │
│  │   (MinIO)   │  │  (MySQL)    │  │   (MinIO)   │  │
│  └─────────────┘  └─────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                   Redis Queue                        │
│              (Celery / asyncio Worker)              │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                   One API                            │
│            (在线推理 + 鉴权验证)                      │
└─────────────────────────────────────────────────────┘
```

**技术栈**：
- API 层：Python FastAPI
- 存储层：MinIO（文件）+ MySQL（任务元数据）
- 队列层：Redis + Celery / asyncio Worker
- 鉴权：复用 One API Token 体系

---

## 5. 部署架构

### 5.1 推荐部署方案（Docker Compose）

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - one-api

  one-api:
    build: ./one-api
    environment:
      - SQL_DSN=root:password@tcp(mysql:3306)/oneapi
      - REDIS_CONN_STRING=redis://redis:6379
      - SESSION_SECRET=${SESSION_SECRET}
      - SYNC_FREQUENCY=60
      - HEALTH_CHECK_INTERVAL=30
      - CIRCUIT_BREAKER_THRESHOLD=5
      - MAX_CONCURRENT_REQUESTS=1000
    depends_on:
      - mysql
      - redis
    deploy:
      replicas: 2  # 多实例

  # 支付服务（P4新增）
  payment-service:
    build: ./payment-service
    environment:
      - DATABASE_URL=mysql://root:password@mysql:3306/oneapi
      - ALIPAY_APP_ID=${ALIPAY_APP_ID}
      - ALIPAY_PRIVATE_KEY=${ALIPAY_PRIVATE_KEY}
      - ALIPAY_PUBLIC_KEY=${ALIPAY_PUBLIC_KEY}
      - WECHAT_APP_ID=${WECHAT_APP_ID}
      - WECHAT_MCH_ID=${WECHAT_MCH_ID}
      - WECHAT_API_KEY=${WECHAT_API_KEY}
    depends_on:
      - mysql
      - redis

  batch-service:
    build: ./batch-service
    environment:
      - DATABASE_URL=mysql://root:password@mysql:3306/oneapi
      - REDIS_URL=redis://redis:6379
      - MINIO_ENDPOINT=minio:9000
    depends_on:
      - mysql
      - redis
      - minio

  batch-worker:
    build: ./batch-service
    command: celery -A worker worker -l info
    environment:
      - DATABASE_URL=mysql://root:password@mysql:3306/oneapi
      - REDIS_URL=redis://redis:6379
      - MINIO_ENDPOINT=minio:9000
    depends_on:
      - batch-service

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=oneapi
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=minioadmin
    volumes:
      - minio_data:/data

volumes:
  mysql_data:
  redis_data:
  minio_data:
```

### 5.2 高可用配置

| 组件 | 方案 |
|------|------|
| One API | 无状态，Nginx upstream 多实例负载 |
| MySQL | 主从复制，读写分离 |
| Redis | Sentinel 或 Cluster 模式 |
| MinIO | 分布式模式（多节点） |

### 5.3 监控方案

| 指标 | 工具 | 告警阈值 |
|------|------|----------|
| 接口延迟 | Prometheus + Grafana | P99 > 5s |
| 错误率 | Prometheus + Grafana | > 1% |
| Token消耗 | Prometheus + Grafana | 趋势监控 |
| 渠道状态 | Prometheus + Grafana | 节点摘除事件 |
| 请求队列深度 | Prometheus + Grafana | > 80% 容量 |
| 熔断器状态 | Prometheus + Grafana | 任意渠道熔断 |
| 日志 | Loki + Grafana | 按 Key/模型检索 |

---

## 6. 客户接入说明

### 6.1 OpenAI SDK 接入（推荐）

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxxx",              # 由服务方分发
    base_url="https://api.yourdomain.com/v1"
)

response = client.chat.completions.create(
    model="glm-4",
    messages=[{"role": "user", "content": "你好"}]
)
```

### 6.2 Anthropic SDK 接入

```python
import anthropic

client = anthropic.Anthropic(
    api_key="sk-xxxxxx",
    base_url="https://api.yourdomain.com"
)

message = client.messages.create(
    model="claude-3-opus-20240229",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}]
)
```

### 6.3 批量推理接入

```bash
# Step 1: 上传输入文件
curl -X POST https://api.yourdomain.com/v1/files \
  -H "Authorization: Bearer sk-xxxxxx" \
  -F "file=@input.jsonl" \
  -F "purpose=batch"

# Step 2: 创建批量任务
curl -X POST https://api.yourdomain.com/v1/batches \
  -H "Authorization: Bearer sk-xxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "input_file_id": "file-xxx",
    "endpoint": "/v1/chat/completions",
    "completion_window": "24h"
  }'

# Step 3: 查询任务状态
curl https://api.yourdomain.com/v1/batches/batch-xxx \
  -H "Authorization: Bearer sk-xxxxxx"

# Step 4: 下载结果文件
curl https://api.yourdomain.com/v1/files/file-yyy/content \
  -H "Authorization: Bearer sk-xxxxxx" \
  -o output.jsonl
```

---

## 7. 风险与约束

| 风险项 | 影响 | 缓解措施 |
|--------|------|----------|
| One API 版本升级冲突 | 二次开发代码与上游合并困难 | Fork 后锁定版本，建独立分支，上游更新人工评估后合并 |
| 并发Bug | 高并发场景下可能 panic | P0 阶段优先修复 |
| 单点故障（One API 挂了） | 所有客户服务中断 | Docker 多实例 + Nginx upstream |
| 上游平台限流/欠费 | 部分渠道失效 | 熔断器 + 健康检查 + 自动摘除，告警通知 |
| Key 泄露 | 客户额度被盗用 | Key 绑定 IP 白名单，异常用量实时告警 |
| Batch 任务积压 | Worker 处理不过来 | 设置任务超时，监控队列深度，扩 Worker 实例 |
| Token 估算偏差 | 流式请求无法精确预知 Token 数 | 使用近似估算，后置校准 |
| vLLM 版本兼容 | 不同版本 API 可能有差异 | 测试验证，锁定 vLLM 版本 |
| 热点渠道 | 某渠道被过度调用 | 加权调度 + 动态权重 + 熔断保护 |

---

## 8. 附录

### 8.1 技术栈清单

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| API 网关 | One API (Go) | 二次开发核心，OpenAI 兼容 + Anthropic 透传 |
| Batch Service | Python / FastAPI | 批量推理任务管理 API |
| Payment Service | Go / Gin | 支付网关服务（支付宝/微信） |
| Task Queue | Celery + Redis | 异步批量任务调度与消费 |
| 数据库 | MySQL 8.0 | One API 主库 + 支付扩展表 |
| 缓存 / 限流 | Redis 7 | 滑动窗口限流计数器 + 会话缓存 |
| 文件存储 | MinIO | 批量推理输入/输出 JSONL 文件 |
| 反向代理 | Nginx | 入口流量 + SSL 终止 |
| 容器化 | Docker Compose | 小团队首选，后续可迁移 K8s |
| 监控 | Prometheus + Grafana | 接口指标 + 用量趋势 + 告警 |
| 日志 | Loki + Grafana | 结构化日志，按维度检索 |
| 后端引擎 | vLLM | 三平台统一推理引擎 |
| 前端框架 | React + Ant Design | 用户门户 + 运营后台 |
| 支付渠道 | 支付宝/微信支付 | 人民币充值 |

### 8.2 关键文件清单

| 文件 | 说明 | 修改频率 |
|------|------|----------|
| `controller/relay.go` | 核心中继逻辑 | 高 |
| `router/relay.go` | 路由定义 | 中 |
| `middleware/rate-limit.go` | 限流中间件 | 高 |
| `middleware/queue.go` | 请求队列 | 新建 |
| `monitor/health.go` | 健康检查 | 新建 |
| `monitor/circuit_breaker.go` | 熔断器 | 新建 |
| `monitor/scheduler.go` | 加权调度器 | 新建 |
| `monitor/metrics.go` | 监控指标 | 新建 |
| `model/batch.go` | Batch模型 | 新建 |
| `controller/batch.go` | Batch控制器 | 新建 |
| `controller/usage.go` | 用量统计API | 新建 |
| `controller/metrics.go` | 监控端点 | 新建 |
| `model/topup.go` | 充值订单模型 | P4新建 |
| `model/invoice.go` | 发票模型 | P4新建 |
| `controller/payment.go` | 支付控制器 | P4新建 |
| `model/token.go` | Token 模型 | 中 |
| `common/client/init.go` | HTTP客户端 | 中 |
| `relay/adaptor/anthropic/` | Anthropic 适配器 | 中 |

### 8.3 环境变量清单

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SQL_DSN` | - | MySQL 连接串 |
| `REDIS_CONN_STRING` | - | Redis 连接串 |
| `SESSION_SECRET` | UUID | 会话密钥 |
| `SYNC_FREQUENCY` | 600 | 缓存同步频率（秒） |
| `HEALTH_CHECK_INTERVAL` | 30 | 健康检查间隔（秒） |
| `HEALTH_CHECK_FAIL_THRESHOLD` | 3 | 健康检查失败阈值 |
| `CIRCUIT_BREAKER_THRESHOLD` | 5 | 熔断器失败阈值 |
| `CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | 3 | 熔断器恢复成功阈值 |
| `CIRCUIT_BREAKER_TIMEOUT` | 30 | 熔断持续时间（秒） |
| `MAX_CONCURRENT_REQUESTS` | 1000 | 最大并发请求数 |
| `REQUEST_QUEUE_TIMEOUT` | 5 | 请求队列超时（秒） |
| `MEMORY_CACHE_ENABLED` | false | 内存缓存开关 |
| `BATCH_UPDATE_ENABLED` | false | 批量更新开关 |
| `METRICS_ENABLED` | true | 启用指标暴露 |
| `METRICS_PATH` | /metrics | 指标端点路径 |
| `ENABLE_LEAST_CONNECTION_LB` | false | 最小连接数负载均衡开关 |

**支付相关环境变量（P4）**：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ALIPAY_APP_ID` | - | 支付宝应用ID |
| `ALIPAY_PRIVATE_KEY` | - | 支付宝私钥 |
| `ALIPAY_PUBLIC_KEY` | - | 支付宝公钥 |
| `WECHAT_APP_ID` | - | 微信支付应用ID |
| `WECHAT_MCH_ID` | - | 微信支付商户号 |
| `WECHAT_API_KEY` | - | 微信支付API密钥 |

### 8.4 优化优先级速查表

| 优先级 | 优化项 | 工期 | 状态 |
|--------|--------|------|------|
| P0 | 并发Bug修复 | 0.5天 | ✅ 已完成 |
| P0 | HTTP连接池优化 | 0.5天 | ✅ 已完成 |
| P0 | 熔断器实现 | 2天 | ✅ 已完成 |
| P1 | 主动健康检查 | 3天 | ✅ 已完成 |
| P1 | 加权调度 | 2天 | ✅ 已完成 |
| P1 | 请求队列 | 1天 | ✅ 已完成 |
| P2 | Batch批量推理 | 5~7天 | ✅ 已完成 |
| P3 | 监控指标暴露 | 3天 | ✅ 已完成 |
| P4 | 充值支付系统 | 10~15天 | 🔴 待开发 |
| P5 | 用户门户重构 | 10~15天 | 🔴 待开发 |
| P7 | 运营管理后台 | 8~10天 | 🔴 待开发 |
| P8 | 多租户权限 | 8~10天 | 🔴 待开发 |
| P9 | 模型生态扩展 | 持续 | 🔴 待开发 |

---

## 9. 更新记录

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2026-03 | 初版 |
| v1.1 | 2026-03 | 根据代码review调整：新增并发Bug修复、调整工期估算、补充实现细节 |
| v1.2 | 2026-03 | 新增路由调度与高并发优化：熔断器、加权负载均衡、HTTP连接池、请求队列等 |
| v1.3 | 2026-03 | 对标n1n.ai：新增P4-P9商业化功能规划（充值支付、模型市场、运营后台、多租户等）|