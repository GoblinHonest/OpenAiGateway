package domain

import "time"

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

type ProviderHealthCheck struct {
	ID           int64        `json:"id" gorm:"primaryKey;autoIncrement"`
	ProviderID   string       `json:"provider_id" gorm:"size:64;not null"`
	TokenID      string       `json:"token_id" gorm:"size:64"`
	Status       HealthStatus `json:"status" gorm:"size:20;not null"`
	LatencyMs    int          `json:"latency_ms"`
	ErrorRate    float64      `json:"error_rate"`
	ErrorMessage string       `json:"error_message" gorm:"type:text"`
	CheckedAt    time.Time    `json:"checked_at" gorm:"not null"`
}

func (ProviderHealthCheck) TableName() string {
	return "provider_health_checks"
}

type CircuitBreakerState struct {
	ProviderID     string     `json:"provider_id" gorm:"primaryKey;size:64"`
	State          string     `json:"state" gorm:"size:20;not null"`
	FailureCount   int        `json:"failure_count" gorm:"default:0"`
	SuccessCount   int        `json:"success_count" gorm:"default:0"`
	LastFailureAt  *time.Time `json:"last_failure_at"`
	LastSuccessAt  *time.Time `json:"last_success_at"`
	NextRetryAt    *time.Time `json:"next_retry_at"`
	HalfOpenReqs   int        `json:"half_open_requests" gorm:"default:0"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (CircuitBreakerState) TableName() string {
	return "circuit_breaker_states"
}
