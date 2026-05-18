package middleware

import (
	"net/http"
	"strings"

	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(gatewayService *service.GatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTH_INVALID_KEY",
					"message": "Missing authorization header",
					"type":    "auth_error",
				},
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTH_INVALID_KEY",
					"message": "Invalid authorization format",
					"type":    "auth_error",
				},
			})
			return
		}

		apiKey, err := gatewayService.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTH_INVALID_KEY",
					"message": err.Error(),
					"type":    "auth_error",
				},
			})
			return
		}

		c.Set("api_key_id", apiKey.ID)
		c.Set("api_key_name", apiKey.Name)
		c.Set("api_key", apiKey.ID)
		c.Next()
	}
}
