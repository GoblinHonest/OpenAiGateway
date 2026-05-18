# 架构设计概览

## 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Layer                             │
│  (External API Consumers with API Keys: sk-xxx...)              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Gateway Entry                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Auth Filter  │→ │ Rate Limiter │→ │ Request Log  │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Protocol Layer                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   OpenAI     │  │  Anthropic   │  │   Gemini     │         │
│  │  Converter   │  │  Converter   │  │  Converter   │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Routing Layer                                │
│  ┌────────────────────────────────────────────────────────┐    │
│  │              Load Balance Strategies                    │    │
│  │  • Round Robin  • Weighted  • Least Connections        │    │
│  │  • Priority     • Adaptive (based on health)           │    │
│  └────────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────────┐    │
│  │              Token Selection Engine                     │    │
│  │  • Quota Check  • Rate Limit Check  • Health Check     │    │
│  │  • Optimistic Lock  • Distributed Lock (Redis)         │    │
│  └────────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Resilience Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Circuit Breaker│ │ Retry Logic  │  │   Fallback   │         │
│  │ (Persistent)  │  │ (Exponential)│  │   Handler    │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Execution Layer                                │
│  ┌────────────────────────────────────────────────────────┐    │
│  │              HTTP Client Pool                           │    │
│  │  • Connection Pooling  • Timeout Control               │    │
│  │  • Keep-Alive  • TLS Configuration                     │    │
│  └────────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────────┐    │
│  │            Stream Handler                               │    │
│  │  • Context Done Detection                              │    │
│  │  • Backpressure Control                                │    │
│  │  • Keepalive Monitoring                                │    │
│  └────────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Provider Services                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   OpenAI     │  │  Anthropic   │  │   Gemini     │         │
│  │   (Token1)   │  │   (Token2)   │  │   (Token3)   │         │
│  │   (Token2)   │  │   (Token3)   │  │   (Token4)   │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    Cross-Cutting Concerns                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Logging    │  │   Metrics    │  │   Tracing    │         │
│  │  (Structured)│  │ (Prometheus) │  │ (OpenTelemetry)│       │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │Health Checker│  │ Reconciliation│ │Event Bus     │         │
│  │  (Periodic)  │  │   (Daily)     │  │ (进程内)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Storage Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   SQLite     │  │    Redis     │  │   MySQL/PG   │         │
│  │ (开发/测试)   │  │  (必须依赖)   │  │  (生产环境)   │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

## 核心设计原则

### 1. 分层架构
- **表示层**: HTTP路由、中间件、请求验证
- **应用层**: 业务逻辑编排、协议转换
- **领域层**: 核心业务模型、策略模式
- **基础设施层**: 数据库、缓存、外部API调用

### 2. 数据库策略

| 环境 | 数据库 | 原因 |
|------|--------|------|
| 开发/测试 | SQLite | 零配置、轻量、便于本地开发 |
| 生产环境 | MySQL/PostgreSQL | 支持多实例写入、高并发、数据安全 |

**重要约束**: SQLite不支持多实例并发写入。生产环境如果需要水平扩展，必须使用MySQL/PostgreSQL。

### 3. Redis依赖说明

Redis是**必须依赖**，用于：
- 分布式锁（Token选择并发控制）
- 熔断器状态同步（Pub/Sub）
- 速率限制计数器
- 热点数据缓存

**Redis不可用时的降级策略**:
```
Redis不可用
    ↓
熔断器状态使用本地内存（单实例模式）
    ↓
分布式锁降级为本地互斥锁（单实例模式）
    ↓
速率限制降级为本地计数器（单实例模式）
    ↓
如果部署了多实例，需切换为单实例模式运行
```

### 4. 关键设计模式

#### Strategy Pattern (策略模式)
```go
type LoadBalanceStrategy interface {
    SelectProvider(ctx context.Context, providers []*Provider) (*Provider, error)
}

type RoundRobinStrategy struct{}
type WeightedStrategy struct{}
type LeastConnectionsStrategy struct{}
type PriorityStrategy struct{}
type AdaptiveStrategy struct{}
```

