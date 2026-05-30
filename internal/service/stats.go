package service

import (
	"context"
	"time"

	"github.com/example/aigateway/internal/repository"
)

type StatsService struct {
	logRepo *repository.RequestLogRepository
}

func NewStatsService(logRepo *repository.RequestLogRepository) *StatsService {
	return &StatsService{logRepo: logRepo}
}

type DashboardOverview struct {
	Overview           OverviewStats     `json:"overview"`
	Trend              []TrendPoint      `json:"trend"`
	ModelDist          []DistEntry       `json:"modelDistribution"`
	ProviderDist       []DistEntry       `json:"providerDistribution"`
	RecentLogs         []RecentLogEntry  `json:"recentLogs"`
}

type RecentLogEntry struct {
	ID              string  `json:"id"`
	RequestID       string  `json:"requestId"`
	Timestamp       string  `json:"timestamp"`
	ModelName       string  `json:"modelName"`
	ProviderName    string  `json:"providerName"`
	Success         bool    `json:"success"`
	TotalLatencyMs  int     `json:"totalLatencyMs"`
	InputTokens     int     `json:"inputTokens"`
	OutputTokens    int     `json:"outputTokens"`
}

type OverviewStats struct {
	TodayRequests       int64   `json:"todayRequests"`
	SevenDayRequests    int64   `json:"sevenDayRequests"`
	ThirtyDayRequests   int64   `json:"thirtyDayRequests"`
	TodayInputTokens    int64   `json:"todayInputTokens"`
	TodayOutputTokens   int64   `json:"todayOutputTokens"`
	TodayFailedRequests int64   `json:"todayFailedRequests"`
	FailureRate         float64 `json:"failureRate"`
}

