package domain

import "time"

type TokenStatus string

const (
	TokenStatusActive    TokenStatus = "active"
	TokenStatusInactive  TokenStatus = "inactive"
	TokenStatusExhausted TokenStatus = "exhausted"
	TokenStatusDisabled  TokenStatus = "disabled"
)

type Token struct {
	ID                   string         `json:"id" gorm:"primaryKey;size:64"`
	ProviderID           string         `json:"provider_id" gorm:"size:64;not null"`
	Name                 string         `json:"name" gorm:"size:255"`
	TokenValue           string         `json:"-" gorm:"type:text;not null"`
	Status               TokenStatus    `json:"status" gorm:"size:32;not null;default:active"`
	QuotaTotal           int64          `json:"quota_total"`
	QuotaUsed            int64          `json:"quota_used" gorm:"default:0"`
	QuotaRemaining       int64          `json:"quota_remaining"`
	QuotaResetAt         *time.Time     `json:"quota_reset_at"`
	RateLimited          bool           `json:"rate_limited" gorm:"default:false"`
	RateLimitResetAt     *time.Time     `json:"rate_limit_reset_at"`
	FailureCount         int            `json:"failure_count" gorm:"default:0"`
	ConsecutiveFailures  int            `json:"consecutive_failures" gorm:"default:0"`
	LastFailureAt        *time.Time     `json:"last_failure_at"`
	LastSuccessAt        *time.Time     `json:"last_success_at"`
	LastUsedAt           *time.Time     `json:"last_used_at"`
	TotalRequests        int64          `json:"total_requests" gorm:"default:0"`
	SuccessRequests      int64          `json:"success_requests" gorm:"default:0"`
	SuccessRate          float64        `json:"success_rate"`
	Version              int            `json:"version" gorm:"not null;default:0"`
	Metadata             map[string]any `json:"metadata" gorm:"serializer:json"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

func (Token) TableName() string {
	return "tokens"
}
