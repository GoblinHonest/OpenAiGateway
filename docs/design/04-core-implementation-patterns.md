# 核心实现模式

## 1. Token选择引擎

### 1.1 Token选择流程

```
请求进入
    ↓
获取当前Model可用的Token列表
    ↓
过滤不可用Token (quota=0, status=inactive, health=fail)
    ↓
检查每个Token的并发锁 (Redis分布式锁)
    ↓
使用负载均衡策略选择Token
    ↓
检查Token的可用配额 (乐观锁)
    ↓
    ┌─────────────────┐
    │ 配额是否足够?    │
    ├─────────────────┤
    │ Yes → 锁定Token │
    │       更新配额   │
    │       返回Token  │
    ├─────────────────┤
    │ No → 重试下一个  │
    │       或报错     │
    └─────────────────┘
```

### 1.2 乐观锁实现

```go
func (r *TokenRepository) DecrementQuota(ctx context.Context, tokenID string, amount int, expectedVersion int) (bool, error) {
    // 使用乐观锁更新配额
    // 1. 先读取当前version
    // 2. 更新时检查version是否变化
    // 3. 更新成功则version+1
    query := `
        UPDATE tokens
        SET quota_remaining = quota_remaining - ?,
            version = version + 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ? AND quota_remaining >= ? AND status = 'active' AND version = ?
    `

    result, err := r.db.ExecContext(ctx, query, amount, tokenID, amount, expectedVersion)
    if err != nil {
        return false, err
    }

    affected, err := result.RowsAffected()
    if err != nil {
        return false, err
    }

    return affected > 0, nil
}

// 使用示例
func (s *TokenService) UseToken(ctx context.Context, tokenID string, amount int) error {
    // 读取当前version
    token, err := s.repo.GetByID(ctx, tokenID)
    if err != nil {
        return err
    }

    // 尝试更新（乐观锁）
    success, err := s.repo.DecrementQuota(ctx, tokenID, amount, token.Version)
    if err != nil {
        return err
    }

    if !success {
        // 版本冲突或配额不足，重试
        return errors.New("optimistic lock conflict or insufficient quota")
    }

    return nil
}
```

### 1.3 分布式锁实现

```go
type DistributedLock struct {
    redis   *redis.Client
    key     string
    value   string // UUID，用于安全释放
    ttl     time.Duration
    cancel  context.CancelFunc // 用于取消自动续期
}

func NewDistributedLock(redis *redis.Client, key string, ttl time.Duration) *DistributedLock {
    return &DistributedLock{
        redis: redis,
        key:   key,
        value: uuid.New().String(),
        ttl:   ttl,
    }
}

func (l *DistributedLock) Acquire(ctx context.Context) (bool, error) {
    success, err := l.redis.SetNX(ctx, l.key, l.value, l.ttl).Result()
    if err != nil {
        return false, err
    }

    if success {
        // 启动自动续期
        l.startAutoRenew(ctx)
    }

    return success, nil
}

func (l *DistributedLock) startAutoRenew(ctx context.Context) {
    ctx, l.cancel = context.WithCancel(ctx)

    go func() {
        ticker := time.NewTicker(l.ttl / 3) // 每1/3 TTL续期一次
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // 续期：只有自己持有锁时才续期
                script := redis.NewScript(`
                    if redis.call("get", KEYS[1]) == ARGV[1] then
                        return redis.call("pexpire", KEYS[1], ARGV[2])
                    else
                        return 0
                    end
                `)
                script.Run(ctx, l.redis, []string{l.key}, l.value, l.ttl.Milliseconds())
            }
        }
    }()
}

func (l *DistributedLock) Release(ctx context.Context) error {
    // 取消自动续期
    if l.cancel != nil {
        l.cancel()
    }

    // 使用Lua脚本保证原子性
    script := redis.NewScript(`
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `)

    result, err := script.Run(ctx, l.redis, []string{l.key}, l.value).Result()
    if err != nil {
        return err
    }

    if result.(int64) == 0 {
        return errors.New("lock not held by this process")
    }

    return nil
}

// 使用示例
func (s *TokenService) SelectToken(ctx context.Context, modelID string) (*Token, error) {
    lock := NewDistributedLock(s.redis, "lock:model:"+modelID, 5*time.Second)
    acquired, err := lock.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    if !acquired {
        return nil, errors.New("failed to acquire lock")
    }
    defer lock.Release(ctx)

    // 选择Token逻辑
    return s.selectTokenInternal(ctx, modelID)
}
```