type TrendPoint struct {
	Time     string `json:"time"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

type DistEntry struct {
	Name   string `json:"name"`
	Tokens int64  `json:"tokens"`
}

func (s *StatsService) GetTokenStats(ctx context.Context, providerID string) ([]map[string]any, error) {
	return s.logRepo.GetTokenStats(ctx, providerID)
}

func (s *StatsService) GetDashboardOverview(ctx context.Context) (*DashboardOverview, error) {
	stats, err := s.logRepo.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}

	trend, err := s.logRepo.GetTrend(ctx, 7)
	if err != nil {
		return nil, err
	}

	var trendPoints []TrendPoint
	for _, t := range trend {
		trendPoints = append(trendPoints, TrendPoint{
			Time:     t.Time,
			Requests: t.Requests,
			Tokens:   t.Tokens,
		})
	}

	// 获取模型分布
	modelDistEntries, err := s.logRepo.GetModelDistribution(ctx, 30)
	if err != nil {
		modelDistEntries = nil
	}
	var modelDist []DistEntry
	for _, d := range modelDistEntries {
		modelDist = append(modelDist, DistEntry{Name: d.Name, Tokens: d.Tokens})
	}

	// 获取服务商分布
	providerDistEntries, err := s.logRepo.GetProviderDistribution(ctx, 30)
	if err != nil {
		providerDistEntries = nil
	}
	var providerDist []DistEntry
	for _, d := range providerDistEntries {
		providerDist = append(providerDist, DistEntry{Name: d.Name, Tokens: d.Tokens})
	}

	// 获取最近请求
	recentLogs, err := s.logRepo.GetRecentLogs(ctx, 10)
	if err != nil {
		recentLogs = nil
	}

	var recentLogEntries []RecentLogEntry
	for _, l := range recentLogs {
		recentLogEntries = append(recentLogEntries, RecentLogEntry{
			ID:              l.ID,
			RequestID:       l.RequestID,
			Timestamp:       l.Timestamp.Format("2006-01-02 15:04:05"),
			ModelName:       l.ModelName,
			ProviderName:    l.ProviderName,
			Success:         l.Success,
			TotalLatencyMs:  l.TotalLatencyMs,
			InputTokens:     l.InputTokens,
			OutputTokens:    l.OutputTokens,
		})
	}

	return &DashboardOverview{
		Overview: OverviewStats{
			TodayRequests:       stats.TodayRequests,
			SevenDayRequests:    stats.SevenDayRequests,
			ThirtyDayRequests:   stats.ThirtyDayRequests,
			TodayInputTokens:    stats.TodayInputTokens,
			TodayOutputTokens:   stats.TodayOutputTokens,
			TodayFailedRequests: stats.TodayFailedRequests,
			FailureRate:         stats.FailureRate,
		},
		Trend:        trendPoints,
		ModelDist:    modelDist,
		ProviderDist: providerDist,
		RecentLogs:   recentLogEntries,
	}, nil
}

type LogQuery struct {
	StartTime     string
	EndTime       string
	APIKeyID      string
	APIKeyName    string
	ProviderName  string
	ModelName     string
	InterfaceType string
	Success       string
	CacheHit      string
	Limit         int
}

func (s *StatsService) QueryLogs(ctx context.Context, q LogQuery) ([]map[string]any, int64, error) {
	query := repository.LogQuery{Limit: q.Limit}

	if q.StartTime != "" {
		t, err := time.Parse(time.RFC3339, q.StartTime)
		if err == nil {
			query.StartTime = &t
		}
	}
	if q.EndTime != "" {
		t, err := time.Parse(time.RFC3339, q.EndTime)
		if err == nil {
			query.EndTime = &t
		}
	}
	query.APIKeyID = q.APIKeyID
	query.APIKeyName = q.APIKeyName
	query.ProviderName = q.ProviderName
	query.ModelName = q.ModelName
	query.InterfaceType = q.InterfaceType

	if q.Success == "true" {
		t := true
		query.Success = &t
	} else if q.Success == "false" {
		t := false
		query.Success = &t
	}
	if q.CacheHit == "true" {
		t := true
		query.CacheHit = &t
	} else if q.CacheHit == "false" {
		t := false
		query.CacheHit = &t
	}

	logs, total, err := s.logRepo.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	var result []map[string]any
	for _, l := range logs {
		result = append(result, map[string]any{
			"id":                  l.ID,
			"requestId":          l.RequestID,
			"timestamp":          l.Timestamp,
			"clientIp":           l.ClientIP,
			"apiKeyName":         l.APIKeyName,
			"interfaceType":      l.InterfaceType,
			"route":              l.Route,
			"modelName":          l.ModelName,
			"providerName":       l.ProviderName,
			"stream":             l.IsStreaming,
			"success":            l.Success,
			"firstTokenLatencyMs": l.FirstByteLatencyMs,
			"totalLatencyMs":     l.TotalLatencyMs,
			"inputTokens":        l.InputTokens,
			"outputTokens":       l.OutputTokens,
			"cacheReadInputTokens": l.CacheReadInputTokens,
			"errorMessage":       l.ErrorMessage,
		})
	}

	return result, total, nil
}

func (s *StatsService) GetLogDetail(ctx context.Context, requestID string) (map[string]any, error) {
	logEntry, err := s.logRepo.GetByRequestID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":              logEntry.ID,
		"requestId":       logEntry.RequestID,
		"timestamp":       logEntry.Timestamp,
		"clientIp":        logEntry.ClientIP,
		"apiKeyName":      logEntry.APIKeyName,
		"interfaceType":   logEntry.InterfaceType,
		"route":           logEntry.Route,
		"modelName":       logEntry.ModelName,
		"providerName":    logEntry.ProviderName,
		"stream":          logEntry.IsStreaming,
		"maxTokens":       logEntry.MaxTokens,
		"success":         logEntry.Success,
		"totalLatencyMs":  logEntry.TotalLatencyMs,
		"inputTokens":     logEntry.InputTokens,
		"outputTokens":    logEntry.OutputTokens,
		"cacheReadInputTokens": logEntry.CacheReadInputTokens,
		"requestHeaders":  logEntry.RequestHeaders,
		"requestBody":     logEntry.RequestBody,
		"responseHeaders": logEntry.ResponseHeaders,
		"responseBody":    logEntry.ResponseBody,
		"errorMessage":    logEntry.ErrorMessage,
	}, nil
}
