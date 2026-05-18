package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/event"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/logger"
	"github.com/example/aigateway/pkg/metrics"
	"go.uber.org/zap"
)

var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

type CircuitState int

const (
	StateClosed   CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

type CircuitBreaker struct {
	mu              sync.RWMutex
	providerID      string
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time

	failureThreshold int
	successThreshold int
	cooldownDuration time.Duration

	repo    *repository.CircuitBreakerRepository
	eventBus *event.EventBus
}

func NewCircuitBreaker(providerID string, failureThreshold, successThreshold int, cooldownDuration time.Duration, repo *repository.CircuitBreakerRepository, eventBus *event.EventBus) *CircuitBreaker {
	cb := &CircuitBreaker{
		providerID:       providerID,
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldownDuration: cooldownDuration,
		repo:             repo,
		eventBus:         eventBus,
	}
	cb.restoreState()
	return cb
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	state := func() CircuitState {
		cb.mu.RLock()
		defer cb.mu.RUnlock()
		return cb.state
	}()

	switch state {
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.cooldownDuration {
			cb.mu.Lock()
			if cb.state == StateOpen {
				cb.state = StateHalfOpen
				cb.persistState()
			}
			cb.mu.Unlock()
			return cb.tryRequest(ctx, fn)
		}
		return ErrCircuitBreakerOpen

	case StateHalfOpen:
		return cb.tryRequest(ctx, fn)

	default:
		return cb.tryRequest(ctx, fn)
	}
}

func (cb *CircuitBreaker) tryRequest(ctx context.Context, fn func() error) error {
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(StateOpen)
		} else if cb.state == StateHalfOpen {
			cb.transitionTo(StateOpen)
		}
	} else {
		if cb.state == StateHalfOpen {
			cb.successCount++
			if cb.successCount >= cb.successThreshold {
				cb.transitionTo(StateClosed)
			}
		} else if cb.state == StateClosed {
			cb.failureCount = 0
		}
	}

	return err
}

func (cb *CircuitBreaker) transitionTo(state CircuitState) {
	oldState := cb.state
	cb.state = state
	cb.failureCount = 0
	cb.successCount = 0
	cb.persistState()
	cb.publishStateChange(oldState, state)
	metrics.CircuitBreakerState.WithLabelValues(cb.providerID).Set(float64(state))
}

func (cb *CircuitBreaker) persistState() {
	ctx := context.Background()
	state := &domain.CircuitBreakerState{
		ProviderID:   cb.providerID,
		State:        cb.state.String(),
		FailureCount: cb.failureCount,
		SuccessCount: cb.successCount,
	}
	if !cb.lastFailureTime.IsZero() {
		state.LastFailureAt = &cb.lastFailureTime
	}

	if err := cb.repo.Save(ctx, state); err != nil {
		logger.L.Warn("failed to persist circuit breaker state", zap.Error(err))
	}
}

func (cb *CircuitBreaker) restoreState() {
	ctx := context.Background()
	state, err := cb.repo.GetByProvider(ctx, cb.providerID)
	if err != nil {
		return
	}

	switch state.State {
	case "open":
		cb.state = StateOpen
	case "half_open":
		cb.state = StateHalfOpen
	default:
		cb.state = StateClosed
	}
	cb.failureCount = state.FailureCount
	cb.successCount = state.SuccessCount
	if state.LastFailureAt != nil {
		cb.lastFailureTime = *state.LastFailureAt
	}
}

func (cb *CircuitBreaker) publishStateChange(oldState, newState CircuitState) {
	cb.eventBus.Publish(event.Event{
		Type: event.EventCircuitBreakerChange,
		Payload: map[string]any{
			"provider_id": cb.providerID,
			"old_state":   oldState.String(),
			"new_state":   newState.String(),
		},
		Timestamp: time.Now(),
	})
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
}