### 1.4 负载均衡策略

#### Round Robin (轮询)

```go
type RoundRobinStrategy struct {
    current atomic.Uint64
}

func (s *RoundRobinStrategy) SelectProvider(ctx context.Context, providers []*Provider) (*Provider, error) {
    if len(providers) == 0 {
        return nil, errors.New("no providers available")
    }

    idx := s.current.Add(1)
    return providers[idx%uint64(len(providers))], nil
}
```

#### Weighted Round Robin (加权轮询)

```go
type WeightedStrategy struct {
    weights map[string]int // provider_id -> weight
}

func (s *WeightedStrategy) SelectProvider(ctx context.Context, providers []*Provider) (*Provider, error) {
    // 使用加权轮询算法
    totalWeight := 0
    for _, p := range providers {
        totalWeight += s.weights[p.ID]
    }

    random := rand.Intn(totalWeight)
    cumulative := 0

    for _, p := range providers {
        cumulative += s.weights[p.ID]
        if random < cumulative {
            return p, nil
        }
    }

    return providers[len(providers)-1], nil
}
```

#### Least Connections (最少连接)

```go
type LeastConnectionsStrategy struct {
    connections map[string]*atomic.Int64 // provider_id -> active connections
}

func (s *LeastConnectionsStrategy) SelectProvider(ctx context.Context, providers []*Provider) (*Provider, error) {
    var selected *Provider
    minConns := int64(math.MaxInt64)

    for _, p := range providers {
        conns := s.connections[p.ID].Load()
        if conns < minConns {
            minConns = conns
            selected = p
        }
    }

    if selected == nil {
        return nil, errors.New("no providers available")
    }

    return selected, nil
}
```

#### Priority (优先级)

```go
type PriorityStrategy struct{}

func (s *PriorityStrategy) SelectProvider(ctx context.Context, providers []*Provider) (*Provider, error) {
    if len(providers) == 0 {
        return nil, errors.New("no providers available")
    }

    // 按优先级排序，选择最高优先级的Provider
    sort.Slice(providers, func(i, j int) bool {
        return providers[i].Priority > providers[j].Priority
    })

    return providers[0], nil
}
```

#### Adaptive (自适应)

```go
type AdaptiveStrategy struct {
    healthChecker *HealthChecker
}

func (s *AdaptiveStrategy) SelectProvider(ctx context.Context, providers []*Provider) (*Provider, error) {
    type scoredProvider struct {
        provider *Provider
        score    float64
    }

    scored := make([]scoredProvider, 0, len(providers))

    for _, p := range providers {
        health := s.healthChecker.GetHealth(p.ID)

        // 计算综合得分
        // 权重: latency=0.4, error_rate=0.3, availability=0.3
        score := 0.0
        score += (1.0 - health.NormalizedLatency) * 0.4
        score += (1.0 - health.ErrorRate) * 0.3
        score += health.Availability * 0.3

        scored = append(scored, scoredProvider{provider: p, score: score})
    }

    // 按得分排序
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    if len(scored) == 0 {
        return nil, errors.New("no providers available")
    }

    return scored[0].provider, nil
}
```

## 2. 熔断器实现

### 2.1 状态机

