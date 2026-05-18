package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a Redis-backed LLM response cache.
// It caches full provider responses keyed by (provider, model, request body hash).
type Cache struct {
	client  *redis.Client
	ttl     time.Duration
	enabled bool
}

// NewCache creates a new Cache instance.
func NewCache(client *redis.Client, ttl time.Duration, enabled bool) *Cache {
	return &Cache{
		client:  client,
		ttl:     ttl,
		enabled: enabled,
	}
}

// CacheEntry is what we store in Redis — the provider's raw response body plus metadata.
type CacheEntry struct {
	Body       []byte    `json:"body"`
	StatusCode int       `json:"status_code"`
	CachedAt   time.Time `json:"cached_at"`
}

// BuildCacheKey generates a deterministic cache key from provider, model and request body.
func BuildCacheKey(provider, model string, requestBody []byte) string {
	hash := sha256.Sum256(requestBody)
	return fmt.Sprintf("llm_cache:%s:%s:%x", provider, model, hash[:16])
}

// Get retrieves a cached response. Returns nil if not found or cache disabled.
func (c *Cache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	if !c.enabled || c.client == nil {
		return nil, nil
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Set stores a response in the cache.
func (c *Cache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	if !c.enabled || c.client == nil {
		return nil
	}

	entry.CachedAt = time.Now()
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// Delete removes a single cache entry.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if !c.enabled || c.client == nil {
		return nil
	}

	return c.client.Del(ctx, key).Err()
}

// Clear removes all cached LLM responses (keys matching llm_cache:*).
func (c *Cache) Clear(ctx context.Context) error {
	if !c.enabled || c.client == nil {
		return nil
	}

	iter := c.client.Scan(ctx, 0, "llm_cache:*", 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Stats returns cache hit/miss/size statistics.
func (c *Cache) Stats(ctx context.Context) (totalKeys int64, err error) {
	if !c.enabled || c.client == nil {
		return 0, nil
	}

	var count int64
	iter := c.client.Scan(ctx, 0, "llm_cache:*", 100).Iterator()
	for iter.Next(ctx) {
		count++
	}
	return count, iter.Err()
}

// Enabled returns whether the cache is enabled.
func (c *Cache) Enabled() bool {
	return c.enabled
}

// SetEnabled enables or disables the cache at runtime.
func (c *Cache) SetEnabled(enabled bool) {
	c.enabled = enabled
}

// TTL returns the current cache TTL.
func (c *Cache) TTL() time.Duration {
	return c.ttl
}

// SetTTL updates the cache TTL at runtime.
func (c *Cache) SetTTL(ttl time.Duration) {
	c.ttl = ttl
}

// ListEntries returns paginated cache entries from Redis.
func (c *Cache) ListEntries(ctx context.Context, page, pageSize int) ([]map[string]any, int64, error) {
	if !c.enabled || c.client == nil {
		return nil, 0, nil
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var allKeys []string
	iter := c.client.Scan(ctx, 0, "llm_cache:*", 100).Iterator()
	for iter.Next(ctx) {
		allKeys = append(allKeys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, 0, err
	}

	total := int64(len(allKeys))
	start := (page - 1) * pageSize
	if start > len(allKeys) {
		return []map[string]any{}, total, nil
	}
	end := start + pageSize
	if end > len(allKeys) {
		end = len(allKeys)
	}

	pageKeys := allKeys[start:end]
	entries := make([]map[string]any, 0, len(pageKeys))
	for _, key := range pageKeys {
		entry := map[string]any{"key": key}

		data, err := c.client.Get(ctx, key).Bytes()
		if err != nil {
			entry["error"] = err.Error()
			entries = append(entries, entry)
			continue
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			entry["error"] = err.Error()
		} else {
			entry["status_code"] = cacheEntry.StatusCode
			entry["cached_at"] = cacheEntry.CachedAt
			entry["body_size"] = len(cacheEntry.Body)
		}

		ttl, err := c.client.TTL(ctx, key).Result()
		if err == nil {
			entry["ttl"] = ttl.String()
		}

		entries = append(entries, entry)
	}

	return entries, total, nil
}

// ============================================================
// Provider-side cache usage parsing (Prompt Caching metadata)
// ============================================================

// CacheUsage 服务商返回的缓存使用信息
type CacheUsage struct {
	CacheReadTokens     int  `json:"cache_read_input_tokens"`     // 从缓存读取的token数
	CacheCreationTokens int  `json:"cache_creation_input_tokens"` // 创建缓存的token数
	IsCacheHit          bool `json:"is_cache_hit"`                // 是否缓存命中
}

// ParseCacheUsage 解析服务商返回的缓存使用信息
func ParseCacheUsage(provider string, usage map[string]any) *CacheUsage {
	if usage == nil {
		return &CacheUsage{}
	}

	switch provider {
	case "openai":
		return parseOpenAICacheUsage(usage)
	case "anthropic":
		return parseAnthropicCacheUsage(usage)
	case "deepseek":
		return parseDeepSeekCacheUsage(usage)
	default:
		return &CacheUsage{}
	}
}

// parseOpenAICacheUsage 解析OpenAI缓存信息
// OpenAI自动缓存，返回 prompt_tokens_details.cached_tokens
func parseOpenAICacheUsage(usage map[string]any) *CacheUsage {
	result := &CacheUsage{}

	details, ok := usage["prompt_tokens_details"].(map[string]any)
	if !ok {
		return result
	}

	cached, ok := details["cached_tokens"].(float64)
	if ok {
		result.CacheReadTokens = int(cached)
		result.IsCacheHit = cached > 0
	}

	return result
}

// parseAnthropicCacheUsage 解析Anthropic缓存信息
// Anthropic返回 cache_read_input_tokens 和 cache_creation_input_tokens
func parseAnthropicCacheUsage(usage map[string]any) *CacheUsage {
	result := &CacheUsage{}

	if readTokens, ok := usage["cache_read_input_tokens"].(float64); ok {
		result.CacheReadTokens = int(readTokens)
		result.IsCacheHit = readTokens > 0
	}

	if createTokens, ok := usage["cache_creation_input_tokens"].(float64); ok {
		result.CacheCreationTokens = int(createTokens)
	}

	return result
}

// parseDeepSeekCacheUsage 解析DeepSeek缓存信息
// DeepSeek返回 prompt_cache_hit_tokens 和 prompt_cache_miss_tokens
func parseDeepSeekCacheUsage(usage map[string]any) *CacheUsage {
	result := &CacheUsage{}

	if hitTokens, ok := usage["prompt_cache_hit_tokens"].(float64); ok {
		result.CacheReadTokens = int(hitTokens)
		result.IsCacheHit = hitTokens > 0
	}

	return result
}

// ExtractUsageFromResponse 从响应体提取usage信息
func ExtractUsageFromResponse(provider string, body []byte) (inputTokens, outputTokens, cacheReadTokens int) {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0
	}

	usage, ok := resp["usage"].(map[string]any)
	if !ok {
		return 0, 0, 0
	}

	switch provider {
	case "openai":
		if v, ok := usage["prompt_tokens"].(float64); ok {
			inputTokens = int(v)
		}
		if v, ok := usage["completion_tokens"].(float64); ok {
			outputTokens = int(v)
		}
		if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
			if v, ok := details["cached_tokens"].(float64); ok {
				cacheReadTokens = int(v)
			}
		}

	case "anthropic":
		if v, ok := usage["input_tokens"].(float64); ok {
			inputTokens = int(v)
		}
		if v, ok := usage["output_tokens"].(float64); ok {
			outputTokens = int(v)
		}
		if v, ok := usage["cache_read_input_tokens"].(float64); ok {
			cacheReadTokens = int(v)
		}

	default:
		if v, ok := usage["prompt_tokens"].(float64); ok {
			inputTokens = int(v)
		}
		if v, ok := usage["completion_tokens"].(float64); ok {
			outputTokens = int(v)
		}
	}

	return inputTokens, outputTokens, cacheReadTokens
}

// BuildCacheStatsKey 构建缓存统计的Redis Key
func BuildCacheStatsKey(provider string) string {
	return fmt.Sprintf("cache_stats:%s", provider)
}
