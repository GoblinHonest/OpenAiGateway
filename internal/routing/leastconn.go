package routing

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/example/aigateway/internal/domain"
)

type LeastConnectionsStrategy struct {
	connections map[string]*atomic.Int64
}

func NewLeastConnectionsStrategy() *LeastConnectionsStrategy {
	return &LeastConnectionsStrategy{
		connections: make(map[string]*atomic.Int64),
	}
}

func (s *LeastConnectionsStrategy) SelectProvider(ctx context.Context, providers []*domain.Provider) (*domain.Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	var selected *domain.Provider
	minConns := int64(math.MaxInt64)

	for _, p := range providers {
		conns := s.getOrCreateCounter(p.ID).Load()
		if conns < minConns {
			minConns = conns
			selected = p
		}
	}

	if selected == nil {
		return nil, ErrNoProvidersAvailable
	}

	return selected, nil
}

func (s *LeastConnectionsStrategy) Increment(providerID string) {
	s.getOrCreateCounter(providerID).Add(1)
}

func (s *LeastConnectionsStrategy) Decrement(providerID string) {
	s.getOrCreateCounter(providerID).Add(-1)
}

func (s *LeastConnectionsStrategy) getOrCreateCounter(id string) *atomic.Int64 {
	if c, ok := s.connections[id]; ok {
		return c
	}
	c := &atomic.Int64{}
	s.connections[id] = c
	return c
}
