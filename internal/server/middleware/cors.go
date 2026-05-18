package middleware

import (
	"net/http"
	"slices"

	"github.com/example/aigateway/internal/config"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a CORS middleware that restricts origins based on config.
// If no allowed origins are configured, it defaults to localhost for development.
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := cfg.Server.CORS.AllowedOrigins
	if len(allowedOrigins) == 0 {
		// Default to localhost origins for development safety
		allowedOrigins = []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	maxAge := cfg.Server.CORS.MaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && slices.Contains(allowedOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")

			if cfg.Server.CORS.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
