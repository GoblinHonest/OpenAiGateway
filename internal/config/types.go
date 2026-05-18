package config

import "time"

type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Auth          AuthConfig          `mapstructure:"auth"`
	RateLimit     RateLimitConfig     `mapstructure:"rate_limit"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	Retry         RetryConfig         `mapstructure:"retry"`
	HealthCheck   HealthCheckConfig   `mapstructure:"health_check"`
	Log           LogConfig           `mapstructure:"log"`
	Metrics       MetricsConfig       `mapstructure:"metrics"`
	Tracing       TracingConfig       `mapstructure:"tracing"`
	Reconciliation ReconciliationConfig `mapstructure:"reconciliation"`
	EventBus      EventBusConfig      `mapstructure:"event_bus"`
	HTTPClient    HTTPClientConfig    `mapstructure:"http_client"`
	Streaming     StreamingConfig     `mapstructure:"streaming"`
	Cache         CacheConfig         `mapstructure:"cache"`
	Protocol      ProtocolConfig      `mapstructure:"protocol"`
}

type ServerConfig struct {
	Host                  string        `mapstructure:"host"`
	Port                  int           `mapstructure:"port"`
	ReadTimeout           time.Duration `mapstructure:"read_timeout"`
	WriteTimeout          time.Duration `mapstructure:"write_timeout"`
	IdleTimeout           time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes        int           `mapstructure:"max_header_bytes"`
	GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
	CORS                  CORSConfig    `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	SQLite          SQLiteConfig  `mapstructure:"sqlite"`
}

type SQLiteConfig struct {
	JournalMode string `mapstructure:"journal_mode"`
	BusyTimeout int    `mapstructure:"busy_timeout"`
	CacheSize   int    `mapstructure:"cache_size"`
}

type RedisConfig struct {
	Addr          string        `mapstructure:"addr"`
	Password      string        `mapstructure:"password"`
	DB            int           `mapstructure:"db"`
	PoolSize      int           `mapstructure:"pool_size"`
	MinIdleConns  int           `mapstructure:"min_idle_conns"`
	DialTimeout   time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	PoolTimeout   time.Duration `mapstructure:"pool_timeout"`
}

type AuthConfig struct {
	AdminToken    string `mapstructure:"admin_token"`
	APIKeyHeader  string `mapstructure:"api_key_header"`
	APIKeyPrefix  string `mapstructure:"api_key_prefix"`
}

type RateLimitConfig struct {
	Enabled   bool  `mapstructure:"enabled"`
	DefaultRPM int  `mapstructure:"default_rpm"`
	DefaultTPM int  `mapstructure:"default_tpm"`
	BurstSize int   `mapstructure:"burst_size"`
}

type CircuitBreakerConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	FailureThreshold int           `mapstructure:"failure_threshold"`
	SuccessThreshold int           `mapstructure:"success_threshold"`
	CooldownDuration time.Duration `mapstructure:"cooldown_duration"`
}

type RetryConfig struct {
	MaxAttempts          int           `mapstructure:"max_attempts"`
	InitialBackoff       time.Duration `mapstructure:"initial_backoff"`
	MaxBackoff           time.Duration `mapstructure:"max_backoff"`
	BackoffMultiplier    float64       `mapstructure:"backoff_multiplier"`
	RetryableStatusCodes []int         `mapstructure:"retryable_status_codes"`
}

type HealthCheckConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	Interval           time.Duration `mapstructure:"interval"`
	Timeout            time.Duration `mapstructure:"timeout"`
	HealthyThreshold   int           `mapstructure:"healthy_threshold"`
	UnhealthyThreshold int           `mapstructure:"unhealthy_threshold"`
}

type LogConfig struct {
	Level  string         `mapstructure:"level"`
	Format string         `mapstructure:"format"`
	Output string         `mapstructure:"output"`
	File   LogFileConfig  `mapstructure:"file"`
}

type LogFileConfig struct {
	Path       string `mapstructure:"path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

type TracingConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Provider   string  `mapstructure:"provider"`
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

type ReconciliationConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Schedule string   `mapstructure:"schedule"`
	Types    []string `mapstructure:"types"`
}

type EventBusConfig struct {
	BufferSize int `mapstructure:"buffer_size"`
	Workers    int `mapstructure:"workers"`
}

type HTTPClientConfig struct {
	MaxIdleConns          int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost   int           `mapstructure:"max_idle_conns_per_host"`
	IdleConnTimeout       time.Duration `mapstructure:"idle_conn_timeout"`
	TLSHandshakeTimeout   time.Duration `mapstructure:"tls_handshake_timeout"`
	ExpectContinueTimeout time.Duration `mapstructure:"expect_continue_timeout"`
}

type StreamingConfig struct {
	BufferSize       int           `mapstructure:"buffer_size"`
	MaxChunkSize     int           `mapstructure:"max_chunk_size"`
	FlushInterval    time.Duration `mapstructure:"flush_interval"`
	KeepaliveInterval time.Duration `mapstructure:"keepalive_interval"`
}

type CacheConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	TTL      time.Duration `mapstructure:"ttl"`
	MaxSize  int           `mapstructure:"max_size"`
}

type ProtocolConfig struct {
	DefaultFormat string `mapstructure:"default_format"`
	AutoDetect    bool   `mapstructure:"auto_detect"`
}