#### Adapter Pattern (适配器模式)
```go
type ProtocolConverter interface {
    ConvertRequest(ctx context.Context, req *http.Request) (*ProviderRequest, error)
    ConvertResponse(ctx context.Context, resp *ProviderResponse) (*http.Response, error)
    ConvertStreamResponse(ctx context.Context, stream <-chan *StreamChunk, writer *SSEHandler) error
}

type OpenAIConverter struct{}
type AnthropicConverter struct{}
type GeminiConverter struct{}
```

#### Circuit Breaker Pattern (熔断器模式)
```go
type CircuitBreaker interface {
    Call(ctx context.Context, fn func() error) error
    State() CircuitState
}

// States: Closed → Open → Half-Open → Closed
```

### 5. 并发安全保证

#### Token选择的并发控制
```go
// 方案1: 数据库乐观锁（推荐）
UPDATE tokens
SET quota_remaining = quota_remaining - ?,
    version = version + 1
WHERE id = ? AND version = ? AND quota_remaining >= ?

// 方案2: 分布式锁 (Redis)
lock := redis.SetNX("lock:token:"+tokenID, uuid, 5*time.Second)
defer redis.Del("lock:token:"+tokenID)
```

#### 熔断器状态持久化
```go
// 三层存储
1. Database (持久化，重启后恢复)
2. Redis (分布式共享，多实例同步)
3. Local Memory (快速访问，减少Redis压力)

// Pub/Sub同步
redis.Publish("circuit_breaker:state_change", stateChangeEvent)
```

### 6. 容错与降级

#### 重试策略
```go
type RetryConfig struct {
    MaxAttempts      int
    InitialBackoff   time.Duration
    MaxBackoff       time.Duration
    BackoffMultiplier float64
    RetryableErrors  []int  // 可重试的HTTP状态码
}

// 指数退避: delay = min(initialBackoff * multiplier^attempt, maxBackoff)
```

#### 降级链路
```
Primary Provider (Token1)
  → Retry with Token2 (同一Provider的其他Token)
    → Fallback to Provider2 (其他Provider)
      → Circuit Breaker Open (熔断器打开)
        → Return Cached Response (返回缓存响应，如果有)
          → Return Error with Retry-After (返回错误，提示重试时间)
```

### 7. 流式响应处理

#### 客户端断开检测（Go 1.20+）
```go
// 方法1: Context Done（推荐，支持HTTP/1.1和HTTP/2）
select {
case <-ctx.Done():
    // 客户端断开或请求取消
    return ctx.Err()
case data <- streamChan:
    // 正常数据
}

// 方法2: http.NewResponseController（Go 1.20+）
rc := http.NewResponseController(w)
rc.SetReadDeadline(time.Now().Add(30 * time.Second))
rc.SetWriteDeadline(time.Now().Add(30 * time.Second))

// 方法3: Keepalive Monitoring（检测客户端是否存活）
ticker := time.NewTicker(15 * time.Second)
defer ticker.Stop()
select {
case <-ticker.C:
    w.Write([]byte(": keepalive\n\n"))
    flusher.Flush()
case <-ctx.Done():
    return ctx.Err()
}
```

**注意**: `http.CloseNotifier` 自Go 1.11起已废弃，不应使用。

#### 背压控制
```go
type StreamHandler struct {
    bufferSize    int
    maxChunkSize  int
    flushInterval time.Duration
}

// 限制缓冲区大小，防止内存溢出
buffer := make([]byte, h.bufferSize)

// 带缓冲的channel控制
streamChan := make(chan []byte, 100)  // 最多缓冲100个chunk
```

### 8. 可观测性