```
                    ┌─────────────────┐
                    │     Closed      │
                    │  (正常状态)      │
                    └────────┬────────┘
                             │
                             │ 失败次数超过阈值
                             ▼
                    ┌─────────────────┐
           ┌────── │      Open       │ ──────┐
           │       │   (熔断状态)     │       │
           │       └────────┬────────┘       │
           │                │                │
           │  冷却时间结束    │                │ 冷却时间未结束
           │                ▼                │
           │       ┌─────────────────┐       │
           │       │   Half-Open     │       │
           │       │  (半开状态)      │       │
           │       └────────┬────────┘       │
           │                │                │
           │   探测请求成功    │   探测请求失败  │
           │                ▼                │
           │       ┌─────────────────┐       │
           └────── │     Closed      │       │
                    │  (恢复正常)      │       │
                    └─────────────────┘       │
                                              │
                        重置计数器 ────────────┘
```

### 2.2 熔断器实现

```go
type CircuitBreaker struct {
    mu              sync.RWMutex
    state           CircuitState
    failureCount    int
    successCount    int
    lastFailureTime time.Time

    // 配置
    failureThreshold int           // 失败次数阈值
    successThreshold int           // 成功次数阈值
    cooldownDuration time.Duration // 冷却时间

    // 持久化
    persistence CircuitBreakerPersistence
    pubsub     *redis.Client
}

type CircuitState int

const (
    StateClosed   CircuitState = iota
    StateOpen
    StateHalfOpen
)

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    cb.mu.RLock()
    state := cb.state
    cb.mu.RUnlock()

    switch state {
    case StateOpen:
        // 检查是否可以进入半开状态
        if time.Since(cb.lastFailureTime) > cb.cooldownDuration {
            cb.mu.Lock()
            if cb.state == StateOpen {
                cb.state = StateHalfOpen
                cb.persistState()
                cb.publishStateChange()
            }
            cb.mu.Unlock()
            return cb.tryRequest(ctx, fn)
        }
        return ErrCircuitBreakerOpen

    case StateHalfOpen:
        return cb.tryRequest(ctx, fn)

    case StateClosed:
        return cb.tryRequest(ctx, fn)
    }

    return nil
}

func (cb *CircuitBreaker) tryRequest(ctx context.Context, fn func() error) error {
    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.recordFailure()
    } else {
        cb.recordSuccess()
    }

    return err
}

func (cb *CircuitBreaker) recordFailure() {
    cb.failureCount++
    cb.lastFailureTime = time.Now()

    if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
        cb.state = StateOpen
        cb.persistState()
        cb.publishStateChange()
    } else if cb.state == StateHalfOpen {
        cb.state = StateOpen
        cb.persistState()
        cb.publishStateChange()
    }
}

func (cb *CircuitBreaker) recordSuccess() {
    if cb.state == StateHalfOpen {
        cb.successCount++
        if cb.successCount >= cb.successThreshold {
            cb.state = StateClosed
            cb.failureCount = 0
            cb.successCount = 0
            cb.persistState()
            cb.publishStateChange()
        }
    } else if cb.state == StateClosed {
        cb.failureCount = 0
    }
}
```

### 2.3 状态持久化与同步

