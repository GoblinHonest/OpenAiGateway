package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redis   *redis.Client
	enabled bool
}

func NewRateLimiter(redis *redis.Client, enabled bool) *RateLimiter {
	return &RateLimiter{
		redis:   redis,
		enabled: enabled,
	}
}

// Allow checks if a request is allowed under the rate limit.
// Returns (allowed bool, remaining int, err error).
// On Redis failure, returns an error instead of silently allowing.
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	if !rl.enabled {
		return true, limit, nil
	}

	if rl.redis == nil {
		// Redis 不可用时降级放行
		return true, limit, nil
	}

	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	script := redis.NewScript(`
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_ms = tonumber(ARGV[4])

		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
		local count = redis.call('ZCARD', key)

		if count < limit then
			redis.call('ZADD', key, now, now .. ':' .. math.random(1, 1000000))
			redis.call('EXPIRE', key, math.ceil(window_ms / 1000) + 1)
			return {1, limit - count - 1}
		else
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local reset_time = 0
			if #oldest > 0 then
				reset_time = tonumber(oldest[2]) + window_ms
			end
			return {0, reset_time}
		end
	`)

	cmd := script.Run(ctx, rl.redis, []string{key}, windowStart, now, limit, window.Milliseconds())
	if cmd.Err() != nil {
		// Redis 执行失败时降级放行
		return true, limit, nil
	}

	results, err := cmd.Int64Slice()
	if err != nil {
		return false, 0, fmt.Errorf("rate limiter: failed to parse redis result: %w", err)
	}

	if results[0] == 1 {
		return true, int(results[1]), nil
	}

	return false, int(results[1]), nil
}

func (rl *RateLimiter) RateLimitMiddleware(rpm int) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID, _ := c.Get("api_key_id")
		key := "rate_limit:apikey:" + keyID.(string)

		allowed, remaining, err := rl.Allow(c.Request.Context(), key, rpm, time.Minute)
		if err != nil {
			// Redis failure — return 503 Service Unavailable instead of silently allowing
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":      "RATE_LIMITER_UNAVAILABLE",
					"message":   "Rate limiter temporarily unavailable. Please retry later.",
					"type":      "server_error",
					"retryable": true,
					"retry_after": 60,
				},
			})
			return
		}

		if !allowed {
			c.Header("Retry-After", "60")
			c.Header("X-RateLimit-Limit", strconv.Itoa(rpm))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":      "RATE_LIMIT_EXCEEDED",
					"message":   "Rate limit exceeded. Please retry after 60 seconds.",
					"type":      "rate_limit_error",
					"retryable": true,
					"retry_after": 60,
				},
			})
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Next()
	}
}
