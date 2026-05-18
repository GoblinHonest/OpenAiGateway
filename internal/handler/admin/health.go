package admin

import (
	"net/http"

	"github.com/example/aigateway/internal/health"
	"github.com/gin-gonic/gin"
)

type AdminHealthHandler struct {
	healthChecker *health.HealthChecker
}

func NewAdminHealthHandler(healthChecker *health.HealthChecker) *AdminHealthHandler {
	return &AdminHealthHandler{healthChecker: healthChecker}
}

func (h *AdminHealthHandler) GetProviderHealth(c *gin.Context) {
	states := h.healthChecker.GetAllStates()

	providers := make([]gin.H, 0, len(states))
	for _, s := range states {
		avgLatencyMs := 0
		if s.AvgLatency > 0 {
			avgLatencyMs = int(s.AvgLatency.Milliseconds())
		}
		providers = append(providers, gin.H{
			"status":               s.Status,
			"avg_latency_ms":       avgLatencyMs,
			"error_rate":           s.ErrorRate,
			"availability":         s.Availability,
			"last_check_at":        s.LastCheckAt,
			"consecutive_failures": s.ConsecutiveFail,
			"consecutive_passes":   s.ConsecutivePass,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}
