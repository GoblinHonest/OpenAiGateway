package routing

import (
	"context"
	"errors"

	"github.com/example/aigateway/internal/domain"
)

var ErrNoProvidersAvailable = errors.New("no providers available")

type Strategy interface {
	SelectProvider(ctx context.Context, providers []*domain.Provider) (*domain.Provider, error)
}

type StrategyFactory struct {
	strategies map[string]Strategy
}

func NewStrategyFactory() *StrategyFactory {
	return &StrategyFactory{
		strategies: make(map[string]Strategy),
	}
}

func (f *StrategyFactory) Register(name string, strategy Strategy) {
	f.strategies[name] = strategy
}

func (f *StrategyFactory) Get(name string) (Strategy, error) {
	s, ok := f.strategies[name]
	if !ok {
		return nil, errors.New("unknown strategy: " + string(name))
	}
	return s, nil
}
