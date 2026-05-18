package routing

import (
	"context"
	"sort"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/health"
)

type AdaptiveStrategy struct {
	healthChecker *health.HealthChecker
}

func NewAdaptiveStrategy(healthChecker *health.HealthChecker) *AdaptiveStrategy {
	return &AdaptiveStrategy{healthChecker: healthChecker}
}

type scoredProvider struct {
	provider *domain.Provider
	score    float64
}

func (s *AdaptiveStrategy) SelectProvider(ctx context.Context, providers []*domain.Provider) (*domain.Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	scored := make([]scoredProvider, 0, len(providers))

	for _, p := range providers {
		hs := s.healthChecker.GetHealth(p.ID)

		var normalizedLatency, errorRate, availability float64
		if hs != nil {
			normalizedLatency = hs.NormalizedLatency
			errorRate = hs.ErrorRate
			availability = hs.Availability
		}

		score := (1.0-normalizedLatency)*0.4 + (1.0-errorRate)*0.3 + availability*0.3
		scored = append(scored, scoredProvider{provider: p, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if len(scored) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	return scored[0].provider, nil
}
