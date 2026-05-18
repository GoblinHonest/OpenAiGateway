package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/example/aigateway/internal/config"
	"github.com/gin-gonic/gin"
)

func AdminAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing admin token"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Use constant-time comparison to prevent timing attacks.
		// Compare lengths first to avoid panics on length mismatch.
		if len(token) != len(cfg.Auth.AdminToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin token"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.Auth.AdminToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin token"})
			return
		}

		c.Next()
	}
}
