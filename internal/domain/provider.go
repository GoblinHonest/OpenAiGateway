package domain

import "time"

type ProviderStatus string

const (
	ProviderStatusActive      ProviderStatus = "active"
	ProviderStatusInactive    ProviderStatus = "inactive"
	ProviderStatusMaintenance ProviderStatus = "maintenance"
)

// FormatEndpoint 格式端点配置
type FormatEndpoint struct {
	Format string `json:"format"` // openai, anthropic, gemini
	URL    string `json:"url"`    // 该格式的完整URL
	Path   string `json:"path"`   // 路径，如 /v1/chat/completions
}

type Provider struct {
	ID               string            `json:"id" gorm:"primaryKey;size:64"`
	Name             string            `json:"name" gorm:"size:255;not null;uniqueIndex"`
	Description      string            `json:"description" gorm:"type:text"`
	ProviderType     string            `json:"provider_type" gorm:"size:64"`
	Status           ProviderStatus    `json:"status" gorm:"size:32;not null;default:active"`
	BaseURL          string            `json:"base_url" gorm:"type:text"`                      // 默认基础URL
	FormatEndpoints  []FormatEndpoint  `json:"format_endpoints" gorm:"serializer:json;default:'[]'"`        // 每个格式独立的端点
	Models           []string          `json:"models" gorm:"serializer:json;default:'[]'"`
	SupportedFormats []string          `json:"supported_formats" gorm:"serializer:json;default:'[]'"`
	Endpoints        map[string]string `json:"endpoints" gorm:"serializer:json;default:'{}'"`
	RateLimitConfig  map[string]any    `json:"rate_limit_config" gorm:"serializer:json"`
	TimeoutConfig    map[string]any    `json:"timeout_config" gorm:"serializer:json"`
	RetryConfig      map[string]any    `json:"retry_config" gorm:"serializer:json"`
	CustomHeaders    map[string]string `json:"custom_headers" gorm:"serializer:json;default:'{}'"`
	Priority         int               `json:"-" gorm:"-"`
	Metadata         map[string]any    `json:"metadata" gorm:"serializer:json"`
	Version          int               `json:"version" gorm:"not null;default:0"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func (Provider) TableName() string {
	return "providers"
}

// GetURLForFormat 获取指定格式的URL
func (p *Provider) GetURLForFormat(format string) string {
	// 先查找FormatEndpoints
	for _, ep := range p.FormatEndpoints {
		if ep.Format == format {
			if ep.URL != "" {
				return ep.URL
			}
		}
	}
	// 回退到BaseURL
	return p.BaseURL
}

// GetPathForFormat 获取指定格式的路径
func (p *Provider) GetPathForFormat(format string) string {
	for _, ep := range p.FormatEndpoints {
		if ep.Format == format {
			if ep.Path != "" {
				return ep.Path
			}
		}
	}
	// 默认路径
	switch format {
	case "openai":
		return "/v1/chat/completions"
	case "anthropic":
		return "/v1/messages"
	case "gemini":
		return "/v1/models/gemini-pro:generateContent"
	default:
		return "/v1/chat/completions"
	}
}