#### 结构化日志
```go
logger.Info("request_completed",
    zap.String("request_id", reqID),
    zap.String("provider", provider),
    zap.Int("status_code", statusCode),
    zap.Duration("latency", latency),
    zap.Int("input_tokens", inputTokens),
    zap.Int("output_tokens", outputTokens),
)

// 敏感数据脱敏
// - token_value 不记录
// - api_key 只记录前缀
// - request_body 中的敏感字段脱敏
```

#### Metrics (Prometheus)
```go
// Counter
requestsTotal.WithLabelValues(provider, model, status).Inc()

// Histogram
requestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())

// Gauge
activeConnections.WithLabelValues(provider).Set(float64(count))
```

#### Tracing (OpenTelemetry)
```go
ctx, span := tracer.Start(ctx, "gateway.request")
defer span.End()

span.SetAttributes(
    attribute.String("provider", provider),
    attribute.String("model", model),
)
```

### 9. 数据一致性

#### 对账系统
```go
type ReconciliationJob struct {
    Type     ReconciliationType // token_quota, usage_stats, cost
    Schedule string              // "0 2 * * *" (每天凌晨2点)
}

// 对账流程
1. 从数据库读取预期值 (quota_remaining)
2. 从日志聚合实际值 (SUM(input_tokens + output_tokens))
3. 比较差异
4. 记录不一致到 reconciliation_records 表
5. 自动修正或告警
```

### 10. 健康检查

#### Provider健康检查
```go
type HealthChecker struct {
    interval      time.Duration // 5分钟
    timeout       time.Duration // 10秒
    healthyThreshold   int      // 连续2次成功
    unhealthyThreshold int      // 连续3次失败
}

// 检查指标
- Latency (延迟) - 使用EWMA计算
- Error Rate (错误率) - 最近5分钟
- Availability (可用性) - 成功率

// 健康状态
type HealthStatus string
const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
)
```

### 11. 优雅关闭

```go
type GracefulShutdown struct {
    server         *http.Server
    activeRequests sync.WaitGroup
    shutdownTimeout time.Duration // 30秒
}

// 关闭流程
1. 设置 shuttingDown 标志，拒绝新请求
2. 等待现有请求完成 (最多30秒)
3. 关闭数据库连接
4. 关闭Redis连接
5. 刷新日志缓冲区
```

### 12. 安全性

#### API Key管理
```go
// 生成
plainKey := "sk-" + base64.URLEncoding.EncodeToString(randomBytes(32))
keyHash := sha256.Sum256([]byte(plainKey))

// 验证（常量时间比较，防止时序攻击）
inputHash := sha256.Sum256([]byte(inputKey))
if subtle.ConstantTimeCompare(storedHash, inputHash) == 1 {
    // 验证通过
}
```

#### Token存储（明文）
```sql
-- 明文存储 (按用户要求)
CREATE TABLE tokens (
    token_value TEXT NOT NULL  -- 明文存储，不加密
);
```

**安全补偿措施**:
1. 数据库访问严格控制（最小权限原则）
2. 管理API使用独立的Admin Token认证
3. 所有管理操作记录审计日志（admin_audit_logs表）
4. 日志中不记录token明文
5. 数据库传输加密（TLS）
6. 定期备份，加密存储

#### 敏感数据日志脱敏
```go
// 需要脱敏的字段
- token_value -> 不记录
- api_key -> 只记录前缀 "sk-xxxx"
- request_body -> 根据模型脱敏敏感字段
- admin_token -> 只记录前缀
```

### 13. Event Bus设计

Event Bus使用**进程内channel实现**，不依赖外部消息队列。

```go
type EventBus struct {
    subscribers map[string][]chan Event
    mu          sync.RWMutex
    bufferSize  int
}

type Event struct {
    Type      string      // circuit_breaker_change, provider_health_change, etc.
    Payload   interface{}
    Timestamp time.Time
}

// 事件类型
const (
    EventCircuitBreakerChange = "circuit_breaker_change"
    EventProviderHealthChange = "provider_health_change"
    EventTokenExhausted       = "token_exhausted"
    EventQuotaExceeded        = "quota_exceeded"
)

// 使用场景
- 熔断器状态变化 → 通知负载均衡器更新可用Provider列表
- Provider健康状态变化 → 通知路由层
- Token配额耗尽 → 通知Token选择器
```