```go
// 三层存储策略
type CircuitBreakerPersistence struct {
    db    *sql.DB       // 持久化存储
    redis *redis.Client // 分布式共享
    local sync.Map      // 本地缓存
}

func (p *CircuitBreakerPersistence) SaveState(ctx context.Context, providerID string, state CircuitBreakerState) error {
    // 1. 写入数据库
    query := `
        INSERT INTO circuit_breaker_states
            (provider_id, state, failure_count, success_count, last_failure_at, updated_at)
        VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(provider_id)
        DO UPDATE SET
            state = excluded.state,
            failure_count = excluded.failure_count,
            success_count = excluded.success_count,
            last_failure_at = excluded.last_failure_at,
            updated_at = CURRENT_TIMESTAMP
    `

    _, err := p.db.ExecContext(ctx, query,
        providerID, state.State, state.FailureCount,
        state.SuccessCount, state.LastFailureAt)
    if err != nil {
        return err
    }

    // 2. 写入Redis
    key := fmt.Sprintf("circuit_breaker:%s", providerID)
    data, _ := json.Marshal(state)
    p.redis.Set(ctx, key, data, 24*time.Hour)

    // 3. 更新本地缓存
    p.local.Store(providerID, state)

    return nil
}

// Pub/Sub同步
func (p *CircuitBreakerPersistence) SubscribeStateChanges(ctx context.Context, callback func(providerID string, state CircuitBreakerState)) {
    pubsub := p.redis.Subscribe(ctx, "circuit_breaker:state_change")
    defer pubsub.Close()

    for msg := range pubsub.Channel() {
        var event struct {
            ProviderID string              `json:"provider_id"`
            State      CircuitBreakerState `json:"state"`
        }
        json.Unmarshal([]byte(msg.Payload), &event)

        // 更新本地缓存
        p.local.Store(event.ProviderID, event.State)

        // 回调
        callback(event.ProviderID, event.State)
    }
}
```

## 3. 重试机制

### 3.1 指数退避重试

```go
type RetryConfig struct {
    MaxAttempts      int
    InitialBackoff   time.Duration
    MaxBackoff       time.Duration
    BackoffMultiplier float64
    RetryableErrors  []int // 可重试的HTTP状态码
}

func WithRetry(ctx context.Context, config RetryConfig, fn func() error) error {
    var lastErr error

    for attempt := 0; attempt < config.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        lastErr = err

        // 检查是否可重试
        if !isRetryable(err, config.RetryableErrors) {
            return err
        }

        // 计算退避时间
        backoff := config.InitialBackoff * time.Duration(math.Pow(config.BackoffMultiplier, float64(attempt)))
        if backoff > config.MaxBackoff {
            backoff = config.MaxBackoff
        }

        // 等待或取消
        select {
        case <-time.After(backoff):
            continue
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error, retryableErrors []int) bool {
    // 检查HTTP状态码
    if httpErr, ok := err.(*HTTPError); ok {
        for _, code := range retryableErrors {
            if httpErr.StatusCode == code {
                return true
            }
        }
    }

    // 检查网络错误（替代已废弃的net.Error.Temporary()）
    var netErr net.Error
    if errors.As(err, &netErr) {
        return netErr.Timeout()
    }

    // 检查连接错误
    var opErr *net.OpError
    if errors.As(err, &opErr) {
        return true
    }

    // 检查DNS错误
    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return true
    }

    return false
}
```

### 3.2 降级链路

```go
type FallbackChain struct {
    providers []*Provider
    breakers  map[string]*CircuitBreaker  // 每个Provider独立的熔断器
    retry     RetryConfig
}

func (c *FallbackChain) Execute(ctx context.Context, req *Request) (*Response, error) {
    var lastErr error

    for _, provider := range c.providers {
        breaker := c.breakers[provider.ID]

        // 检查熔断器
        if breaker.State() == StateOpen {
            continue
        }

        // 带重试的请求
        // 注意：需要缓存请求Body以支持重试
        var resp *Response
        err := breaker.Call(ctx, func() error {
            // 每次重试创建新的请求（Body已缓存）
            providerReq := req.Clone()
            var providerErr error
            resp, providerErr = provider.Execute(ctx, providerReq)
            return providerErr
        })

        if err == nil {
            return resp, nil
        }

        lastErr = err
    }

    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
```

## 4. 流式响应处理

### 4.1 SSE处理器

