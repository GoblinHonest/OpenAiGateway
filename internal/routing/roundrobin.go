package routing

import (
	"context"
	"sync/atomic"

	"github.com/example/aigateway/internal/domain"
)

type RoundRobinStrategy struct {
	current atomic.Uint64
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{}
}

func (s *RoundRobinStrategy) SelectProvider(ctx context.Context, providers []*domain.Provider) (*domain.Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}
	idx := s.current.Add(1) - 1
	return providers[idx%uint64(len(providers))], nil
}
