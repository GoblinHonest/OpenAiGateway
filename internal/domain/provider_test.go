package domain

import (
	"testing"
	"time"
)

func TestProviderStatus(t *testing.T) {
	if ProviderStatusActive != "active" {
		t.Errorf("expected active, got %s", ProviderStatusActive)
	}
	if ProviderStatusInactive != "inactive" {
		t.Errorf("expected inactive, got %s", ProviderStatusInactive)
	}
}

func TestProviderTableName(t *testing.T) {
	p := Provider{}
	if p.TableName() != "providers" {
		t.Errorf("expected providers, got %s", p.TableName())
	}
}

func TestTokenStatus(t *testing.T) {
	if TokenStatusActive != "active" {
		t.Errorf("expected active, got %s", TokenStatusActive)
	}
}

func TestModelProviderBinding(t *testing.T) {
	b := ModelProviderBinding{
		ID:         "test-1",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		Weight:     10,
		Priority:   1,
		Enabled:    true,
	}

	if b.TableName() != "model_provider_bindings" {
		t.Errorf("expected model_provider_bindings, got %s", b.TableName())
	}

	if b.Weight != 10 {
		t.Errorf("expected weight 10, got %d", b.Weight)
	}
}

func TestGroupDefaults(t *testing.T) {
	g := Group{
		ID:   "group-1",
		Name: "test-group",
	}

	if g.LoadBalanceStrategy != "" {
		t.Errorf("expected empty strategy, got %s", g.LoadBalanceStrategy)
	}

	g2 := Group{
		ID:                  "group-2",
		Name:                "test-group-2",
		LoadBalanceStrategy: StrategyRoundRobin,
	}

	if g2.LoadBalanceStrategy != StrategyRoundRobin {
		t.Errorf("expected round_robin, got %s", g2.LoadBalanceStrategy)
	}
}

func TestModelDefaults(t *testing.T) {
	m := Model{
		ID:      "model-1",
		Name:    "gpt-4",
		Enabled: true, // GORM default:true applies at DB level; Go zero value is false
	}

	if !m.Enabled {
		t.Errorf("expected model to be enabled when explicitly set")
	}
}

func TestAPIKeyStatus(t *testing.T) {
	if APIKeyStatusActive != "active" {
		t.Errorf("expected active, got %s", APIKeyStatusActive)
	}
	if APIKeyStatusRevoked != "revoked" {
		t.Errorf("expected revoked, got %s", APIKeyStatusRevoked)
	}
}

func TestRequestLogCreation(t *testing.T) {
	now := time.Now()
	log := RequestLog{
		ID:         "log-1",
		RequestID:  "req-abc123",
		Timestamp:  now,
		ModelName:  "gpt-4",
		Success:    true,
	}

	if !log.Success {
		t.Errorf("expected success to be true")
	}

	if log.ID != "log-1" {
		t.Errorf("expected log-1, got %s", log.ID)
	}

	if log.TableName() != "request_logs" {
		t.Errorf("expected request_logs, got %s", log.TableName())
	}
}

func TestHealthStatusValues(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestCircuitBreakerStateTable(t *testing.T) {
	s := CircuitBreakerState{}
	if s.TableName() != "circuit_breaker_states" {
		t.Errorf("expected circuit_breaker_states, got %s", s.TableName())
	}
}

func TestAdminAuditLog(t *testing.T) {
	log := AdminAuditLog{
		Action:       "create",
		ResourceType: "provider",
		ResourceID:   "provider-1",
	}

	if log.Action != "create" {
		t.Errorf("expected create, got %s", log.Action)
	}

	if log.TableName() != "admin_audit_logs" {
		t.Errorf("expected admin_audit_logs, got %s", log.TableName())
	}
}
