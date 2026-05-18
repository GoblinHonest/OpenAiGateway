# 大模型缓存设计（Prompt Caching）

## 1. 概述

大模型缓存是指**服务商侧的Prompt Caching功能**，不是网关侧的响应缓存。

### 1.1 工作原理

```
客户端请求（含cache_control参数）
    ↓
网关透传缓存参数到服务商
    ↓
服务商检查缓存
    ↓
命中 → 返回 cached_tokens，费用减半
未命中 → 正常计费，创建缓存
    ↓
网关记录缓存状态到日志
```

### 1.2 支持的服务商

| 服务商 | 支持状态 | 缓存参数 | 计费优惠 |
|--------|----------|----------|----------|
| OpenAI | 支持 | 自动缓存长prompt | 缓存部分半价 |
| Anthropic | 支持 | `cache_control` | 缓存部分半价 |
| DeepSeek | 支持 | 自动缓存 | 缓存部分半价 |
| Gemini | 支持 | `cachedContent` | 缓存部分减费 |

## 2. 缓存参数透传

### 2.1 Anthropic 缓存控制

Anthropic支持在消息中指定缓存控制：

```json
// 客户端请求
{
  "model": "claude-3-opus-20240229",
  "max_tokens": 1024,
  "system": [
    {
      "type": "text",
      "text": "你是一个专业的助手...",
      "cache_control": {"type": "ephemeral"}
    }
  ],
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

**网关处理**：直接透传，不做修改。

### 2.2 OpenAI 缓存

OpenAI自动缓存超过1024 tokens的prompt，无需特殊参数。

**网关处理**：无需处理，服务商自动生效。

### 2.3 DeepSeek 缓存

DeepSeek自动缓存，无需特殊参数。

**网关处理**：无需处理，服务商自动生效。

## 3. 缓存状态解析

### 3.1 响应中的缓存信息

```json
// OpenAI 响应
{
  "usage": {
    "prompt_tokens": 1000,
    "completion_tokens": 200,
    "total_tokens": 1200,
    "prompt_tokens_details": {
      "cached_tokens": 800
    }
  }
}

// Anthropic 响应
{
  "usage": {
    "input_tokens": 1000,
    "output_tokens": 200,
    "cache_read_input_tokens": 800,
    "cache_creation_input_tokens": 200
  }
}
```

### 3.2 统一解析接口

```go
type CacheUsage struct {
    CacheReadTokens    int  // 从缓存读取的token数
    CacheCreationTokens int  // 创建缓存的token数
    IsCacheHit         bool // 是否缓存命中
}

func ParseCacheUsage(provider string, usage map[string]interface{}) *CacheUsage {
    switch provider {
    case "openai":
        // prompt_tokens_details.cached_tokens
        details, _ := usage["prompt_tokens_details"].(map[string]interface{})
        cached, _ := details["cached_tokens"].(float64)
        return &CacheUsage{
            CacheReadTokens: int(cached),
            IsCacheHit:      cached > 0,
        }

    case "anthropic":
        // cache_read_input_tokens
        readTokens, _ := usage["cache_read_input_tokens"].(float64)
        createTokens, _ := usage["cache_creation_input_tokens"].(float64)
        return &CacheUsage{
            CacheReadTokens:     int(readTokens),
            CacheCreationTokens: int(createTokens),
            IsCacheHit:          readTokens > 0,
        }

    case "deepseek":
        // prompt_cache_hit_tokens / prompt_cache_miss_tokens
        hitTokens, _ := usage["prompt_cache_hit_tokens"].(float64)
        return &CacheUsage{
            CacheReadTokens: int(hitTokens),
            IsCacheHit:      hitTokens > 0,
        }

    default:
        return &CacheUsage{}
    }
}
```

## 4. 日志记录

### 4.1 request_logs 表字段

```sql
-- 已在数据库设计中添加
cache_read_input_tokens INT DEFAULT 0,  -- 从缓存读取的input tokens
```

### 4.2 日志记录示例

```json
{
  "request_id": "req-abc123",
  "model_name": "gpt-4",
  "provider_name": "openai",
  "input_tokens": 1000,
  "output_tokens": 200,
  "cache_read_input_tokens": 800,  // 800 tokens从缓存读取
  "success": true,
  "total_latency_ms": 500
}
```

## 5. 仪表盘统计

### 5.1 缓存概览

```http
GET /admin/v1/dashboard/overview
```

响应中包含缓存统计：

```json
{
  "overview": {
    "todayRequests": 1000,
    "todayCacheHitTokens": 500000,  // 今日缓存命中token数
    "todayCacheHitRate": 45.2       // 今日缓存命中率
  }
}
```

### 5.2 缓存趋势

```json
{
  "trend": [
    {
      "time": "2026-05-15",
      "requests": 1000,
      "tokens": 1000000,
      "cache_hit_tokens": 500000
    }
  ]
}
```

## 6. 成本计算

### 6.1 缓存token计费规则

| 服务商 | 普通token | 缓存读取token | 缓存创建token |
|--------|-----------|---------------|---------------|
| OpenAI | $0.03/1K | $0.015/1K (半价) | $0.03/1K |
| Anthropic | $0.015/1K | $0.001875/1K (1/8) | $0.01875/1K |

### 6.2 成本计算示例

```go
func CalculateCost(provider string, model string, usage *CacheUsage, inputTokens int) float64 {
    price := GetModelPrice(provider, model)

    // 普通token数 = 总输入 - 缓存读取
    normalTokens := inputTokens - usage.CacheReadTokens

    // 计算成本
    cost := float64(normalTokens) * price.InputPricePer1K / 1000
    cost += float64(usage.CacheReadTokens) * price.CacheReadPricePer1K / 1000

    return cost
}
```

## 7. 日志查询API

### 7.1 查询参数

```http
GET /admin/v1/logs/requests?cache_hit=true
```

**新增查询参数**:
- `cache_hit`: 是否缓存命中 (true/false)

### 7.2 响应字段

```json
{
  "items": [
    {
      "id": 6860,
      "modelName": "gpt-4",
      "providerName": "openai",
      "inputTokens": 1000,
      "outputTokens": 200,
      "cacheReadInputTokens": 800,
      "success": true,
      "createdAt": "2026-05-15T21:01:27Z"
    }
  ]
}
```

## 8. 网关实现要点

### 8.1 请求透传

```go
func (g *Gateway) handleRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // 直接透传请求，包括cache_control参数
    // 不修改请求内容
    resp, err := g.provider.Execute(ctx, req)
    if err != nil {
        return nil, err
    }

    // 解析缓存状态
    cacheUsage := ParseCacheUsage(g.provider.Name, resp.Usage)

    // 记录日志（包含缓存信息）
    g.logRequest(ctx, req, resp, cacheUsage)

    return resp, nil
}
```

### 8.2 响应透传

```go
// 网关直接透传服务商响应，不做修改
// 客户端直接看到服务商返回的缓存信息
```

## 9. 配置

```yaml
# config.yaml
cache:
  # 是否记录缓存统计
  enable_stats: true

  # 是否在响应中包含缓存信息
  pass_through: true
```

## 10. 注意事项

1. **不要自己做响应缓存** - Prompt Caching是服务商侧的功能
2. **直接透传参数** - 网关不需要修改请求/响应
3. **记录缓存状态** - 在日志中记录，用于统计和成本计算
4. **缓存是免费功能** - 不需要额外配置，服务商自动处理
