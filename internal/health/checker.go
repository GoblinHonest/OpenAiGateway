package health

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/event"
)

func buildHealthCheckRequest(ctx context.Context, provider *domain.Provider) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(provider.BaseURL, "/")+"/health", nil)
}

type EWMA struct {
	value   float64
	alpha   float64
	started bool
}

func NewEWMA(alpha float64) *EWMA {
	return &EWMA{alpha: alpha}
}

func (e *EWMA) Add(value float64) {
	if !e.started {
		e.value = value
		e.started = true
		return
	}
	e.value = e.alpha*value + (1-e.alpha)*e.value
}

func (e *EWMA) Value() float64 {
	return e.value
}

type HealthChecker struct {
	interval           time.Duration
	timeout            time.Duration
	healthyThreshold   int
	unhealthyThreshold int
	alpha              float64

	states      map[string]*ProviderHealthState
	latencyEWMA map[string]*EWMA
	errorEWMA   map[string]*EWMA
	mu          sync.RWMutex
	eventBus    *event.EventBus
	stopCh      chan struct{}
}

func NewHealthChecker(interval, timeout time.Duration, healthyThreshold, unhealthyThreshold int, alpha float64, eventBus *event.EventBus) *HealthChecker {
	if alpha == 0 {
		alpha = 0.3
	}
	return &HealthChecker{
		interval:           interval,
		timeout:            timeout,
		healthyThreshold:   healthyThreshold,
		unhealthyThreshold: unhealthyThreshold,
		alpha:              alpha,
		states:             make(map[string]*ProviderHealthState),
		latencyEWMA:        make(map[string]*EWMA),
		errorEWMA:          make(map[string]*EWMA),
		eventBus:           eventBus,
		stopCh:             make(chan struct{}),
	}
}

func (hc *HealthChecker) GetHealth(providerID string) *ProviderHealthState {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.states[providerID]
}

func (hc *HealthChecker) GetAllStates() []*ProviderHealthState {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	result := make([]*ProviderHealthState, 0, len(hc.states))
	for _, s := range hc.states {
		result = append(result, s)
	}
	return result
}

func (hc *HealthChecker) RecordLatency(providerID string, latency time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if _, ok := hc.latencyEWMA[providerID]; !ok {
		hc.latencyEWMA[providerID] = NewEWMA(hc.alpha)
	}
	hc.latencyEWMA[providerID].Add(float64(latency.Milliseconds()))

	if state, ok := hc.states[providerID]; ok {
		state.NormalizedLatency = hc.latencyEWMA[providerID].Value() / 10000.0
		if state.NormalizedLatency > 1 {
			state.NormalizedLatency = 1
		}
	}
}

func (hc *HealthChecker) RecordError(providerID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if _, ok := hc.errorEWMA[providerID]; !ok {
		hc.errorEWMA[providerID] = NewEWMA(hc.alpha)
	}
	hc.errorEWMA[providerID].Add(1.0)

	if state, ok := hc.states[providerID]; ok {
		state.ErrorRate = hc.errorEWMA[providerID].Value()
	}
}

func (hc *HealthChecker) RecordSuccess(providerID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if _, ok := hc.errorEWMA[providerID]; !ok {
		hc.errorEWMA[providerID] = NewEWMA(hc.alpha)
	}
	hc.errorEWMA[providerID].Add(0.0)

	if state, ok := hc.states[providerID]; ok {
		state.ErrorRate = hc.errorEWMA[providerID].Value()
		state.Availability = 1.0 - state.ErrorRate
	}
}

func (hc *HealthChecker) CheckProvider(ctx context.Context, provider *domain.Provider) *ProviderHealthState {
	ctx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	start := time.Now()

	var checkErr error
	if provider.BaseURL != "" {
		checkErr = hc.pingProvider(ctx, provider)
	}
	latency := time.Since(start)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	state := hc.states[provider.ID]
	if state == nil {
		state = &ProviderHealthState{ProviderID: provider.ID}
		hc.states[provider.ID] = state
	}

	state.LastCheckAt = time.Now()
	if state.AvgLatency == 0 {
		state.AvgLatency = latency
	} else {
		state.AvgLatency = (state.AvgLatency + latency) / 2
	}

	if checkErr != nil {
		state.ConsecutivePass = 0
		state.ConsecutiveFail++

		if state.ConsecutiveFail >= hc.unhealthyThreshold {
			oldStatus := state.Status
			state.Status = StatusUnhealthy

			if oldStatus != StatusUnhealthy {
				hc.eventBus.Publish(event.Event{
					Type:    event.EventProviderHealthChange,
					Payload: map[string]any{"provider_id": provider.ID, "status": StatusUnhealthy},
					Timestamp: time.Now(),
				})
			}
		} else if state.ConsecutiveFail >= int(math.Ceil(float64(hc.unhealthyThreshold)/2)) {
			state.Status = StatusDegraded
		}
	} else {
		state.ConsecutiveFail = 0
		state.ConsecutivePass++

		if state.ConsecutivePass >= hc.healthyThreshold {
			oldStatus := state.Status
			state.Status = StatusHealthy

			if oldStatus != StatusHealthy {
				hc.eventBus.Publish(event.Event{
					Type:    event.EventProviderHealthChange,
					Payload: map[string]any{"provider_id": provider.ID, "status": StatusHealthy},
					Timestamp: time.Now(),
				})
			}
		}
	}

	return state
}

func (hc *HealthChecker) pingProvider(ctx context.Context, provider *domain.Provider) error {
	req, err := buildHealthCheckRequest(ctx, provider)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: hc.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

func (hc *HealthChecker) Start(ctx context.Context, providers func() []*domain.Provider) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, p := range providers() {
				hc.CheckProvider(ctx, p)
			}
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		}
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}
