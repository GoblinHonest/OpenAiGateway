package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/example/aigateway/pkg/logger"
	"go.uber.org/zap"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)

		// 对于 /v1/models 路径，打印所有请求头
		if strings.HasPrefix(path, "/v1/models") {
			// 收集所有请求头
			headers := make(map[string]string)
			for k, v := range c.Request.Header {
				if k == "Authorization" || k == "x-api-key" {
					// 脱敏处理
					if len(v) > 0 && len(v[0]) > 4 {
						headers[k] = v[0][:4] + "****"
					} else {
						headers[k] = "***"
					}
				} else {
					headers[k] = strings.Join(v, ", ")
				}
			}
			logger.L.Info("request",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("duration", duration),
				zap.String("client_ip", c.ClientIP()),
				zap.Any("headers", headers),
			)
		} else {
			logger.L.Info("request",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("duration", duration),
				zap.String("client_ip", c.ClientIP()),
			)
		}
	}
}
