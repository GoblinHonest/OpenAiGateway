package resilience

import (
	"context"
	"fmt"

	"github.com/example/aigateway/internal/domain"
)

type FallbackChain struct {
	providers []*domain.Provider
	breakers  map[string]*CircuitBreaker
	retry     RetryConfig
}

func NewFallbackChain(providers []*domain.Provider, breakers map[string]*CircuitBreaker, retCfg RetryConfig) *FallbackChain {
	return &FallbackChain{
		providers: providers,
		breakers:  breakers,
		retry:     retCfg,
	}
}

type ProviderRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

type ProviderResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

type ProviderExecutor interface {
	Execute(ctx context.Context, provider *domain.Provider, req *ProviderRequest) (*ProviderResponse, error)
}

func (c *FallbackChain) Execute(ctx context.Context, executor ProviderExecutor, req *ProviderRequest) (*ProviderResponse, error) {
	var lastErr error

	for _, provider := range c.providers {
		breaker, ok := c.breakers[provider.ID]
		if ok && breaker.State() == StateOpen {
			continue
		}

		var resp *ProviderResponse
		err := breaker.Call(ctx, func() error {
			var providerErr error
			resp, providerErr = executor.Execute(ctx, provider, req)
			return providerErr
		})

		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