```go
type SSEHandler struct {
    writer      http.ResponseWriter
    flusher     http.Flusher
    done        chan struct{}
    mu          sync.Mutex
    closed      bool
}

func NewSSEHandler(w http.ResponseWriter) (*SSEHandler, error) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        return nil, errors.New("streaming not supported")
    }

    // 设置SSE Headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    return &SSEHandler{
        writer:  w,
        flusher: flusher,
        done:    make(chan struct{}),
    }, nil
}

func (h *SSEHandler) WriteEvent(data []byte) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    if h.closed {
        return errors.New("handler closed")
    }

    _, err := fmt.Fprintf(h.writer, "data: %s\n\n", data)
    if err != nil {
        return err
    }

    h.flusher.Flush()
    return nil
}

func (h *SSEHandler) Close() {
    h.mu.Lock()
    defer h.mu.Unlock()

    if !h.closed {
        fmt.Fprintf(h.writer, "data: [DONE]\n\n")
        h.flusher.Flush()
        h.closed = true
        close(h.done)
    }
}
```

### 4.2 客户端断开检测

```go
func (h *SSEHandler) MonitorClient(ctx context.Context) {
    // 方法1: 使用Context Done（推荐，支持HTTP/1.1和HTTP/2）
    go func() {
        select {
        case <-ctx.Done():
            // 客户端断开或请求取消
            h.Close()
        case <-h.done:
            return
        }
    }()

    // 方法2: Keepalive心跳（检测客户端是否存活）
    go func() {
        ticker := time.NewTicker(15 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                h.mu.Lock()
                if !h.closed {
                    // 发送SSE注释作为心跳
                    _, err := fmt.Fprintf(h.writer, ": keepalive\n\n")
                    if err != nil {
                        // 写入失败，客户端可能已断开
                        h.mu.Unlock()
                        h.Close()
                        return
                    }
                    h.flusher.Flush()
                }
                h.mu.Unlock()
            case <-h.done:
                return
            }
        }
    }()
}

// 注意：http.CloseNotifier 自Go 1.11起已废弃，不应使用
// 使用 http.NewResponseController (Go 1.20+) 设置超时
func (h *SSEHandler) SetTimeouts(readTimeout, writeTimeout time.Duration) {
    rc := http.NewResponseController(h.writer)
    rc.SetReadDeadline(time.Now().Add(readTimeout))
    rc.SetWriteDeadline(time.Now().Add(writeTimeout))
}
```

### 4.3 背压控制

```go
type BackpressureHandler struct {
    bufferSize   int
    maxQueueSize int
    queue        chan []byte
    dropped      atomic.Int64
}

func NewBackpressureHandler(bufferSize, maxQueueSize int) *BackpressureHandler {
    return &BackpressureHandler{
        bufferSize:   bufferSize,
        maxQueueSize: maxQueueSize,
        queue:        make(chan []byte, maxQueueSize),
    }
}

func (h *BackpressureHandler) Write(data []byte) (int, error) {
    select {
    case h.queue <- data:
        return len(data), nil
    default:
        // 队列满，丢弃数据
        h.dropped.Add(1)
        return 0, errors.New("queue full, data dropped")
    }
}

func (h *BackpressureHandler) Read() ([]byte, bool) {
    data, ok := <-h.queue
    return data, ok
}
```

## 5. 协议转换

### 5.1 转换器接口

```go
type ProtocolConverter interface {
    // 检测请求格式
    DetectFormat(r *http.Request) ProtocolFormat

    // 转换请求
    ConvertRequest(ctx context.Context, req *http.Request, targetFormat ProtocolFormat) (*ProviderRequest, error)

    // 转换响应
    ConvertResponse(ctx context.Context, resp *ProviderResponse, sourceFormat ProtocolFormat) (*http.Response, error)

    // 转换流式响应
    ConvertStreamResponse(ctx context.Context, stream <-chan *StreamChunk, sourceFormat ProtocolFormat, writer *SSEHandler) error
}
```

### 5.2 OpenAI → Anthropic 转换

