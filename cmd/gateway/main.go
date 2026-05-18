package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/aigateway/internal/cache"
	"github.com/example/aigateway/internal/client"
	"github.com/example/aigateway/internal/config"
	"github.com/example/aigateway/internal/event"
	"github.com/example/aigateway/internal/handler/admin"
	"github.com/example/aigateway/internal/health"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/internal/routing"
	"github.com/example/aigateway/internal/server"
	"github.com/example/aigateway/internal/server/middleware"
	"github.com/example/aigateway/internal/service"
	"github.com/example/aigateway/pkg/logger"
	"github.com/example/aigateway/pkg/tracing"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "config/config.yaml", "Configuration file path")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if cfg.Tracing.Enabled {
		if err := tracing.Init("aigateway", cfg.Tracing.Endpoint, cfg.Tracing.SampleRate); err != nil {
			logger.L.Warn("failed to initialize tracing", zap.Error(err))
		}
	}

	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		logger.L.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	if err := repository.AutoMigrate(db); err != nil {
		logger.L.Fatal("Failed to run migrations", zap.Error(err))
		os.Exit(1)
	}

	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:         cfg.Redis.Addr,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
			PoolTimeout:  cfg.Redis.PoolTimeout,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.L.Warn("Redis not available, running in degraded mode", zap.Error(err))
			redisClient = nil
		}
	}

	providerRepo := repository.NewProviderRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	modelRepo := repository.NewModelRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	logRepo := repository.NewRequestLogRepository(db)
	cbRepo := repository.NewCircuitBreakerRepository(db)

	eventBus := event.NewEventBus(cfg.EventBus.BufferSize)

	healthChecker := health.NewHealthChecker(
		cfg.HealthCheck.Interval,
		cfg.HealthCheck.Timeout,
		cfg.HealthCheck.HealthyThreshold,
		cfg.HealthCheck.UnhealthyThreshold,
		0.3,
		eventBus,
	)

	httpClient := client.NewHTTPClient(
		cfg.HTTPClient.MaxIdleConns,
		cfg.HTTPClient.MaxIdleConnsPerHost,
		cfg.HTTPClient.IdleConnTimeout,
		cfg.HTTPClient.TLSHandshakeTimeout,
		cfg.HTTPClient.ExpectContinueTimeout,
	)

	// LLM response cache
	llmCache := cache.NewCache(redisClient, cfg.Cache.TTL, cfg.Cache.Enabled)

	strategyFactory := routing.NewStrategyFactory()
	strategyFactory.Register("round_robin", routing.NewRoundRobinStrategy())
	strategyFactory.Register("weighted", routing.NewWeightedStrategy(nil))
	strategyFactory.Register("least_connections", routing.NewLeastConnectionsStrategy())
	strategyFactory.Register("priority", routing.NewPriorityStrategy())
	strategyFactory.Register("adaptive", routing.NewAdaptiveStrategy(healthChecker))

	gatewayService := service.NewGatewayService(
		providerRepo, tokenRepo, modelRepo, groupRepo, apiKeyRepo, logRepo, cbRepo,
		redisClient, llmCache, eventBus, healthChecker, httpClient, strategyFactory, cfg,
	)

	providerSvc := service.NewProviderService(providerRepo)
	tokenSvc := service.NewTokenService(tokenRepo, eventBus)
	modelSvc := service.NewModelService(modelRepo)
	groupSvc := service.NewGroupService(groupRepo)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	statsSvc := service.NewStatsService(logRepo)
	reconSvc := service.NewReconciliationService(db)

	auditLogger := admin.NewAuditLogger(db)
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimit.Enabled)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := server.New(
		addr, nil,
		cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout,
		cfg.Server.MaxHeaderBytes, cfg.Server.GracefulShutdownTimeout,
	)

	deps := &server.RouterDeps{
		DB:             db,
		GatewayService: gatewayService,
		ProviderSvc:    providerSvc,
		TokenSvc:       tokenSvc,
		ModelSvc:       modelSvc,
		GroupSvc:       groupSvc,
		APIKeySvc:      apiKeySvc,
		StatsSvc:       statsSvc,
		LogRepo:        logRepo,
		HealthChecker:  healthChecker,
		AuditLogger:    auditLogger,
		RateLimiter:    rateLimiter,
		Cache:          llmCache,
		Cfg:            cfg,
		Server:         srv,
	}

	router := server.SetupRouter(deps)
	srv.SetHandler(router)

	go func() {
		logger.L.Info("starting daily reconciliation scheduler")
		reconTicker := time.NewTicker(24 * time.Hour)
		defer reconTicker.Stop()

		go func() {
			time.Sleep(1 * time.Hour)
			reconSvc.RunAll(context.Background())
		}()

		for range reconTicker.C {
			reconSvc.RunAll(context.Background())
		}
	}()

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("Server failed", zap.Error(err))
		}
	}()

	logger.L.Info("AI Gateway started", zap.String("addr", addr), zap.String("version", version))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.L.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.L.Info("Server exited")
}
