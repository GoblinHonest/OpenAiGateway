package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

type APIKeyHandler struct {
	svc         *service.APIKeyService
	auditLogger *AuditLogger
}

func NewAPIKeyHandler(svc *service.APIKeyService, auditLogger *AuditLogger) *APIKeyHandler {
	return &APIKeyHandler{svc: svc, auditLogger: auditLogger}
}

type CreateAPIKeyRequest struct {
	Name            string         `json:"name"`
	GroupID         string         `json:"group_id"`
	RateLimitConfig map[string]any `json:"rate_limit_config"`
	QuotaConfig     map[string]any `json:"quota_config"`
	ExpiresAt       string         `json:"expires_at"`
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.svc.Create(c.Request.Context(), req.Name, req.GroupID, req.RateLimitConfig, req.QuotaConfig, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "create", "api_key", key.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, key)
}

type UpdateAPIKeyRequest struct {
	Name            string         `json:"name"`
	GroupID         string         `json:"group_id"`
	RateLimitConfig map[string]any `json:"rate_limit_config"`
}

func (h *APIKeyHandler) Update(c *gin.Context) {
	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.svc.Update(c.Request.Context(), c.Param("id"), req.Name, req.GroupID, req.RateLimitConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "update", "api_key", c.Param("id"), nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, key)
}

func (h *APIKeyHandler) List(c *gin.Context) {
	status := c.Query("status")
	groupID := c.Query("group_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	keys, total, err := h.svc.List(c.Request.Context(), status, groupID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     keys,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Reveal 返回完整的 API Key（小眼睛查看）
func (h *APIKeyHandler) Reveal(c *gin.Context) {
	key, err := h.svc.Reveal(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, key)
}

func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Revoke(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "revoke", "api_key", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}

func (h *APIKeyHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "delete", "api_key", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}

// validateExpiry validates an expiry string (used by validateCreateAPIKeyRequest if needed).
func validateExpiry(expiresAt string) error {
	if expiresAt == "" {
		return nil
	}
	_, err := time.Parse(time.RFC3339, expiresAt)
	return err
}
