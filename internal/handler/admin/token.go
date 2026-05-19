package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

type TokenHandler struct {
	svc         *service.TokenService
	auditLogger *AuditLogger
}

func NewTokenHandler(svc *service.TokenService, auditLogger *AuditLogger) *TokenHandler {
	return &TokenHandler{svc: svc, auditLogger: auditLogger}
}

// createTokenRequest 用于解析创建 Token 的请求
type createTokenRequest struct {
	ProviderID     string         `json:"provider_id"`
	Name           string         `json:"name"`
	TokenValue     string         `json:"token_value"`
	QuotaTotal     int64          `json:"quota_total"`
	QuotaRemaining int64          `json:"quota_remaining"`
	ExpiresAt      string         `json:"expires_at"`
	Metadata       map[string]any `json:"metadata"`
}

func (h *TokenHandler) Create(c *gin.Context) {
	var req createTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ProviderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider_id is required"})
		return
	}
	if req.TokenValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token_value is required"})
		return
	}

	token := &domain.Token{
		ProviderID:     req.ProviderID,
		Name:           req.Name,
		TokenValue:     req.TokenValue,
		QuotaTotal:     req.QuotaTotal,
		QuotaRemaining: req.QuotaRemaining,
		Metadata:       req.Metadata,
	}

	// 如果设置了配额且未指定剩余配额，初始化为配额总量
	if token.QuotaTotal > 0 && token.QuotaRemaining == 0 {
		token.QuotaRemaining = token.QuotaTotal
	}

	token.Status = domain.TokenStatusActive
	if err := h.svc.Create(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "create", "token", token.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, gin.H{
		"id":              token.ID,
		"provider_id":     token.ProviderID,
		"name":            token.Name,
		"status":          token.Status,
		"quota_total":     token.QuotaTotal,
		"quota_remaining": token.QuotaRemaining,
		"created_at":      token.CreatedAt,
	})
}

func (h *TokenHandler) List(c *gin.Context) {
	providerID := c.Query("provider_id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tokens, total, err := h.svc.ListByProvider(c.Request.Context(), providerID, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     tokens,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *TokenHandler) Update(c *gin.Context) {
	var token domain.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token.ID = c.Param("id")
	if errs := validateTokenForUpdate(&token); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errs.Error()})
		return
	}

	if err := h.svc.Update(c.Request.Context(), &token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "update", "token", token.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, token)
}

// Reveal 返回完整的 Token 值（小眼睛查看）
func (h *TokenHandler) Reveal(c *gin.Context) {
	id := c.Param("id")
	if err := validateID(id, "id"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          token.ID,
		"name":        token.Name,
		"provider_id": token.ProviderID,
		"token_value": token.TokenValue,
	})
}

func (h *TokenHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := validateID(id, "id"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "delete", "token", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}

// validateToken validates a token for creation.
func validateToken(t *domain.Token) ValidationErrors {
	var errs ValidationErrors

	if err := validateOptionalName(t.Name); err != nil {
		errs = append(errs, *err)
	}
	if err := validateID(t.ProviderID, "provider_id"); err != nil {
		errs = append(errs, *err)
	}
	if t.TokenValue != "" {
		if strings.TrimSpace(t.TokenValue) == "" {
			errs = append(errs, ValidationError{Field: "token_value", Message: "must not be blank"})
		}
		if len(t.TokenValue) > maxTokenValueLen {
			errs = append(errs, ValidationError{Field: "token_value", Message: "is too long"})
		}
	}
	if err := validateJSON(t.Metadata, "metadata"); err != nil {
		errs = append(errs, *err)
	}

	return errs
}

// validateTokenForUpdate validates a token for update.
func validateTokenForUpdate(t *domain.Token) ValidationErrors {
	var errs ValidationErrors

	if err := validateID(t.ID, "id"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateOptionalName(t.Name); err != nil {
		errs = append(errs, *err)
	}
	if t.ProviderID != "" {
		if err := validateID(t.ProviderID, "provider_id"); err != nil {
			errs = append(errs, *err)
		}
	}
	if err := validateJSON(t.Metadata, "metadata"); err != nil {
		errs = append(errs, *err)
	}

	return errs
}
