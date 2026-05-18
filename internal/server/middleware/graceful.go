package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GracefulShutdownMiddleware(srv interface{ IsShuttingDown() bool }) gin.HandlerFunc {
	return func(c *gin.Context) {
		if srv.IsShuttingDown() {
			c.Header("Connection", "close")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":    "SERVER_SHUTTING_DOWN",
					"message": "Server is shutting down, please retry later",
					"type":    "server_error",
				},
			})
			return
		}
		c.Next()
	}
}
