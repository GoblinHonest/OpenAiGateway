package routing

import (
	"context"
	"sort"

	"github.com/example/aigateway/internal/domain"
)

type PriorityStrategy struct{}

func NewPriorityStrategy() *PriorityStrategy {
	return &PriorityStrategy{}
}

func (s *PriorityStrategy) SelectProvider(ctx context.Context, providers []*domain.Provider) (*domain.Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority > providers[j].Priority
	})

	return providers[0], nil
}
