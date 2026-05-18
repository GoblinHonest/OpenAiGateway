package routing

import (
	"context"
	"math/rand"

	"github.com/example/aigateway/internal/domain"
)

type WeightedStrategy struct {
	weights map[string]int
}

func NewWeightedStrategy(weights map[string]int) *WeightedStrategy {
	return &WeightedStrategy{weights: weights}
}

func (s *WeightedStrategy) SelectProvider(ctx context.Context, providers []*domain.Provider) (*domain.Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	totalWeight := 0
	for _, p := range providers {
		w := s.weights[p.ID]
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	r := rand.Intn(totalWeight)
	cumulative := 0

	for _, p := range providers {
		w := s.weights[p.ID]
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if r < cumulative {
			return p, nil
		}
	}

	return providers[len(providers)-1], nil
}
