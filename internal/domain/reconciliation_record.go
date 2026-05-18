package domain

import "time"

type ReconciliationRecord struct {
	ID             string    `json:"id" gorm:"primaryKey;size:64"`
	Type           string    `json:"type" gorm:"size:64;not null;index"`
	Status         string    `json:"status" gorm:"size:32;not null;default:pending"`
	BeforeState    string    `json:"before_state" gorm:"type:json"`
	AfterState     string    `json:"after_state" gorm:"type:json"`
	Difference     string    `json:"difference" gorm:"type:json"`
	CorrectedAt    *time.Time `json:"corrected_at"`
	Notes          string    `json:"notes" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ReconciliationRecord) TableName() string {
	return "reconciliation_records"
}
