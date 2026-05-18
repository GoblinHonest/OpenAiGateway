package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests",
		},
		[]string{"provider", "model", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "model"},
	)

	TokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_tokens_total",
			Help: "Total number of tokens processed",
		},
		[]string{"provider", "model", "type"},
	)

	ActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_active_connections",
			Help: "Active connections",
		},
		[]string{"provider"},
	)

	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gateway_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	CacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gateway_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"provider"},
	)
)
