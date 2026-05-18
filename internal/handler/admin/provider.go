package admin

import (
	"net/http"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/service"
	"github.com/example/aigateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProviderHandler struct {
	svc         *service.ProviderService
	gateway     *service.GatewayService
	auditLogger *AuditLogger
}

func NewProviderHandler(svc *service.ProviderService, gateway *service.GatewayService, auditLogger *AuditLogger) *ProviderHandler {
	return &ProviderHandler{svc: svc, gateway: gateway, auditLogger: auditLogger}
}

func (h *ProviderHandler) Create(c *gin.Context) {
	var provider domain.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if errs := validateProvider(&provider); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errs.Error()})
		return
	}

	if err := h.svc.Create(c.Request.Context(), &provider); err != nil {
		logger.L.Error("failed to create provider",
			zap.String("name", provider.Name),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "create", "provider", provider.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, provider)
}

func (h *ProviderHandler) List(c *gin.Context) {
	status := c.Query("status")
	page := 1
	pageSize := 20

	providers, total, err := h.svc.List(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": providers,
		"total": total,
	})
}

func (h *ProviderHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if err := validateID(id, "id"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func (h *ProviderHandler) Update(c *gin.Context) {
	var provider domain.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider.ID = c.Param("id")
	if errs := validateProviderForUpdate(&provider); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errs.Error()})
		return
	}

	if err := h.svc.Update(c.Request.Context(), &provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "update", "provider", provider.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, provider)
}

func (h *ProviderHandler) FetchModels(c *gin.Context) {
	id := c.Param("id")
	if err := validateID(id, "id"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	models, err := h.gateway.FetchProviderModels(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "models": []string{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

func (h *ProviderHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := validateID(id, "id"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "delete", "provider", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}

func validateProvider(p *domain.Provider) ValidationErrors {
	var errs ValidationErrors
	if err := validateName(p.Name); err != nil {
		errs = append(errs, *err)
	}
	if err := validateDescription(p.Description); err != nil {
		errs = append(errs, *err)
	}
	if err := validateRequiredURL(p.BaseURL, "base_url"); err != nil {
		errs = append(errs, *err)
	}
	for _, ep := range p.FormatEndpoints {
		if ep.URL != "" {
			if err := validateURL(ep.URL, "format_endpoints.url"); err != nil {
				errs = append(errs, *err)
			}
		}
	}
	if err := validateJSON(p.RateLimitConfig, "rate_limit_config"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.TimeoutConfig, "timeout_config"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.RetryConfig, "retry_config"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.Metadata, "metadata"); err != nil {
		errs = append(errs, *err)
	}
	return errs
}

func validateProviderForUpdate(p *domain.Provider) ValidationErrors {
	var errs ValidationErrors
	if err := validateID(p.ID, "id"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateName(p.Name); err != nil {
		errs = append(errs, *err)
	}
	if err := validateDescription(p.Description); err != nil {
		errs = append(errs, *err)
	}
	if err := validateRequiredURL(p.BaseURL, "base_url"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.RateLimitConfig, "rate_limit_config"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.TimeoutConfig, "timeout_config"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.RetryConfig, "retry_config"); err != nil {
		errs = append(errs, *err)
	}
	if err := validateJSON(p.Metadata, "metadata"); err != nil {
		errs = append(errs, *err)
	}
	return errs
}
