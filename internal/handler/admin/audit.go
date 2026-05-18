package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditLogger struct {
	db *gorm.DB
}

func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

func (a *AuditLogger) Log(ctx context.Context, action, resourceType, resourceID string, changes map[string]any, adminTokenPrefix, ipAddress, userAgent string) {
	log := &domain.AdminAuditLog{
		AdminTokenPrefix: adminTokenPrefix,
		Action:           action,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		Changes:          changes,
		IPAddress:        ipAddress,
		UserAgent:        userAgent,
		CreatedAt:        time.Now(),
	}

	if log.AdminTokenPrefix == "" {
		log.AdminTokenPrefix = "admin"
	}
	if len(log.AdminTokenPrefix) > 16 {
		log.AdminTokenPrefix = log.AdminTokenPrefix[:16]
	}

	if err := a.db.WithContext(ctx).Create(log).Error; err != nil {
		// silent fail for audit logs
	}
}

// AuditLogQuery defines filter criteria for querying audit logs.
type AuditLogQuery struct {
	Action     string
	ResourceType string
	AdminID    string
	StartTime  *time.Time
	EndTime    *time.Time
	Page       int
	PageSize   int
}

// Query retrieves audit logs with filtering and pagination.
func (a *AuditLogger) Query(ctx context.Context, q AuditLogQuery) ([]*domain.AdminAuditLog, int64, error) {
	var logs []*domain.AdminAuditLog
	var total int64

	query := a.db.WithContext(ctx).Model(&domain.AdminAuditLog{})

	if q.Action != "" {
		query = query.Where("action = ?", q.Action)
	}
	if q.ResourceType != "" {
		query = query.Where("resource_type = ?", q.ResourceType)
	}
	if q.AdminID != "" {
		query = query.Where("admin_token_prefix = ?", q.AdminID)
	}
	if q.StartTime != nil {
		query = query.Where("created_at >= ?", q.StartTime)
	}
	if q.EndTime != nil {
		query = query.Where("created_at <= ?", q.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListAuditLogs handles GET /admin/v1/audit-logs
func (a *AuditLogger) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	q := AuditLogQuery{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		AdminID:      c.Query("admin_id"),
		Page:         page,
		PageSize:     pageSize,
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			q.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			q.EndTime = &t
		}
	}

	logs, total, err := a.Query(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func ExtractTokenPrefix(token string) string {
	if len(token) > 16 {
		return token[:16] + "..."
	}
	return token
}

func GenerateChangeMap(field string, oldVal, newVal any) map[string]any {
	return map[string]any{
		field: map[string]any{
			"old": oldVal,
			"new": newVal,
		},
	}
}

func GenerateID(prefix string) string {
	return prefix + "-" + utils.GenerateID()
}
