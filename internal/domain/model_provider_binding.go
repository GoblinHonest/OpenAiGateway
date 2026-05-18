package domain

import "time"

type ModelProviderBinding struct {
	ID                string    `json:"id" gorm:"primaryKey;size:64"`
	ModelID           string    `json:"model_id" gorm:"size:64;not null"`
	ProviderID        string    `json:"provider_id" gorm:"size:64;not null"`
	ProviderName      string    `json:"provider_name" gorm:"-"`
	UpstreamModelName string    `json:"upstream_model_name" gorm:"size:255"`
	Weight            int       `json:"weight" gorm:"default:1"`
	Priority          int       `json:"priority" gorm:"default:0"`
	Enabled           bool      `json:"enabled" gorm:"default:true"`
	Version           int       `json:"version" gorm:"not null;default:0"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (ModelProviderBinding) TableName() string {
	return "model_provider_bindings"
}