```go
type OpenAIToAnthropicConverter struct{}

func (c *OpenAIToAnthropicConverter) ConvertRequest(ctx context.Context, req *http.Request) (*ProviderRequest, error) {
    var openAIReq OpenAIChatRequest
    if err := json.NewDecoder(req.Body).Decode(&openAIReq); err != nil {
        return nil, err
    }

    anthropicReq := &AnthropicRequest{
        Model:     mapModel(openAIReq.Model, "openai", "anthropic"),
        MaxTokens: openAIReq.MaxTokens,
    }

    // 转换消息
    var systemPrompt string
    for _, msg := range openAIReq.Messages {
        if msg.Role == "system" {
            systemPrompt = msg.Content
            continue
        }
        anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
            Role:    msg.Role,
            Content: msg.Content,
        })
    }

    if systemPrompt != "" {
        anthropicReq.System = systemPrompt
    }

    return &ProviderRequest{
        Body: anthropicReq,
    }, nil
}
```

### 5.3 Model映射

```go
var modelMapping = map[string]map[string]string{
    "openai": {
        "anthropic": {
            "gpt-4":          "claude-3-opus-20240229",
            "gpt-4-turbo":    "claude-3-opus-20240229",
            "gpt-3.5-turbo":  "claude-3-haiku-20240307",
        },
        "gemini": {
            "gpt-4":          "gemini-pro",
            "gpt-3.5-turbo":  "gemini-pro",
        },
    },
    "anthropic": {
        "openai": {
            "claude-3-opus-20240229":    "gpt-4",
            "claude-3-sonnet-20240229":  "gpt-4-turbo",
            "claude-3-haiku-20240307":   "gpt-3.5-turbo",
        },
    },
}

func mapModel(sourceModel, sourceFormat, targetFormat string) string {
    if mapping, ok := modelMapping[sourceFormat]; ok {
        if target, ok := mapping[targetFormat]; ok {
            return target
        }
    }
    return sourceModel // 默认返回原模型名
}
```

## 6. 健康检查

### 6.1 Provider健康检查器

```go
type HealthChecker struct {
    interval      time.Duration
    timeout       time.Duration
    healthyThreshold   int
    unhealthyThreshold int

    states map[string]*ProviderHealthState
    mu     sync.RWMutex
}

type ProviderHealthState struct {
    Status          HealthStatus
    ConsecutivePass int
    ConsecutiveFail int
    LastCheckAt     time.Time
    AvgLatency      time.Duration
    ErrorRate       float64
}

func (hc *HealthChecker) CheckProvider(ctx context.Context, provider *Provider) *ProviderHealthState {
    ctx, cancel := context.WithTimeout(ctx, hc.timeout)
    defer cancel()

    // 发送健康检查请求
    start := time.Now()
    err := provider.HealthCheck(ctx)
    latency := time.Since(start)

    hc.mu.Lock()
    defer hc.mu.Unlock()

    state := hc.states[provider.ID]
    if state == nil {
        state = &ProviderHealthState{}
        hc.states[provider.ID] = state
    }

    state.LastCheckAt = time.Now()
    state.AvgLatency = (state.AvgLatency + latency) / 2

    if err != nil {
        state.ConsecutivePass = 0
        state.ConsecutiveFail++

        if state.ConsecutiveFail >= hc.unhealthyThreshold {
            state.Status = HealthStatusUnhealthy
        }
    } else {
        state.ConsecutiveFail = 0
        state.ConsecutivePass++

        if state.ConsecutivePass >= hc.healthyThreshold {
            state.Status = HealthStatusHealthy
        }
    }

    return state
}
```

## 7. 请求上下文传递

### 7.1 Context扩展

```go
type RequestContext struct {
    RequestID    string
    APIKeyID     string
    ProviderID   string
    TokenID      string
    ModelName    string
    SourceFormat ProtocolFormat
    TargetFormat ProtocolFormat
    StartTime    time.Time
    IsStreaming  bool
}

func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
    return context.WithValue(ctx, requestContextKey, reqCtx)
}

func GetRequestContext(ctx context.Context) *RequestContext {
    if reqCtx, ok := ctx.Value(requestContextKey).(*RequestContext); ok {
        return reqCtx
    }
    return nil
}
```

## 8. 优雅关闭

