package handler

import (
	"net/http"
	"time"

	"github.com/example/aigateway/internal/health"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db            *gorm.DB
	healthChecker *health.HealthChecker
	startTime     time.Time
	version       string
}

func NewHealthHandler(db *gorm.DB, healthChecker *health.HealthChecker, version string) *HealthHandler {
	return &HealthHandler{
		db:            db,
		healthChecker: healthChecker,
		startTime:     time.Now(),
		version:       version,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	dbStatus := "healthy"
	if sqlDB, err := h.db.DB(); err != nil {
		dbStatus = "unhealthy"
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"version":        h.version,
		"uptime_seconds": int(time.Since(h.startTime).Seconds()),
		"checks": gin.H{
			"database": dbStatus,
		},
	})
}
