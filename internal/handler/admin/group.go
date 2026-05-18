package admin

import (
	"net/http"
	"strconv"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	svc         *service.GroupService
	auditLogger *AuditLogger
}

func NewGroupHandler(svc *service.GroupService, auditLogger *AuditLogger) *GroupHandler {
	return &GroupHandler{svc: svc, auditLogger: auditLogger}
}

func (h *GroupHandler) Create(c *gin.Context) {
	var req struct {
		domain.Group
		ModelIDs []string `json:"model_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Create(c.Request.Context(), &req.Group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 绑定模型
	if len(req.ModelIDs) > 0 {
		if err := h.svc.SetModels(c.Request.Context(), req.Group.ID, req.ModelIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	h.auditLogger.Log(c.Request.Context(), "create", "group", req.Group.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, req.Group)
}

func (h *GroupHandler) Get(c *gin.Context) {
	group, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	// 查询绑定的模型
	type GroupWithModels struct {
		*domain.Group
		ModelIDs []string `json:"model_ids"`
	}

	models, _ := h.svc.GetModels(c.Request.Context(), group.ID)
	modelIDs := make([]string, 0, len(models))
	for _, m := range models {
		modelIDs = append(modelIDs, m.ModelID)
	}

	c.JSON(http.StatusOK, GroupWithModels{
		Group:    group,
		ModelIDs: modelIDs,
	})
}

func (h *GroupHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	groups, total, err := h.svc.List(c.Request.Context(), false, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 为每个分组查询绑定的模型
	type GroupWithModels struct {
		*domain.Group
		ModelIDs []string `json:"model_ids"`
	}

	result := make([]GroupWithModels, 0, len(groups))
	for _, g := range groups {
		models, _ := h.svc.GetModels(c.Request.Context(), g.ID)
		modelIDs := make([]string, 0, len(models))
		for _, m := range models {
			modelIDs = append(modelIDs, m.ModelID)
		}
		result = append(result, GroupWithModels{
			Group:    g,
			ModelIDs: modelIDs,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *GroupHandler) Update(c *gin.Context) {
	var req struct {
		domain.Group
		ModelIDs []string `json:"model_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Group.ID = c.Param("id")
	if err := h.svc.Update(c.Request.Context(), &req.Group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新模型绑定
	if req.ModelIDs != nil {
		if err := h.svc.SetModels(c.Request.Context(), req.Group.ID, req.ModelIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	h.auditLogger.Log(c.Request.Context(), "update", "group", req.Group.ID, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, req.Group)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "delete", "group", id, nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}

func (h *GroupHandler) GetModels(c *gin.Context) {
	models, err := h.svc.GetModels(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": models})
}

func (h *GroupHandler) SetModels(c *gin.Context) {
	var req struct {
		ModelIDs []string `json:"model_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SetModels(c.Request.Context(), c.Param("id"), req.ModelIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.auditLogger.Log(c.Request.Context(), "update", "group_models", c.Param("id"), nil,
		ExtractTokenPrefix(c.GetHeader("Authorization")), c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