### 8.1 优雅关闭管理器

```go
type GracefulShutdown struct {
    server          *http.Server
    activeRequests  sync.WaitGroup
    shutdownTimeout time.Duration
    shuttingDown    atomic.Bool
}

func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
    gs.shuttingDown.Store(true)

    // 创建带超时的Context
    ctx, cancel := context.WithTimeout(ctx, gs.shutdownTimeout)
    defer cancel()

    // 停止接受新请求
    // (在中间件中检查 shuttingDown 标志)

    // 等待现有请求完成
    done := make(chan struct{})
    go func() {
        gs.activeRequests.Wait()
        close(done)
    }()

    select {
    case <-done:
        // 所有请求已完成
    case <-ctx.Done():
        // 超时，强制关闭
    }

    // 关闭服务器
    return gs.server.Shutdown(ctx)
}

// 中间件：在关闭期间拒绝新请求
func (gs *GracefulShutdown) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if gs.shuttingDown.Load() {
            w.Header().Set("Connection", "close")
            http.Error(w, "server shutting down", http.StatusServiceUnavailable)
            return
        }

        gs.activeRequests.Add(1)
        defer gs.activeRequests.Done()

        next.ServeHTTP(w, r)
    })
}
```

## 9. 请求Body缓存

### 9.1 Body缓冲层

http.Request.Body只能读取一次，重试时需要缓存Body。

```go
type BufferedRequest struct {
    Original *http.Request
    Body     []byte
}

func NewBufferedRequest(r *http.Request) (*BufferedRequest, error) {
    // 读取并缓存Body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return nil, err
    }
    r.Body.Close()

    return &BufferedRequest{
        Original: r,
        Body:     body,
    }, nil
}

func (br *BufferedRequest) Clone() *http.Request {
    // 创建新的请求，使用缓存的Body
    clone := br.Original.Clone(br.Original.Context())
    clone.Body = io.NopCloser(bytes.NewReader(br.Body))
    return clone
}

// 使用示例
func (g *Gateway) handleRequest(w http.ResponseWriter, r *http.Request) {
    // 缓存请求Body
    buffered, err := NewBufferedRequest(r)
    if err != nil {
        http.Error(w, "failed to read request body", http.StatusBadRequest)
        return
    }

    // 重试时使用缓存的Body
    resp, err := WithRetry(ctx, retryConfig, func() (*Response, error) {
        req := buffered.Clone()
        return g.provider.Execute(ctx, req)
    })
}
```

## 10. EWMA健康检查

### 10.1 指数加权移动平均

使用EWMA计算延迟，比简单移动平均更平滑。

```go
type EWMA struct {
    value    float64
    alpha    float64 // 衰减因子 (0, 1)
    started  bool
}

func NewEWMA(alpha float64) *EWMA {
    return &EWMA{alpha: alpha}
}

func (e *EWMA) Add(value float64) {
    if !e.started {
        e.value = value
        e.started = true
        return
    }
    e.value = e.alpha*value + (1-e.alpha)*e.value
}

func (e *EWMA) Value() float64 {
    return e.value
}

// 健康检查器使用EWMA
type HealthChecker struct {
    latencyEWMA map[string]*EWMA  // provider_id -> EWMA
    errorRateEWMA map[string]*EWMA
    alpha       float64  // 0.3 表示最近的值权重30%
}

func (hc *HealthChecker) RecordLatency(providerID string, latency time.Duration) {
    if _, ok := hc.latencyEWMA[providerID]; !ok {
        hc.latencyEWMA[providerID] = NewEWMA(hc.alpha)
    }
    hc.latencyEWMA[providerID].Add(float64(latency.Milliseconds()))
}

func (hc *HealthChecker) GetAvgLatency(providerID string) time.Duration {
    if ewma, ok := hc.latencyEWMA[providerID]; ok {
        return time.Duration(ewma.Value()) * time.Millisecond
    }
    return 0
}
```
