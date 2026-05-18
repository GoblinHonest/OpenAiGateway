package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/example/aigateway/internal/client"
	"github.com/example/aigateway/internal/config"
	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/protocol"
	"github.com/example/aigateway/internal/service"
	"github.com/example/aigateway/pkg/utils"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	gatewayService *service.GatewayService
	cfg            *config.Config
}

func NewChatHandler(gatewayService *service.GatewayService, cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		gatewayService: gatewayService,
		cfg:            cfg,
	}
}

func (h *ChatHandler) HandleChat(c *gin.Context) {
	reqCtx := &service.RequestContext{
		RequestID:    utils.GenerateRequestID(),
		ClientIP:     c.ClientIP(),
		Route:        c.Request.URL.Path,
		StartTime:    time.Now(),
		SourceFormat: protocol.DetectFormat(c.Request),
	}

	if apiKeyID, ok := c.Get("api_key_id"); ok {
		reqCtx.APIKeyID = apiKeyID.(string)
	}
	if apiKeyName, ok := c.Get("api_key_name"); ok {
		reqCtx.APIKeyName = apiKeyName.(string)
	}

	// 读取请求体
	var bodyBytes []byte
	var bodyMap map[string]any
	if reqCtx.SourceFormat != protocol.FormatGemini {
		if err := c.ShouldBindJSON(&bodyMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
			return
		}
		bodyBytes, _ = json.Marshal(bodyMap)
		model, _ := bodyMap["model"].(string)
		stream, _ := bodyMap["stream"].(bool)
		maxTokens, _ := bodyMap["max_tokens"].(float64)
		reqCtx.ModelName = model
		reqCtx.IsStreaming = stream
		reqCtx.MaxTokens = int(maxTokens)
	} else {
		var geminiReq protocol.GeminiRequest
		if err := c.ShouldBindJSON(&geminiReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
			return
		}
		bodyBytes, _ = json.Marshal(geminiReq)
		reqCtx.ModelName = "gemini-pro"
	}

	if reqCtx.ModelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "model is required"}})
		return
	}

	provider, token, _, err := h.gatewayService.SelectProviderAndToken(c.Request.Context(), reqCtx.ModelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "MODEL_NOT_FOUND", "message": err.Error()}})
		return
	}

	reqCtx.ProviderID = provider.ID
	reqCtx.ProviderName = provider.Name
	reqCtx.TokenID = token.ID
	reqCtx.TargetFormat = protocol.ProtocolFormat(provider.ProviderType)

	// 构建请求头（脱敏）
	requestHeaders := make(map[string]any)
	for k, v := range c.Request.Header {
		if k == "Authorization" {
			requestHeaders[k] = []string{"Bearer sk-****"}
		} else {
			requestHeaders[k] = v
		}
	}

	stubReq, _ := http.NewRequest("POST", c.Request.URL.Path, io.NopCloser(bytes.NewReader(bodyBytes)))
	for k, v := range c.Request.Header {
		stubReq.Header[k] = v
	}

	converter := getConverter(reqCtx.SourceFormat)
	providerReq, err := converter.ConvertRequest(c.Request.Context(), stubReq, reqCtx.TargetFormat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "CONVERSION_ERROR", "message": err.Error()}})
		return
	}

	if token != nil {
		providerReq.Headers["Authorization"] = "Bearer " + token.TokenValue
	}

	if reqCtx.IsStreaming {
		h.handleStream(c, reqCtx, provider, token, providerReq, bodyBytes, requestHeaders)
		return
	}

	resp, err := h.gatewayService.ExecuteWithFallback(c.Request.Context(), provider, token, providerReq)
	if err != nil {
		// 记录日志（包含请求体）
		h.gatewayService.LogRequestWithBody(c.Request.Context(), reqCtx, providerReq, nil, bodyBytes, requestHeaders, nil, nil, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "PROVIDER_UNAVAILABLE", "message": err.Error(), "retryable": true}})
		return
	}

	httpResp, err := converter.ConvertResponse(c.Request.Context(), resp, reqCtx.TargetFormat)
	if err != nil {
		h.gatewayService.LogRequestWithBody(c.Request.Context(), reqCtx, providerReq, resp, bodyBytes, requestHeaders, nil, nil, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CONVERSION_ERROR", "message": err.Error()}})
		return
	}

	// 读取响应体（token解析由 LogRequestWithBody 内部完成）
	respBody, _ := io.ReadAll(httpResp.Body)

	// 构建响应头
	responseHeaders := make(map[string]any)
	for k, v := range httpResp.Header {
		responseHeaders[k] = v
	}

	// 记录日志（包含请求/响应体）
	h.gatewayService.LogRequestWithBody(c.Request.Context(), reqCtx, providerReq, resp, bodyBytes, requestHeaders, respBody, responseHeaders, nil)

	c.Header("X-Request-ID", reqCtx.RequestID)
	c.Header("X-Provider", provider.Name)
	c.Header("X-Token-ID", token.ID)

	for k, v := range httpResp.Header {
		c.Header(k, v[0])
	}

	c.Data(httpResp.StatusCode, httpResp.Header.Get("Content-Type"), respBody)
}

func (h *ChatHandler) handleStream(c *gin.Context, reqCtx *service.RequestContext, provider *domain.Provider, token *domain.Token, providerReq *protocol.ProviderRequest, requestBody []byte, requestHeaders map[string]any) {
	sse, err := client.NewSSEHandler(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SSE_ERROR", "message": err.Error()}})
		return
	}
	defer sse.Close()

	sse.MonitorClient(c.Request.Context())

	c.Header("X-Request-ID", reqCtx.RequestID)
	c.Header("X-Provider", provider.Name)
	c.Header("X-Token-ID", token.ID)

	var respBody []byte
	if reqCtx.SourceFormat == reqCtx.TargetFormat {
		respBody, err = h.gatewayService.ExecuteStreamPassthrough(c.Request.Context(), provider, token, providerReq, sse)
	} else {
		respBody, err = h.gatewayService.ExecuteStreamWithConversion(c.Request.Context(), provider, token, providerReq, sse, reqCtx.SourceFormat, reqCtx.TargetFormat)
	}

	// 记录日志（token解析由 LogRequestWithBody 内部完成）
	h.gatewayService.LogRequestWithBody(c.Request.Context(), reqCtx, providerReq, nil, requestBody, requestHeaders, respBody, nil, err)
}

func getConverter(format protocol.ProtocolFormat) protocol.Converter {
	switch format {
	case protocol.FormatAnthropic:
		return protocol.NewAnthropicConverter()
	case protocol.FormatGemini:
		return protocol.NewGeminiConverter()
	default:
		return protocol.NewOpenAIConverter()
	}
}
