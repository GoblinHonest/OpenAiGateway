package middleware

import (
	"net/http"
	"strings"

	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(gatewayService *service.GatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 支持多种认证格式:
		// 1. Authorization: Bearer sk-xxx (OpenAI 标准)
		// 2. Authorization: sk-xxx (无 Bearer 前缀)
		// 3. x-api-key: sk-xxx (Anthropic 格式)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 尝试 Anthropic 格式
			token = c.GetHeader("x-api-key")
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTH_INVALID_KEY",
					"message": "Missing authorization header",
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
		c.Set("api_key_obj", apiKey)
		c.Next()
	}
}
