package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("AIGATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:                    "0.0.0.0",
			Port:                    8080,
			ReadTimeout:             30 * time.Second,
			WriteTimeout:            60 * time.Second,
			IdleTimeout:             120 * time.Second,
			MaxHeaderBytes:          1048576,
			GracefulShutdownTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:          "sqlite",
			DSN:             "./data/gateway.db",
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 1 * time.Hour,
			ConnMaxIdleTime: 30 * time.Minute,
			SQLite: SQLiteConfig{
				JournalMode: "WAL",
				BusyTimeout: 5000,
				CacheSize:   -20000,
			},
		},
		Redis: RedisConfig{
			Addr:         "localhost:6379",
			DB:           0,
			PoolSize:     100,
			MinIdleConns: 10,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolTimeout:  4 * time.Second,
		},
		Auth: AuthConfig{
			APIKeyHeader: "Authorization",
			APIKeyPrefix: "Bearer ",
		},
		RateLimit: RateLimitConfig{
			Enabled:   true,
			DefaultRPM: 60,
			DefaultTPM: 100000,
			BurstSize: 10,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:           true,
			FailureThreshold:  5,
			SuccessThreshold:  2,
			CooldownDuration:  60 * time.Second,
		},
		Retry: RetryConfig{
			MaxAttempts:          3,
			InitialBackoff:       1 * time.Second,
			MaxBackoff:           10 * time.Second,
			BackoffMultiplier:    2.0,
			RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
		},
		HealthCheck: HealthCheckConfig{
			Enabled:            true,
			Interval:           5 * time.Minute,
			Timeout:            10 * time.Second,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Port:    9090,
		},
		Tracing: TracingConfig{
			Enabled:    false,
			Provider:   "jaeger",
			SampleRate: 0.1,
		},
		Reconciliation: ReconciliationConfig{
			Enabled:  true,
			Schedule: "0 2 * * *",
			Types:    []string{"token_quota", "usage_stats", "cost"},
		},
		EventBus: EventBusConfig{
			BufferSize: 1000,
			Workers:    4,
		},
		HTTPClient: HTTPClientConfig{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Streaming: StreamingConfig{
			BufferSize:        4096,
			MaxChunkSize:      1024,
			FlushInterval:     100 * time.Millisecond,
			KeepaliveInterval: 15 * time.Second,
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			MaxSize: 1000,
		},
		Protocol: ProtocolConfig{
			DefaultFormat: "openai",
			AutoDetect:    true,
		},
	}
}
