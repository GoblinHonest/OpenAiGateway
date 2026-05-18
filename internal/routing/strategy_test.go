package routing

import (
	"context"
	"testing"

	"github.com/example/aigateway/internal/domain"
)

func TestRoundRobinStrategy(t *testing.T) {
	strategy := NewRoundRobinStrategy()
	providers := []*domain.Provider{
		{ID: "p1", Name: "Provider 1"},
		{ID: "p2", Name: "Provider 2"},
		{ID: "p3", Name: "Provider 3"},
	}

	selected := make(map[string]int)
	for i := 0; i < 6; i++ {
		p, err := strategy.SelectProvider(context.Background(), providers)
		if err != nil {
			t.Fatal(err)
		}
		selected[p.ID]++
	}

	if selected["p1"] != 2 || selected["p2"] != 2 || selected["p3"] != 2 {
		t.Errorf("expected equal distribution, got %v", selected)
	}
}

func TestRoundRobin_Empty(t *testing.T) {
	strategy := NewRoundRobinStrategy()
	_, err := strategy.SelectProvider(context.Background(), nil)
	if err != ErrNoProvidersAvailable {
		t.Errorf("expected ErrNoProvidersAvailable, got %v", err)
	}
}

func TestWeightedStrategy(t *testing.T) {
	weights := map[string]int{"p1": 8, "p2": 2}
	strategy := NewWeightedStrategy(weights)
	providers := []*domain.Provider{
		{ID: "p1", Name: "Provider 1"},
		{ID: "p2", Name: "Provider 2"},
	}

	selected := make(map[string]int)
	for i := 0; i < 100; i++ {
		p, err := strategy.SelectProvider(context.Background(), providers)
		if err != nil {
			t.Fatal(err)
		}
		selected[p.ID]++
	}

	t.Logf("Weighted distribution: p1=%d, p2=%d", selected["p1"], selected["p2"])

	if selected["p1"] < selected["p2"] {
		t.Error("expected p1 (weight 8) to be selected more than p2 (weight 2)")
	}
}

func TestWeightedStrategy_Empty(t *testing.T) {
	strategy := NewWeightedStrategy(nil)
	_, err := strategy.SelectProvider(context.Background(), nil)
	if err != ErrNoProvidersAvailable {
		t.Errorf("expected ErrNoProvidersAvailable, got %v", err)
	}
}

func TestPriorityStrategy(t *testing.T) {
	strategy := NewPriorityStrategy()
	providers := []*domain.Provider{
		{ID: "p1", Name: "Provider 1", Priority: 5},
		{ID: "p2", Name: "Provider 2", Priority: 10},
	}

	p, err := strategy.SelectProvider(context.Background(), providers)
	if err != nil {
		t.Fatal(err)
	}

	if p.ID != "p2" {
		t.Errorf("expected p2 (priority 10 > 5), got %s", p.ID)
	}
}

func TestStrategyFactory(t *testing.T) {
	factory := NewStrategyFactory()
	factory.Register("round_robin", NewRoundRobinStrategy())
	factory.Register("weighted", NewWeightedStrategy(nil))

	s, err := factory.Get("round_robin")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected strategy, got nil")
	}

	_, err = factory.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown strategy")
	}
}
