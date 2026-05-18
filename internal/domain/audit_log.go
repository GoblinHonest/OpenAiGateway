package domain

import "time"

type AdminAuditLog struct {
	ID              int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	AdminTokenPrefix string        `json:"admin_token_prefix" gorm:"size:16"`
	Action          string         `json:"action" gorm:"size:64;not null"`
	ResourceType    string         `json:"resource_type" gorm:"size:64;not null"`
	ResourceID      string         `json:"resource_id" gorm:"size:64;not null"`
	Changes         map[string]any `json:"changes" gorm:"serializer:json"`
	IPAddress       string         `json:"ip_address" gorm:"size:64"`
	UserAgent       string         `json:"user_agent" gorm:"type:text"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (AdminAuditLog) TableName() string {
	return "admin_audit_logs"
}
