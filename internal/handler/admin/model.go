package admin

import (
	"net/http"
	"strconv"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelHandler struct {
	svc         *service.ModelService
	auditLogger *AuditLogger
}

func NewModelHandler(svc *service.ModelService, auditLogger *AuditLogger) *ModelHandler {
	return &ModelHandler{svc: svc, auditLogger: auditLogger}
}

func (h *ModelHandler) Create(c *gin.Context) {
	var model domain.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Create(c.Request.Context(), &model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "create", "model", model.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, model)
}

func (h *ModelHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	models, total, err := h.svc.List(c.Request.Context(), false, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 为每个模型查询绑定的服务商
	type BindingInfo struct {
		ID                string `json:"id"`
		ProviderID        string `json:"provider_id"`
		ProviderName      string `json:"provider_name"`
		UpstreamModelName string `json:"upstream_model_name"`
		Weight            int    `json:"weight"`
		Priority          int    `json:"priority"`
		Enabled           bool   `json:"enabled"`
	}
	type ModelWithBindings struct {
		*domain.Model
		Bindings []BindingInfo `json:"bindings"`
	}

	result := make([]ModelWithBindings, 0, len(models))
	for _, m := range models {
		bindings, _ := h.svc.GetBindings(c.Request.Context(), m.ID)
		bInfos := make([]BindingInfo, 0, len(bindings))
		for _, b := range bindings {
			bInfos = append(bInfos, BindingInfo{
				ID:                b.ID,
				ProviderID:        b.ProviderID,
				ProviderName:      b.ProviderName,
				UpstreamModelName: b.UpstreamModelName,
				Weight:            b.Weight,
				Priority:          b.Priority,
				Enabled:           b.Enabled,
			})
		}
		result = append(result, ModelWithBindings{
			Model:    m,
			Bindings: bInfos,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *ModelHandler) BindProvider(c *gin.Context) {
	var binding domain.ModelProviderBinding
	if err := c.ShouldBindJSON(&binding); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.BindProvider(c.Request.Context(), &binding); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "create", "model_provider_binding", binding.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, binding)
}

func (h *ModelHandler) Update(c *gin.Context) {
	var model domain.Model
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model.ID = c.Param("id")
	if err := h.svc.Update(c.Request.Context(), &model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "update", "model", model.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, model)
}

func (h *ModelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "delete", "model", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}

// GetBindings 获取模型的服务商绑定列表
func (h *ModelHandler) GetBindings(c *gin.Context) {
	bindings, err := h.svc.GetBindings(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": bindings})
}

// CreateWithBindings 创建模型并同时绑定服务商（一体化接口）
type CreateModelWithBindingsRequest struct {
	Model    domain.Model                  `json:"model"`
	Bindings []domain.ModelProviderBinding `json:"bindings"`
}

func (h *ModelHandler) CreateWithBindings(c *gin.Context) {
	var req CreateModelWithBindingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.CreateWithBindings(c.Request.Context(), &req.Model, req.Bindings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "create", "model", req.Model.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, result)
}

func (h *ModelHandler) RemoveBinding(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.RemoveBinding(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "delete", "model_provider_binding", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}
