package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/example/aigateway/pkg/logger"
	"go.uber.org/zap"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.L.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
						"type":    "server_error",
					},
				})
			}
		}()
		c.Next()
	}
}
