package admin

import (
	"net/http"

	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	svc *service.StatsService
}

func NewLogHandler(svc *service.StatsService) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) QueryRequests(c *gin.Context) {
	limit := 100
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

func (h *LogHandler) GetDetail(c *gin.Context) {
	detail, err := h.svc.GetLogDetail(c.Request.Context(), c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}
