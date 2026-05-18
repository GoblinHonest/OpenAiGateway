package domain

import "time"

type ModelType string

const (
	ModelTypeChat       ModelType = "chat"
	ModelTypeEmbeddings ModelType = "embeddings"
	ModelTypeImage      ModelType = "image"
	ModelTypeAudio      ModelType = "audio"
)

type Model struct {
	ID               string         `json:"id" gorm:"primaryKey;size:64"`
	Name             string         `json:"name" gorm:"size:255;not null;uniqueIndex"`
	DisplayName      string         `json:"display_name" gorm:"size:255"`
	Description      string         `json:"description" gorm:"type:text"`
	ModelType        ModelType      `json:"model_type" gorm:"size:64"`
	ContextWindow    int            `json:"context_window"`
	MaxOutputTokens  int            `json:"max_output_tokens"`
	InputPricePer1K  float64        `json:"input_price_per_1k"`
	OutputPricePer1K float64        `json:"output_price_per_1k"`
	Enabled          bool           `json:"enabled" gorm:"default:true"`
	Version          int            `json:"version" gorm:"not null;default:0"`
	Metadata         map[string]any `json:"metadata" gorm:"serializer:json"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (Model) TableName() string {
	return "models"
}
