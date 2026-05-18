package health

import "time"

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type ProviderHealthState struct {
	ProviderID      string
	Status          HealthStatus
	ConsecutivePass int
	ConsecutiveFail int
	LastCheckAt     time.Time
	AvgLatency      time.Duration
	ErrorRate       float64
	NormalizedLatency float64
	Availability    float64
}
