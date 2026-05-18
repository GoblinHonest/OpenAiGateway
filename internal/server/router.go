package server

import (
	"github.com/example/aigateway/internal/cache"
	"github.com/example/aigateway/internal/config"
	"github.com/example/aigateway/internal/handler"
	"github.com/example/aigateway/internal/handler/admin"
	"github.com/example/aigateway/internal/health"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/internal/server/middleware"
	"github.com/example/aigateway/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

type RouterDeps struct {
	DB              *gorm.DB
	GatewayService  *service.GatewayService
	ProviderSvc     *service.ProviderService
	TokenSvc        *service.TokenService
	ModelSvc        *service.ModelService
	GroupSvc        *service.GroupService
	APIKeySvc       *service.APIKeyService
	StatsSvc        *service.StatsService
	LogRepo         *repository.RequestLogRepository
	HealthChecker   *health.HealthChecker
	AuditLogger     *admin.AuditLogger
	RateLimiter     *middleware.RateLimiter
	Cache           *cache.Cache
	Cfg             *config.Config
	Server          *Server
}

func SetupRouter(deps *RouterDeps) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.CORSMiddleware(deps.Cfg))

	// Serve static files from UI dist directory
	r.Static("/assets", "./ui/dist/assets")
	r.StaticFile("/favicon.ico", "./ui/dist/favicon.ico")

	healthHandler := handler.NewHealthHandler(deps.DB, deps.HealthChecker, "1.0.0")

	r.GET("/health", healthHandler.Health)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Serve index.html for all non-API routes (SPA support)
	r.NoRoute(func(c *gin.Context) {
		// Don't serve index.html for API routes
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:5] == "/api/" {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.File("./ui/dist/index.html")
	})

	chatHandler := handler.NewChatHandler(deps.GatewayService, deps.Cfg)
	client := r.Group("/v1")
	client.Use(middleware.AuthMiddleware(deps.GatewayService))
	client.Use(deps.RateLimiter.RateLimitMiddleware(deps.Cfg.RateLimit.DefaultRPM))
	{
		client.POST("/chat/completions", chatHandler.HandleChat)
		client.POST("/completions", chatHandler.HandleChat)
		client.POST("/embeddings", chatHandler.HandleChat)
		client.POST("/messages", chatHandler.HandleChat)
		client.POST("/models/*modelpath", chatHandler.HandleChat)
	}

	r.Use(middleware.GracefulShutdownMiddleware(deps.Server))

	adminRouter := r.Group("/admin/v1")
	adminRouter.Use(middleware.AdminAuthMiddleware(deps.Cfg))
	{
		providerHandler := admin.NewProviderHandler(deps.ProviderSvc, deps.GatewayService, deps.AuditLogger)
		adminRouter.POST("/providers", providerHandler.Create)
		adminRouter.GET("/providers", providerHandler.List)
		adminRouter.GET("/providers/:id", providerHandler.Get)
		adminRouter.GET("/providers/:id/models", providerHandler.FetchModels)
		adminRouter.PUT("/providers/:id", providerHandler.Update)
		adminRouter.DELETE("/providers/:id", providerHandler.Delete)

		tokenHandler := admin.NewTokenHandler(deps.TokenSvc, deps.AuditLogger)
		adminRouter.POST("/tokens", tokenHandler.Create)
		adminRouter.GET("/tokens", tokenHandler.List)
		adminRouter.PUT("/tokens/:id", tokenHandler.Update)
		adminRouter.DELETE("/tokens/:id", tokenHandler.Delete)
		adminRouter.GET("/tokens/:id/reveal", tokenHandler.Reveal)

		modelHandler := admin.NewModelHandler(deps.ModelSvc, deps.AuditLogger)
		adminRouter.POST("/models", modelHandler.Create)
		adminRouter.POST("/models/with-bindings", modelHandler.CreateWithBindings)
		adminRouter.GET("/models", modelHandler.List)
		adminRouter.GET("/models/:id/bindings", modelHandler.GetBindings)
		adminRouter.POST("/model-provider-bindings", modelHandler.BindProvider)
		adminRouter.PUT("/models/:id", modelHandler.Update)
		adminRouter.DELETE("/models/:id", modelHandler.Delete)

		adminRouter.DELETE("/model-provider-bindings/:id", modelHandler.RemoveBinding)

		groupHandler := admin.NewGroupHandler(deps.GroupSvc, deps.AuditLogger)
		adminRouter.POST("/groups", groupHandler.Create)
		adminRouter.GET("/groups", groupHandler.List)
		adminRouter.GET("/groups/:id", groupHandler.Get)
		adminRouter.PUT("/groups/:id", groupHandler.Update)
		adminRouter.DELETE("/groups/:id", groupHandler.Delete)
		adminRouter.GET("/groups/:id/models", groupHandler.GetModels)
		adminRouter.PUT("/groups/:id/models", groupHandler.SetModels)

		apiKeyHandler := admin.NewAPIKeyHandler(deps.APIKeySvc, deps.AuditLogger)
		adminRouter.POST("/api-keys", apiKeyHandler.Create)
		adminRouter.GET("/api-keys", apiKeyHandler.List)
		adminRouter.GET("/api-keys/:id/reveal", apiKeyHandler.Reveal)
		adminRouter.DELETE("/api-keys/:id", apiKeyHandler.Revoke)

		statsHandler := admin.NewStatsHandler(deps.StatsSvc)
		adminRouter.GET("/dashboard/overview", statsHandler.Dashboard)
		adminRouter.GET("/stats/requests", statsHandler.QueryLogs)
		adminRouter.GET("/stats/tokens", statsHandler.TokenStats)

		logHandler := admin.NewLogHandler(deps.StatsSvc)
		adminRouter.GET("/logs/requests", logHandler.QueryRequests)
		adminRouter.GET("/logs/requests/:request_id", logHandler.GetDetail)

		healthAdminHandler := admin.NewAdminHealthHandler(deps.HealthChecker)
		adminRouter.GET("/health/providers", healthAdminHandler.GetProviderHealth)

		cacheHandler := admin.NewCacheHandler(deps.Cache)
		adminRouter.GET("/cache/config", cacheHandler.GetConfig)
		adminRouter.PUT("/cache/config", cacheHandler.UpdateConfig)
		adminRouter.GET("/cache/stats", cacheHandler.GetStats)
		adminRouter.DELETE("/cache", cacheHandler.Clear)
		adminRouter.GET("/cache/entries", cacheHandler.ListEntries)

		adminRouter.GET("/audit-logs", deps.AuditLogger.ListAuditLogs)
	}

	return r
}
