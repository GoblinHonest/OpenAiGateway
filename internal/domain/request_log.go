package domain

import "time"

type RequestLog struct {
	ID                     string            `json:"id" gorm:"primaryKey;size:64"`
	RequestID              string            `json:"request_id" gorm:"size:64;not null;uniqueIndex"`
	Timestamp              time.Time         `json:"timestamp" gorm:"precision:3"`
	ClientIP               string            `json:"client_ip" gorm:"size:64"`
	UserAgent              string            `json:"user_agent" gorm:"type:text"`
	APIKeyID               string            `json:"api_key_id" gorm:"size:64"`
	APIKeyName             string            `json:"api_key_name" gorm:"size:255"`
	GroupID                string            `json:"group_id" gorm:"size:64"`
	Route                  string            `json:"route" gorm:"size:128"`
	ModelID                string            `json:"model_id" gorm:"size:64"`
	ModelName              string            `json:"model_name" gorm:"size:255"`
	ProviderID             string            `json:"provider_id" gorm:"size:64"`
	ProviderName           string            `json:"provider_name" gorm:"size:255"`
	TokenID                string            `json:"token_id" gorm:"size:64"`
	UpstreamModelName      string            `json:"upstream_model_name" gorm:"size:255"`
	InterfaceType          string            `json:"interface_type" gorm:"size:32"`
	SourceFormat           string            `json:"source_format" gorm:"size:32"`
	TargetFormat           string            `json:"target_format" gorm:"size:32"`
	ProtocolConverted      bool              `json:"protocol_converted" gorm:"default:false"`
	IsStreaming            bool              `json:"is_streaming" gorm:"default:false"`
	RequestBodySize        int               `json:"request_body_size"`
	MaxTokens              int               `json:"max_tokens"`
	RequestHeaders         map[string]any    `json:"request_headers" gorm:"serializer:json"`
	RequestBody            string            `json:"request_body" gorm:"type:text"`
	ResponseHeaders        map[string]any    `json:"response_headers" gorm:"serializer:json"`
	ResponseBody           string            `json:"response_body" gorm:"type:text"`
	InputTokens            int               `json:"input_tokens" gorm:"default:0"`
	OutputTokens           int               `json:"output_tokens" gorm:"default:0"`
	TotalTokens            int               `json:"total_tokens" gorm:"default:0"`
	CacheHit               bool              `json:"cache_hit" gorm:"default:false"`
	CacheType              string            `json:"cache_type" gorm:"size:16"`
	CacheKey               string            `json:"cache_key" gorm:"size:64"`
	CacheReadInputTokens   int               `json:"cache_read_input_tokens" gorm:"default:0"`
	CacheSavedTokens       int               `json:"cache_saved_tokens" gorm:"default:0"`
	FirstByteLatencyMs     int               `json:"first_byte_latency_ms"`
	TotalLatencyMs         int               `json:"total_latency_ms"`
	ProviderLatencyMs      int               `json:"provider_latency_ms"`
	StatusCode             int               `json:"status_code"`
	Success                bool              `json:"success" gorm:"default:false"`
	ErrorCode              string            `json:"error_code" gorm:"size:64"`
	ErrorMessage           string            `json:"error_message" gorm:"type:text"`
	RetryCount             int               `json:"retry_count" gorm:"default:0"`
	FallbackUsed           bool              `json:"fallback_used" gorm:"default:false"`
	EstimatedCost          float64            `json:"estimated_cost"`
	CreatedAt              time.Time         `json:"created_at"`
}

func (RequestLog) TableName() string {
	return "request_logs"
}
