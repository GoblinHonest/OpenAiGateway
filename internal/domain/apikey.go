package domain

import "time"

type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusInactive APIKeyStatus = "inactive"
	APIKeyStatusRevoked  APIKeyStatus = "revoked"
)

type APIKey struct {
	ID              string         `json:"id" gorm:"primaryKey;size:64"`
	KeyHash         string         `json:"-" gorm:"size:128;not null;uniqueIndex"`
	KeyPrefix       string         `json:"key_prefix" gorm:"size:16;not null"`
	PlainKey        string         `json:"-" gorm:"size:128"`
	Name            string         `json:"name" gorm:"size:255"`
	GroupID         string         `json:"group_id" gorm:"size:64"`
	RateLimitConfig map[string]any `json:"rate_limit_config" gorm:"serializer:json"`
	QuotaConfig     map[string]any `json:"quota_config" gorm:"serializer:json"`
	Status          APIKeyStatus   `json:"status" gorm:"size:32;not null;default:active"`
	ExpiresAt       *time.Time     `json:"expires_at"`
	LastUsedAt      *time.Time     `json:"last_used_at"`
	Version         int            `json:"version" gorm:"not null;default:0"`
	Metadata        map[string]any `json:"metadata" gorm:"serializer:json"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (APIKey) TableName() string {
	return "api_keys"
}
