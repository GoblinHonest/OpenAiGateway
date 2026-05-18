# API接口设计

## 1. 客户端API (External API)

### 1.1 认证

所有请求需要在Header中携带API Key:

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 1.2 统一请求格式

#### OpenAI格式 (默认)

```http
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer sk-xxx

{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1000
}
```

#### Anthropic格式

```http
POST /v1/messages
Content-Type: application/json
Authorization: Bearer sk-xxx
anthropic-version: 2023-06-01

{
  "model": "claude-3-opus-20240229",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "max_tokens": 1000,
  "stream": false
}
```

#### Gemini格式

```http
POST /v1/models/gemini-pro:generateContent
Content-Type: application/json
Authorization: Bearer sk-xxx

{
  "contents": [
    {
      "parts": [
        {"text": "Hello!"}
      ]
    }
  ],
  "generationConfig": {
    "maxOutputTokens": 1000,
    "temperature": 0.7
  }
}
```

### 1.3 统一响应格式

#### 非流式响应 (OpenAI格式)

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 9,
    "total_tokens": 19
  }
}
```

#### 流式响应 (SSE)

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### 1.4 错误响应

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Please retry after 60 seconds.",
    "type": "rate_limit_error",
    "retryable": true,
    "retry_after": 60
  }
}
```

#### 错误码列表

| 错误码 | HTTP状态码 | 说明 | 可重试 |
|--------|-----------|------|--------|
| `AUTH_INVALID_KEY` | 401 | API Key无效 | 否 |
| `AUTH_KEY_EXPIRED` | 401 | API Key已过期 | 否 |
| `RATE_LIMIT_EXCEEDED` | 429 | 超过速率限制 | 是 |
| `QUOTA_EXCEEDED` | 429 | 配额已用尽 | 否 |
| `TOKEN_EXHAUSTED` | 503 | 所有Token已耗尽 | 是 |
| `PROVIDER_UNAVAILABLE` | 503 | 服务商不可用 | 是 |
| `CIRCUIT_BREAKER_OPEN` | 503 | 熔断器已打开 | 是 |
| `INVALID_REQUEST` | 400 | 请求参数无效 | 否 |
| `MODEL_NOT_FOUND` | 404 | 模型不存在 | 否 |
| `INTERNAL_ERROR` | 500 | 内部错误 | 是 |

## 2. 管理API (Admin API)

### 2.1 认证

管理API使用独立的Admin Token:

```http
Authorization: Bearer admin-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 2.2 Provider管理

#### 创建Provider

```http
POST /admin/v1/providers
Content-Type: application/json

{
  "name": "OpenAI",
  "base_url": "https://api.openai.com",
  "supported_formats": ["openai"],
  "endpoints": {
    "chat": "/v1/chat/completions",
    "embeddings": "/v1/embeddings"
  },
  "rate_limit_config": {
    "requests_per_minute": 60,
    "tokens_per_minute": 90000
  },
  "timeout_config": {
    "connect_timeout": 10,
    "read_timeout": 60
  },
  "retry_config": {
    "max_attempts": 3,
    "initial_backoff": 1,
    "max_backoff": 10
  }
}
```

#### 列出Providers

```http
GET /admin/v1/providers?status=active&page=1&page_size=20
```

#### 更新Provider

```http
PUT /admin/v1/providers/{provider_id}
Content-Type: application/json

{
  "status": "inactive",
  "rate_limit_config": {
    "requests_per_minute": 100
  }
}
```

#### 删除Provider

```http
DELETE /admin/v1/providers/{provider_id}
```

### 2.3 Token管理

#### 添加Token

```http
POST /admin/v1/tokens
Content-Type: application/json

{
  "provider_id": "openai-001",
  "name": "OpenAI Token 1",
  "token_value": "sk-proj-xxxxxxxxxxxxxxxxxxxxx",
  "quota_total": 1000000,
  "quota_reset_at": "2026-06-01T00:00:00Z"
}
```

#### 列出Tokens

```http
GET /admin/v1/tokens?provider_id=openai-001&status=active
```

#### 更新Token

```http
PUT /admin/v1/tokens/{token_id}
Content-Type: application/json

