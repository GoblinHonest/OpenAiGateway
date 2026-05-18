package admin

import (
	"net/http"
	"strconv"

	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

func (h *StatsHandler) Dashboard(c *gin.Context) {
	overview, err := h.svc.GetDashboardOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, overview)
}

func (h *StatsHandler) QueryLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	q := service.LogQuery{
		StartTime:     c.Query("start_time"),
		EndTime:       c.Query("end_time"),
		APIKeyID:      c.Query("api_key_id"),
		APIKeyName:    c.Query("api_key_name"),
		ProviderName:  c.Query("provider_name"),
		ModelName:     c.Query("model_name"),
		InterfaceType: c.Query("interface_type"),
		Success:       c.Query("success"),
		CacheHit:      c.Query("cache_hit"),
		Limit:         limit,
	}

	logs, total, err := h.svc.QueryLogs(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": logs,
		"total": total,
	})
}

func (h *StatsHandler) TokenStats(c *gin.Context) {
	providerID := c.Query("provider_id")
	tokens, err := h.svc.GetTokenStats(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}
