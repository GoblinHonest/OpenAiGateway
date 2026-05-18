package domain

import "time"

type LoadBalanceStrategy string

const (
	StrategyRoundRobin       LoadBalanceStrategy = "round_robin"
	StrategyWeighted         LoadBalanceStrategy = "weighted"
	StrategyLeastConnections LoadBalanceStrategy = "least_connections"
	StrategyPriority         LoadBalanceStrategy = "priority"
	StrategyAdaptive         LoadBalanceStrategy = "adaptive"
)

type Group struct {
	ID                  string              `json:"id" gorm:"primaryKey;size:64"`
	Name                string              `json:"name" gorm:"size:255;not null;uniqueIndex"`
	Description         string              `json:"description" gorm:"type:text"`
	LoadBalanceStrategy LoadBalanceStrategy `json:"load_balance_strategy" gorm:"size:64;not null;default:round_robin"`
	RateLimitConfig     map[string]any      `json:"rate_limit_config" gorm:"serializer:json"`
	QuotaConfig         map[string]any      `json:"quota_config" gorm:"serializer:json"`
	Enabled             bool                `json:"enabled" gorm:"default:true"`
	Version             int                 `json:"version" gorm:"not null;default:0"`
	Metadata            map[string]any      `json:"metadata" gorm:"serializer:json"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

func (Group) TableName() string {
	return "groups"
}

// GroupModel 分组-模型绑定关系
type GroupModel struct {
	ID        string    `json:"id" gorm:"primaryKey;size:64"`
	GroupID   string    `json:"group_id" gorm:"size:64;not null;uniqueIndex:idx_group_model"`
	ModelID   string    `json:"model_id" gorm:"size:64;not null;uniqueIndex:idx_group_model"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GroupModel) TableName() string {
	return "group_models"
}