{
  "status": "inactive",
  "quota_total": 2000000
}
```

#### 删除Token

```http
DELETE /admin/v1/tokens/{token_id}
```

### 2.4 Model管理

#### 创建Model

```http
POST /admin/v1/models
Content-Type: application/json

{
  "name": "gpt-4",
  "display_name": "GPT-4",
  "model_type": "chat",
  "context_window": 8192,
  "max_output_tokens": 4096,
  "input_price_per_1k": 0.03,
  "output_price_per_1k": 0.06,
  "enabled": true
}
```

#### 绑定Model到Provider

```http
POST /admin/v1/model-provider-bindings
Content-Type: application/json

{
  "model_id": "gpt-4",
  "provider_id": "openai-001",
  "weight": 10,
  "priority": 1,
  "enabled": true
}
```

### 2.5 Group管理

#### 创建Group

```http
POST /admin/v1/groups
Content-Type: application/json

{
  "name": "production",
  "description": "Production environment group",
  "load_balance_strategy": "weighted",
  "enabled": true
}
```

#### 列出Groups

```http
GET /admin/v1/groups?enabled=true
```

### 2.6 API Key管理

#### 生成API Key

```http
POST /admin/v1/api-keys
Content-Type: application/json

{
  "name": "Client App 1",
  "group_id": "production",
  "rate_limit_config": {
    "requests_per_minute": 60,
    "tokens_per_minute": 100000
  },
  "quota_config": {
    "total_tokens": 10000000,
    "reset_period": "monthly"
  },
  "expires_at": "2027-01-01T00:00:00Z"
}
```

响应:

```json
{
  "id": "key-123",
  "key": "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "key_prefix": "sk-xxxx",
  "name": "Client App 1",
  "group_id": "production",
  "status": "active",
  "created_at": "2026-05-15T10:00:00Z",
  "expires_at": "2027-01-01T00:00:00Z"
}
```

**注意**: `key`字段只在创建时返回一次，后续无法再次获取。

#### 列出API Keys

```http
GET /admin/v1/api-keys?status=active&group_id=production
```

#### 撤销API Key

```http
DELETE /admin/v1/api-keys/{key_id}
```

### 2.7 统计查询

#### 请求统计

```http
GET /admin/v1/stats/requests?start_time=2026-05-01T00:00:00Z&end_time=2026-05-15T23:59:59Z&group_by=provider,model
```

响应:

```json
{
  "total_requests": 1000000,
  "success_requests": 995000,
  "failed_requests": 5000,
  "success_rate": 99.5,
  "total_tokens": 50000000,
  "total_cost": 1500.00,
  "breakdown": [
    {
      "provider": "openai",
      "model": "gpt-4",
      "requests": 500000,
      "tokens": 25000000,
      "cost": 750.00
    }
  ]
}
```

#### 仪表盘概览

```http
GET /admin/v1/dashboard/overview
```

响应:

```json
{
  "overview": {
    "todayRequests": 90,
    "sevenDayRequests": 90,
    "thirtyDayRequests": 90,
    "todayInputTokens": 1028588,
    "todayOutputTokens": 6824,
    "sevenDayInputTokens": 1028588,
    "sevenDayOutputTokens": 6824,
    "thirtyDayInputTokens": 1028588,
    "thirtyDayOutputTokens": 6824,
    "todayFailedRequests": 60,
    "failureRate": 0.67
  },
  "trend": [
    {"time": "2026-05-14", "requests": 1, "tokens": 469},
    {"time": "2026-05-15", "requests": 1136, "tokens": 79925526}
  ],
  "keyTokenDistribution": [
    {"name": "生产", "tokens": 74528296}
  ],
  "modelDistribution": [
    {"name": "deepseek-v4-flash", "tokens": 77662403},
    {"name": "step-3.5-flash", "tokens": 2263123}
  ],
  "providerDistribution": [
    {"name": "opencode", "tokens": 76526461},
    {"name": "modelscope", "tokens": 2263123},
    {"name": "sensenova", "tokens": 1135942}
  ]
}
```

**查询参数**:
- `days`: 趋势天数 (默认90)

#### Token使用情况

```http
GET /admin/v1/stats/tokens?provider_id=openai-001
```

响应:

```json
{
  "tokens": [
    {
      "id": "token-001",
      "name": "OpenAI Token 1",
      "quota_total": 1000000,
      "quota_used": 750000,
      "quota_remaining": 250000,
      "success_rate": 99.8,
      "last_used_at": "2026-05-15T10:30:00Z"
    }
  ]
}
```

#### Provider健康状态

```http
GET /admin/v1/health/providers
```

响应:

```json
{
  "providers": [
    {
      "id": "openai-001",
      "name": "OpenAI",
      "status": "healthy",
      "circuit_breaker_state": "closed",
      "avg_latency_ms": 250,
      "error_rate": 0.2,
      "available_tokens": 5,
      "last_check_at": "2026-05-15T10:35:00Z"
    }
  ]
}
```

### 2.8 日志查询

#### 查询请求日志

```http
GET /admin/v1/logs/requests?start_time=2026-05-15T00:00:00Z&limit=100&api_key_id=key-123&success=false
```

响应:

```json
{
  "items": [
    {
      "id": 6860,
      "interfaceType": "openai",
      "route": "/chat/completions",
      "apiKeyName": "生产",
      "modelName": "deepseek-v4-flash",
      "providerName": "sensenova",
      "clientIp": "171.36.36.177",
      "stream": true,
      "success": false,
      "firstTokenLatencyMs": 45,
      "totalLatencyMs": 45,
      "errorMessage": "{\"error\":{\"message\":\"field MaxTokens invalid\"}}",
      "inputTokens": 0,
      "outputTokens": 0,
      "cacheReadInputTokens": 0,
      "detailId": "4e56a169330afa8e62851367eb9f67b8",
      "createdAt": "2026-05-15T21:01:27.767332+08:00"
    }
  ],
  "total": 1000
}
```

**查询参数**:
- `start_time`: 开始时间
- `end_time`: 结束时间
- `api_key_id`: API Key ID
- `api_key_name`: API Key名称
- `provider_name`: 服务商名称
- `model_name`: 模型名称
- `interface_type`: 接口类型 (openai, anthropic, gemini)
- `success`: 是否成功
- `cache_hit`: 是否缓存命中
- `limit`: 返回条数 (默认100)
- `cursor`: 分页游标（用于大数据量分页）

#### 日志详情

```http
GET /admin/v1/logs/requests/{request_id}
```

响应:

```json
{
  "id": 6860,
  "requestId": "4e56a169330afa8e62851367eb9f67b8",
  "timestamp": "2026-05-15T21:01:27.767+08:00",
  "clientIp": "171.36.36.177",
  "apiKeyName": "生产",
  "interfaceType": "openai",
  "route": "/chat/completions",
  "modelName": "deepseek-v4-flash",
  "providerName": "sensenova",
  "stream": true,
  "maxTokens": 384000,
  "success": false,
  "firstTokenLatencyMs": 45,
  "totalLatencyMs": 45,
  "inputTokens": 0,
  "outputTokens": 0,
  "cacheReadInputTokens": 0,

  "requestHeaders": {
    "Accept": ["application/json"],
    "Content-Type": ["application/json"],
    "Authorization": ["Bearer sk-30953c...cf0"],
    "User-Agent": ["OpenAI/JS 6.37.0"]
  },
  "requestBody": "{\"model\":\"deepseek-v4-flash\",\"messages\":[...],\"stream\":true}",
  "requestBodyParsed": {
    "model": "deepseek-v4-flash",
    "stream": true,
    "max_tokens": 384000,
    "thinking": { "type": "disabled" },
    "tool_choice": "auto",
    "messages": [
      {
        "role": "system",
        "content": "You are a personal assistant...",
        "tokenEstimate": 9245
      },
      {
        "role": "user",
        "content": "[cron:2cc24ef2...] 运行 session 状态备份..."
      }
    ],
    "tools": [
      { "name": "agents_list", "description": "List OpenClaw agent ids..." },
      { "name": "read", "description": "Read the contents of a file..." },
      { "name": "write", "description": "Write content to a file..." }
    ],
    "toolsCount": 28
  },
  "requestBodyTruncated": false,

  "responseHeaders": {
    "Content-Type": ["application/json"],
    "X-Request-Id": ["70f83642-f1a7-44fb-b798-9c7c8913d0db"]
  },
  "responseBody": "{\"error\":{\"message\":\"field MaxTokens invalid, should be in [1, 65536]\"}}",
  "responseBodyTruncated": false,

  "errorMessage": "field MaxTokens invalid, should be in [1, 65536]"
}
```

**字段说明**:
- `requestHeaders`: 完整请求头（敏感信息脱敏）
- `requestBody`: 原始请求体JSON字符串
- `requestBodyParsed`: 解析后的可读格式
  - `messages[].tokenEstimate`: 每条消息的token估算
  - `toolsCount`: 工具数量
- `requestBodyTruncated`: 请求体是否被截断
- `responseHeaders`: 完整响应头
- `responseBody`: 原始响应体JSON字符串
- `responseBodyTruncated`: 响应体是否被截断

## 3. 协议转换

### 3.1 自动协议检测

网关根据请求路径和Content-Type自动检测协议:

- `/v1/chat/completions` → OpenAI格式
- `/v1/messages` → Anthropic格式
- `/v1/models/*/generateContent` → Gemini格式

### 3.2 协议转换流程

```
Client Request (OpenAI)
    ↓
Gateway (检测格式)
    ↓
Provider支持OpenAI?
    ├─ Yes → 直接转发
    └─ No → 转换为Provider格式
        ↓
    Provider Response
        ↓
    转换回OpenAI格式
        ↓
    Client Response
```

### 3.3 转换示例

#### OpenAI → Anthropic

请求转换:

```json
// OpenAI格式
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are helpful."},
    {"role": "user", "content": "Hello"}
  ]
}

// 转换为Anthropic格式
{
  "model": "claude-3-opus-20240229",
  "system": "You are helpful.",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

响应转换:

```json
// Anthropic响应
{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [
    {"type": "text", "text": "Hello!"}
  ],
  "usage": {
    "input_tokens": 10,
    "output_tokens": 5
  }
}

// 转换为OpenAI格式
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "Hello!"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 5,
    "total_tokens": 15
  }
}
```

## 4. 健康检查API

```http
GET /health
```

响应:

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "providers": {
      "total": 3,
      "healthy": 3,
      "degraded": 0,
      "down": 0
    }
  }
}
```

## 5. Metrics API (Prometheus)

```http
GET /metrics
```

响应 (Prometheus格式):

```
# HELP gateway_requests_total Total number of requests
# TYPE gateway_requests_total counter
gateway_requests_total{provider="openai",model="gpt-4",status="success"} 1000000

# HELP gateway_request_duration_seconds Request duration in seconds
# TYPE gateway_request_duration_seconds histogram
gateway_request_duration_seconds_bucket{provider="openai",model="gpt-4",le="0.1"} 100000
gateway_request_duration_seconds_bucket{provider="openai",model="gpt-4",le="0.5"} 500000
gateway_request_duration_seconds_bucket{provider="openai",model="gpt-4",le="1.0"} 900000

# HELP gateway_tokens_total Total number of tokens processed
# TYPE gateway_tokens_total counter
gateway_tokens_total{provider="openai",model="gpt-4",type="input"} 50000000
gateway_tokens_total{provider="openai",model="gpt-4",type="output"} 25000000
```

## 6. 速率限制

### 6.1 限制维度

- **API Key级别**: 每个API Key独立限制
- **Group级别**: 同一Group内所有API Key共享限制
- **Provider级别**: 每个Provider的全局限制

### 6.2 限制响应

当触发速率限制时:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1715774400

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Please retry after 60 seconds.",
    "retry_after": 60
  }
}
```

## 7. 请求追踪

### 7.1 追踪Header

每个请求都会返回追踪信息:

```http
X-Request-ID: req-abc123def456
X-Provider: openai
X-Token-ID: token-001
X-Latency-Ms: 1500
X-Cache-Hit: false
```

### 7.2 详细日志

客户端可以请求详细日志:

```http
GET /admin/v1/requests/{request_id}
```

响应:

```json
{
  "request_id": "req-abc123",
  "timestamp": "2026-05-15T10:30:00.123Z",
  "client_ip": "192.168.1.100",
  "api_key_id": "key-123",
  "api_key_name": "生产",
  "model_name": "gpt-4",
  "provider_name": "OpenAI",
  "token_id": "token-001",
  "interface_type": "openai",
  "route": "/chat/completions",
  "source_format": "openai",
  "target_format": "openai",
  "protocol_converted": false,
  "is_streaming": false,
  "input_tokens": 100,
  "output_tokens": 200,
  "total_tokens": 300,
  "cache_hit": false,
  "cache_type": "none",
  "cache_read_input_tokens": 0,
  "first_byte_latency_ms": 500,
  "total_latency_ms": 1500,
  "provider_latency_ms": 1400,
  "status_code": 200,
  "success": true,
  "retry_count": 0,
  "fallback_used": false,
  "estimated_cost": 0.024
}
```

## 8. 缓存管理API

### 8.1 查询缓存配置

```http
GET /admin/v1/cache/config
```

响应:

```json
{
  "enabled": true,
  "policy": {
    "min_tokens": 10,
    "max_cache_size": 100000,
    "default_ttl": "1h",
    "respect_no_cache": true,
    "cache_stream": false
  },
  "local_cache": {
    "enabled": true,
    "max_size": 1000,
    "ttl": "5m"
  },
  "redis_cache": {
    "enabled": true,
    "prefix": "llm_cache:",
    "ttl": "1h"
  }
}
```

### 8.2 更新缓存配置

```http
PUT /admin/v1/cache/config
Content-Type: application/json

{
  "enabled": true,
  "policy": {
    "min_tokens": 10,
    "max_cache_size": 100000,
    "default_ttl": "1h"
  }
}
```

### 8.3 查询缓存统计

```http
GET /admin/v1/cache/stats
```

响应:

```json
{
  "enabled": true,
  "hit_rate": 45.2,
  "total_hits": 125000,
  "total_misses": 151000,
  "cached_tokens": 50000000,
  "saved_tokens": 22500000,
  "saved_cost": 675.00,
  "local_cache": {
    "size": 987,
    "max_size": 1000,
    "hit_rate": 65.3
  },
  "redis_cache": {
    "size": 45231,
    "max_size": 100000,
    "hit_rate": 38.7
  },
  "top_models": [
    {
      "model": "gpt-4",
      "hit_rate": 52.3,
      "saved_tokens": 15000000
    }
  ]
}
```

### 8.4 清除缓存

```http
DELETE /admin/v1/cache?model=gpt-4
DELETE /admin/v1/cache?prefix=cache:a1b2c3
DELETE /admin/v1/cache  # 清除所有
```

### 8.5 查询缓存条目

```http
GET /admin/v1/cache/entries?model=gpt-4&limit=100
```

响应:

```json
{
  "entries": [
    {
      "cache_key": "cache:a1b2c3d4e5f6...",
      "model": "gpt-4",
      "created_at": "2026-05-15T10:00:00Z",
      "expires_at": "2026-05-15T11:00:00Z",
      "hit_count": 25,
      "input_tokens": 100,
      "output_tokens": 200,
      "total_tokens": 300
    }
  ],
  "total": 1234
}
```

## 9. 客户端缓存控制

### 9.1 请求头控制

```http
# 强制不使用缓存
POST /v1/chat/completions
Cache-Control: no-cache

# 强制刷新缓存（请求并更新缓存）
POST /v1/chat/completions
Cache-Control: no-cache, force-update
```

### 9.2 响应头

```http
# 缓存命中
HTTP/1.1 200 OK
X-Cache: HIT
X-Cache-Type: local
X-Cache-Key: cache:a1b2c3d4e5f6...

# 缓存未命中
HTTP/1.1 200 OK
X-Cache: MISS
```

### 9.3 响应体扩展

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "model": "gpt-4",
  "choices": [...],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 200,
    "total_tokens": 300,
    "cache_read_input_tokens": 100,
    "cache_creation_input_tokens": 0
  },
  "x_cache": {
    "hit": true,
    "type": "local",
    "key": "cache:a1b2c3d4e5f6..."
  }
}
```
