package repository

import (
	"context"
	"time"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type RequestLogRepository struct {
	db *gorm.DB
}

func NewRequestLogRepository(db *gorm.DB) *RequestLogRepository {
	return &RequestLogRepository{db: db}
}

func (r *RequestLogRepository) Create(ctx context.Context, log *domain.RequestLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *RequestLogRepository) GetByRequestID(ctx context.Context, requestID string) (*domain.RequestLog, error) {
	var log domain.RequestLog
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&log).Error
	return &log, err
}

type LogQuery struct {
	StartTime    *time.Time
	EndTime      *time.Time
	APIKeyID     string
	APIKeyName   string
	ProviderName string
	ModelName    string
	InterfaceType string
	Success      *bool
	CacheHit     *bool
	Limit        int
	Offset       int
	Cursor       string
}

func (r *RequestLogRepository) Query(ctx context.Context, q LogQuery) ([]*domain.RequestLog, int64, error) {
	var logs []*domain.RequestLog
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.RequestLog{})

	if q.StartTime != nil {
		query = query.Where("timestamp >= ?", q.StartTime)
	}
	if q.EndTime != nil {
		query = query.Where("timestamp <= ?", q.EndTime)
	}
	if q.APIKeyID != "" {
		query = query.Where("api_key_id = ?", q.APIKeyID)
	}
	if q.APIKeyName != "" {
		query = query.Where("api_key_name = ?", q.APIKeyName)
	}
	if q.ProviderName != "" {
		query = query.Where("provider_name = ?", q.ProviderName)
	}
	if q.ModelName != "" {
		query = query.Where("model_name = ?", q.ModelName)
	}
	if q.InterfaceType != "" {
		query = query.Where("interface_type = ?", q.InterfaceType)
	}
	if q.Success != nil {
		query = query.Where("success = ?", *q.Success)
	}
	if q.CacheHit != nil {
		query = query.Where("cache_hit = ?", *q.CacheHit)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}

	if err := query.Order("timestamp DESC").Offset(q.Offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

type DashboardStats struct {
	TodayRequests       int64   `json:"today_requests"`
	SevenDayRequests    int64   `json:"seven_day_requests"`
	ThirtyDayRequests   int64   `json:"thirty_day_requests"`
	TodayInputTokens    int64   `json:"today_input_tokens"`
	TodayOutputTokens   int64   `json:"today_output_tokens"`
	TodayFailedRequests int64   `json:"today_failed_requests"`
	FailureRate         float64 `json:"failure_rate"`
}

func (r *RequestLogRepository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDaysAgo := todayStart.AddDate(0, 0, -7)
	thirtyDaysAgo := todayStart.AddDate(0, 0, -30)

	var stats DashboardStats

	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ?", todayStart).
		Select("COUNT(*) as today_requests, COALESCE(SUM(input_tokens), 0) as today_input_tokens, COALESCE(SUM(output_tokens), 0) as today_output_tokens").
		Scan(&stats)

	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ? AND success = ?", todayStart, false).
		Select("COUNT(*) as today_failed_requests").
		Scan(&stats)

	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ?", sevenDaysAgo).
		Select("COUNT(*) as seven_day_requests").
		Scan(&stats)

	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ?", thirtyDaysAgo).
		Select("COUNT(*) as thirty_day_requests").
		Scan(&stats)

	if stats.TodayRequests > 0 {
		stats.FailureRate = float64(stats.TodayFailedRequests) / float64(stats.TodayRequests)
	}

	return &stats, nil
}

type TrendPoint struct {
	Time    string `json:"time"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

func (r *RequestLogRepository) GetTokenStats(ctx context.Context, providerID string) ([]map[string]any, error) {
	var results []map[string]any
	query := r.db.WithContext(ctx).Model(&domain.Token{}).Select("id, name, quota_total, quota_used, quota_remaining, success_rate")
	if providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		var quotaTotal, quotaUsed, quotaRemaining int64
		var successRate float64
		if err := rows.Scan(&id, &name, &quotaTotal, &quotaUsed, &quotaRemaining, &successRate); err != nil {
			continue
		}
		results = append(results, map[string]any{
			"id":              id,
			"name":            name,
			"quota_total":     quotaTotal,
			"quota_used":      quotaUsed,
			"quota_remaining": quotaRemaining,
			"success_rate":    successRate,
		})
	}
	return results, nil
}

func (r *RequestLogRepository) GetTrend(ctx context.Context, days int) ([]TrendPoint, error) {
	var results []TrendPoint
	start := time.Now().AddDate(0, 0, -days)

	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ?", start).
		Select("DATE(timestamp) as time, COUNT(*) as requests, COALESCE(SUM(total_tokens), 0) as tokens").
		Group("DATE(timestamp)").
		Order("time ASC").
		Scan(&results)

	return results, nil
}

type CacheStats struct {
	TotalHits    int64   `json:"total_hits"`
	HitRate      float64 `json:"hit_rate"`
	CachedTokens int64   `json:"cached_tokens"`
	SavedCost    float64 `json:"saved_cost"`
}

func (r *RequestLogRepository) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	var stats CacheStats

	// 统计缓存命中
	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("cache_hit = ?", true).
		Select("COUNT(*) as total_hits").
		Scan(&stats)

	// 统计总请求数
	var totalRequests int64
	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Select("COUNT(*)").
		Scan(&totalRequests)

	if totalRequests > 0 {
		stats.HitRate = float64(stats.TotalHits) / float64(totalRequests) * 100
	}

	// 统计缓存的token数
	r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Select("COALESCE(SUM(cache_read_input_tokens), 0)").
		Scan(&stats.CachedTokens)

	// 估算节省的成本（假设每1000 tokens $0.01）
	stats.SavedCost = float64(stats.CachedTokens) / 1000 * 0.01

	return &stats, nil
}

func (r *RequestLogRepository) GetRecentLogs(ctx context.Context, limit int) ([]*domain.RequestLog, error) {
	var logs []*domain.RequestLog
	if limit <= 0 {
		limit = 10
	}
	err := r.db.WithContext(ctx).
		Order("timestamp DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

type DistributionEntry struct {
	Name   string `json:"name"`
	Tokens int64  `json:"tokens"`
}

func (r *RequestLogRepository) GetModelDistribution(ctx context.Context, days int) ([]DistributionEntry, error) {
	var results []DistributionEntry
	start := time.Now().AddDate(0, 0, -days)

	err := r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ?", start).
		Select("model_name as name, COALESCE(SUM(input_tokens + output_tokens), 0) as tokens").
		Group("model_name").
		Order("tokens DESC").
		Scan(&results).Error

	return results, err
}

func (r *RequestLogRepository) GetProviderDistribution(ctx context.Context, days int) ([]DistributionEntry, error) {
	var results []DistributionEntry
	start := time.Now().AddDate(0, 0, -days)

	err := r.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ?", start).
		Select("provider_name as name, COALESCE(SUM(input_tokens + output_tokens), 0) as tokens").
		Group("provider_name").
		Order("tokens DESC").
		Scan(&results).Error

	return results, err
}