### 14. 速率限制实现

```go
type RateLimiter struct {
    redis *redis.Client
}

// 滑动窗口算法
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
    now := time.Now().UnixMilli()
    windowStart := now - window.Milliseconds()

    // Redis Lua脚本保证原子性
    script := redis.NewScript(`
        local key = KEYS[1]
        local window_start = tonumber(ARGV[1])
        local now = tonumber(ARGV[2])
        local limit = tonumber(ARGV[3])

        -- 移除窗口外的记录
        redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

        -- 获取当前窗口内的请求数
        local count = redis.call('ZCARD', key)

        if count < limit then
            -- 添加当前请求
            redis.call('ZADD', key, now, now .. '-' .. math.random())
            redis.call('EXPIRE', key, math.ceil(tonumber(ARGV[4]) / 1000))
            return 1
        else
            return 0
        end
    `)

    result, err := script.Run(ctx, rl.redis, []string{key},
        windowStart, now, limit, window.Milliseconds()).Int()
    if err != nil {
        return false, err
    }

    return result == 1, nil
}
```

**限制维度**:
- API Key级别: `rate_limit:apikey:{key_id}`
- Group级别: `rate_limit:group:{group_id}`
- Provider级别: `rate_limit:provider:{provider_id}`

## 技术栈

### 核心框架
- **语言**: Go 1.21+
- **Web框架**: Gin (高性能HTTP路由)
- **数据库**: SQLite (开发/测试), MySQL/PostgreSQL (生产)
- **缓存/锁**: Redis (必须依赖)

### 依赖库
```go
// HTTP客户端
"net/http"
"golang.org/x/net/http2"

// 数据库
"github.com/mattn/go-sqlite3"  // SQLite驱动
"gorm.io/gorm"                  // ORM

// Redis
"github.com/redis/go-redis/v9"

// 日志
"go.uber.org/zap"

// Metrics
"github.com/prometheus/client_golang/prometheus"

// Tracing
"go.opentelemetry.io/otel"

// 配置
"github.com/spf13/viper"

// 迁移
"github.com/golang-migrate/migrate/v4"

// 测试
"github.com/stretchr/testify"
```

## 性能目标

**前提假设**: 不含Provider响应时间，仅计算网关内部处理延迟。

| 指标 | 目标 | 测试条件 |
|------|------|----------|
| 吞吐量 | 10,000+ req/s | 单实例，16核CPU，32GB内存 |
| 延迟 P50 | < 10ms | 数据库查询+Redis操作+协议转换 |
| 延迟 P99 | < 50ms | 同上 |
| 可用性 | 99.9% | 多实例部署，Redis集群 |
| 并发连接 | 10,000+ | WebSocket/SSE长连接 |
| 内存占用 | < 500MB (空闲) | 单实例 |
| 内存占用 | < 2GB (高负载) | 10,000并发连接 |

**基准测试计划**:
1. 使用 `wrk` 或 `k6` 进行压测
2. 测试场景: 非流式请求、流式请求、并发连接
3. 逐步增加负载，记录P50/P95/P99延迟
4. 监控内存、CPU、Goroutine数量

## 扩展性

### 水平扩展
- 无状态设计，支持多实例部署
- Redis共享状态（分布式锁、熔断器状态、速率限制）
- **注意**: 使用SQLite时无法水平扩展，必须切换到MySQL/PostgreSQL

### 垂直扩展
- 连接池大小可配置
- Worker数量可配置
- 缓冲区大小可配置

### 功能扩展
- 插件化协议转换器
- 可插拔负载均衡策略
- 自定义中间件支持
- 模型映射可配置（数据库驱动，非硬编码）
