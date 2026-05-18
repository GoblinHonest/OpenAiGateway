package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/aigateway/internal/cache"
	"github.com/example/aigateway/internal/client"
	"github.com/example/aigateway/internal/config"
	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/event"
	"github.com/example/aigateway/internal/health"
	"github.com/example/aigateway/internal/protocol"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/internal/resilience"
	"github.com/example/aigateway/internal/routing"
	"github.com/example/aigateway/pkg/logger"
	"github.com/example/aigateway/pkg/metrics"
	"github.com/example/aigateway/pkg/tracing"
	"github.com/example/aigateway/pkg/utils"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type RequestContext struct {
	RequestID    string
	ClientIP     string
	APIKeyID     string
	APIKeyName   string
	Route        string
	ProviderID   string
	ProviderName string
	TokenID      string
	ModelName    string
	SourceFormat protocol.ProtocolFormat
	TargetFormat protocol.ProtocolFormat
	StartTime    time.Time
	IsStreaming  bool
	MaxTokens    int
}

type GatewayService struct {
	repo            *repository.ProviderRepository
	tokenRepo       *repository.TokenRepository
	modelRepo       *repository.ModelRepository
	groupRepo       *repository.GroupRepository
	apiKeyRepo      *repository.APIKeyRepository
	logRepo         *repository.RequestLogRepository
	cbRepo          *repository.CircuitBreakerRepository
	redisClient     *redis.Client
	cache           *cache.Cache
	eventBus        *event.EventBus
	healthChecker   *health.HealthChecker
	httpClient      *client.HTTPClient
	strategyFactory *routing.StrategyFactory
	circuitBreakers map[string]*resilience.CircuitBreaker
	retryConfig     resilience.RetryConfig
	cfg             *config.Config
}

func NewGatewayService(
	providerRepo *repository.ProviderRepository,
	tokenRepo *repository.TokenRepository,
	modelRepo *repository.ModelRepository,
	groupRepo *repository.GroupRepository,
	apiKeyRepo *repository.APIKeyRepository,
	logRepo *repository.RequestLogRepository,
	cbRepo *repository.CircuitBreakerRepository,
	redisClient *redis.Client,
	cache *cache.Cache,
	eventBus *event.EventBus,
	healthChecker *health.HealthChecker,
	httpClient *client.HTTPClient,
	strategyFactory *routing.StrategyFactory,
	cfg *config.Config,
) *GatewayService {
	circuitBreakers := make(map[string]*resilience.CircuitBreaker)

	return &GatewayService{
		repo:            providerRepo,
		tokenRepo:       tokenRepo,
		modelRepo:       modelRepo,
		groupRepo:       groupRepo,
		apiKeyRepo:      apiKeyRepo,
		logRepo:         logRepo,
		cbRepo:          cbRepo,
		redisClient:     redisClient,
		cache:           cache,
		eventBus:        eventBus,
		healthChecker:   healthChecker,
		httpClient:      httpClient,
		strategyFactory: strategyFactory,
		circuitBreakers: circuitBreakers,
		retryConfig: resilience.RetryConfig{
			MaxAttempts:          cfg.Retry.MaxAttempts,
			InitialBackoff:       cfg.Retry.InitialBackoff,
			MaxBackoff:           cfg.Retry.MaxBackoff,
			BackoffMultiplier:    cfg.Retry.BackoffMultiplier,
			RetryableStatusCodes: cfg.Retry.RetryableStatusCodes,
		},
		cfg: cfg,
	}
}

func (s *GatewayService) Authenticate(ctx context.Context, bearerToken string) (*domain.APIKey, error) {
	hash := utils.HashKey(bearerToken)
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	if apiKey.Status != domain.APIKeyStatusActive {
		return nil, fmt.Errorf("API key is %s", apiKey.Status)
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	s.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID)
	return apiKey, nil
}

func (s *GatewayService) SelectProviderAndToken(ctx context.Context, modelName string) (*domain.Provider, *domain.Token, *domain.Model, error) {
	model, err := s.modelRepo.GetByName(ctx, modelName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("model not found: %s", modelName)
	}

	bindings, err := s.modelRepo.GetBindings(ctx, model.ID)
	if err != nil || len(bindings) == 0 {
		return nil, nil, nil, fmt.Errorf("no providers configured for model: %s", modelName)
	}

	var providers []*domain.Provider
	for _, b := range bindings {
		provider, err := s.repo.GetByID(ctx, b.ProviderID)
		if err != nil || provider.Status != domain.ProviderStatusActive {
			continue
		}

		provider.Priority = b.Priority

		breaker := s.getOrCreateBreaker(provider.ID)
		if breaker.State() == resilience.StateOpen {
			continue
		}

		providers = append(providers, provider)
	}

	if len(providers) == 0 {
		return nil, nil, nil, fmt.Errorf("no available providers for model: %s", modelName)
	}

	strategyName := string(domain.StrategyRoundRobin)
	if group, err := s.groupRepo.GetByModelID(ctx, model.ID); err == nil && group.LoadBalanceStrategy != "" {
		strategyName = string(group.LoadBalanceStrategy)
	}
	strategy, err := s.strategyFactory.Get(strategyName)
	if err != nil {
		return nil, nil, nil, err
	}

	provider, err := strategy.SelectProvider(ctx, providers)
	if err != nil {
		return nil, nil, nil, err
	}

	tokens, err := s.tokenRepo.GetAvailableByProvider(ctx, provider.ID)
	if err != nil || len(tokens) == 0 {
		return nil, nil, nil, fmt.Errorf("no available tokens for provider: %s", provider.Name)
	}

	token := s.selectToken(ctx, tokens)

	return provider, token, model, nil
}

func (s *GatewayService) selectToken(ctx context.Context, tokens []*domain.Token) *domain.Token {
	if len(tokens) == 1 {
		return tokens[0]
	}

	for _, t := range tokens {
		if t.ConsecutiveFailures > 0 {
			continue
		}
		if t.QuotaRemaining > 0 {
			return t
		}
	}

	return tokens[0]
}

func (s *GatewayService) ExecuteRequest(ctx context.Context, provider *domain.Provider, token *domain.Token, req *protocol.ProviderRequest) (*protocol.ProviderResponse, error) {
	var span trace.Span
	if tracing.Tracer != nil {
		ctx, span = tracing.Tracer.Start(ctx, "gateway.ExecuteRequest")
		span.SetAttributes(
			attribute.String("provider", provider.Name),
			attribute.String("provider_id", provider.ID),
		)
		defer span.End()
	}

	body, err := json.Marshal(req.Body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, strings.TrimRight(provider.BaseURL, "/")+"/"+strings.TrimLeft(req.URL, "/"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token.TokenValue)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
		return nil, err
	}

	if span != nil {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	}

	return &protocol.ProviderResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
	}, nil
}

func (s *GatewayService) ExecuteWithFallback(ctx context.Context, provider *domain.Provider, token *domain.Token, req *protocol.ProviderRequest) (*protocol.ProviderResponse, error) {
	var span trace.Span
	if tracing.Tracer != nil {
		ctx, span = tracing.Tracer.Start(ctx, "gateway.ExecuteWithFallback")
		span.SetAttributes(
			attribute.String("provider", provider.Name),
			attribute.String("provider_id", provider.ID),
		)
		defer span.End()
	}

	// Track active connections
	metrics.ActiveConnections.WithLabelValues(provider.Name).Inc()
	defer metrics.ActiveConnections.WithLabelValues(provider.Name).Dec()

	// Check LLM response cache
	if s.cache != nil && s.cache.Enabled() {
		reqBody, _ := json.Marshal(req.Body)
		modelName := extractModelFromBody(req.Body)
		cacheKey := cache.BuildCacheKey(provider.Name, modelName, reqBody)
		if entry, _ := s.cache.Get(ctx, cacheKey); entry != nil {
			metrics.CacheHitsTotal.Inc()
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
			}
			return &protocol.ProviderResponse{
				StatusCode: entry.StatusCode,
				Body:       entry.Body,
			}, nil
		}
		metrics.CacheMissesTotal.Inc()
	}

	breaker := s.getOrCreateBreaker(provider.ID)

	var resp *protocol.ProviderResponse
	err := resilience.WithRetry(ctx, s.retryConfig, func() error {
		return breaker.Call(ctx, func() error {
			var execErr error
			resp, execErr = s.ExecuteRequest(ctx, provider, token, req)
			return execErr
		})
	})

	if err != nil {
		s.tokenRepo.RecordFailure(ctx, token.ID)
		s.healthChecker.RecordError(provider.ID)
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
		return nil, err
	}

	s.tokenRepo.RecordSuccess(ctx, token.ID)
	s.healthChecker.RecordSuccess(provider.ID)

	if span != nil {
		span.SetStatus(codes.Ok, "")
	}

	// Store in LLM response cache
	if s.cache != nil && s.cache.Enabled() && resp != nil && resp.StatusCode < 500 {
		reqBody, _ := json.Marshal(req.Body)
		modelName := extractModelFromBody(req.Body)
		cacheKey := cache.BuildCacheKey(provider.Name, modelName, reqBody)
		respBytes, _ := json.Marshal(resp.Body)
		entry := &cache.CacheEntry{
			Body:       respBytes,
			StatusCode: resp.StatusCode,
		}
		if err := s.cache.Set(ctx, cacheKey, entry); err != nil {
			logger.L.Warn("failed to cache response", zap.Error(err))
		}
	}

	return resp, nil
}

func (s *GatewayService) ExecuteStreamPassthrough(ctx context.Context, provider *domain.Provider, token *domain.Token, req *protocol.ProviderRequest, sseWriter *client.SSEHandler) ([]byte, error) {
	body, err := json.Marshal(req.Body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, strings.TrimRight(provider.BaseURL, "/")+"/"+strings.TrimLeft(req.URL, "/"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token.TokenValue)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var allData []byte
	reader := client.NewStreamReader(resp.Body)
	for chunk := range reader.Stream(ctx) {
		if chunk.Error != nil {
			return allData, chunk.Error
		}
		if chunk.Done {
			break
		}
		if len(chunk.Data) > 0 {
			allData = append(allData, chunk.Data...)
			if err := sseWriter.WriteData(chunk.Data); err != nil {
				return allData, err
			}
		}
	}

	return allData, nil
}

func (s *GatewayService) ExecuteStreamWithConversion(ctx context.Context, provider *domain.Provider, token *domain.Token, req *protocol.ProviderRequest, sseWriter *client.SSEHandler, sourceFormat, targetFormat protocol.ProtocolFormat) ([]byte, error) {
	body, err := json.Marshal(req.Body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, strings.TrimRight(provider.BaseURL, "/")+"/"+strings.TrimLeft(req.URL, "/"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token.TokenValue)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var allData []byte
	reader := client.NewStreamReader(resp.Body)
	chunks := reader.Stream(ctx)

	convertChunks := make(chan protocol.StreamChunk, 100)
	go func() {
		defer close(convertChunks)
		for ch := range chunks {
			allData = append(allData, ch.Data...)
			convertChunks <- protocol.StreamChunk{
				Data:  ch.Data,
				Event: ch.Event,
				Error: ch.Error,
				Done:  ch.Done,
			}
		}
	}()

	conv := selectStreamConverter(sourceFormat, targetFormat)
	if conv != nil {
		return allData, conv.ConvertStream(ctx, convertChunks, sseWriter)
	}

	for ch := range convertChunks {
		if ch.Error != nil {
			return allData, ch.Error
		}
		if ch.Done {
			break
		}
		if len(ch.Data) > 0 {
			sseWriter.WriteData(ch.Data)
		}
	}
	return allData, nil
}

// extractModelFromBody extracts the model name from the request body.
func extractModelFromBody(body any) string {
	if bodyMap, ok := body.(map[string]any); ok {
		if model, ok := bodyMap["model"].(string); ok {
			return model
		}
	}
	return ""
}

func selectStreamConverter(sourceFormat, targetFormat protocol.ProtocolFormat) protocol.StreamConverter {
	switch {
	case sourceFormat == protocol.FormatOpenAI && targetFormat == protocol.FormatAnthropic:
		return protocol.NewOpenAIToAnthropicStreamConverter()
	case sourceFormat == protocol.FormatOpenAI && targetFormat == protocol.FormatGemini:
		return protocol.NewOpenAIToGeminiStreamConverter()
	case sourceFormat == protocol.FormatAnthropic && targetFormat == protocol.FormatOpenAI:
		return protocol.NewAnthropicToOpenAIStreamConverter()
	case sourceFormat == protocol.FormatAnthropic && targetFormat == protocol.FormatGemini:
		return protocol.NewAnthropicToGeminiStreamConverter()
	case sourceFormat == protocol.FormatGemini && targetFormat == protocol.FormatOpenAI:
		return protocol.NewGeminiToOpenAIStreamConverter()
	case sourceFormat == protocol.FormatGemini && targetFormat == protocol.FormatAnthropic:
		return protocol.NewGeminiToAnthropicStreamConverter()
	default:
		return nil
	}
}

func (s *GatewayService) LogRequest(ctx context.Context, reqCtx *RequestContext, req *protocol.ProviderRequest, resp *protocol.ProviderResponse, inputTokens, outputTokens, cacheReadTokens int, err error) {
	duration := time.Since(reqCtx.StartTime)
	success := err == nil && resp != nil && resp.StatusCode < 500

	logEntry := &domain.RequestLog{
		ID:                   utils.GenerateRequestID(),
		RequestID:            reqCtx.RequestID,
		Timestamp:            time.Now(),
		APIKeyID:             reqCtx.APIKeyID,
		APIKeyName:           reqCtx.APIKeyName,
		ModelName:            reqCtx.ModelName,
		ProviderID:           reqCtx.ProviderID,
		ProviderName:         reqCtx.ProviderName,
		InterfaceType:        string(reqCtx.SourceFormat),
		IsStreaming:          reqCtx.IsStreaming,
		MaxTokens:            reqCtx.MaxTokens,
		InputTokens:          inputTokens,
		OutputTokens:         outputTokens,
		TotalTokens:          inputTokens + outputTokens,
		CacheReadInputTokens: cacheReadTokens,
		CacheHit:             cacheReadTokens > 0,
		TotalLatencyMs:       int(duration.Milliseconds()),
		Success:              success,
	}

	if resp != nil {
		logEntry.StatusCode = resp.StatusCode
	}

	if err != nil {
		logEntry.ErrorMessage = err.Error()
	}

	metrics.RequestsTotal.WithLabelValues(reqCtx.ProviderID, reqCtx.ModelName, fmt.Sprint(success)).Inc()
	metrics.RequestDuration.WithLabelValues(reqCtx.ProviderID, reqCtx.ModelName).Observe(duration.Seconds())

	// Increment tokens_total metrics
	if inputTokens > 0 {
		metrics.TokensTotal.WithLabelValues(reqCtx.ProviderName, reqCtx.ModelName, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		metrics.TokensTotal.WithLabelValues(reqCtx.ProviderName, reqCtx.ModelName, "output").Add(float64(outputTokens))
	}

	if err := s.logRepo.Create(ctx, logEntry); err != nil {
		logger.L.Warn("failed to save request log", zap.Error(err))
	}

	// Decrement token quota by actual usage
	if success && reqCtx.TokenID != "" && (inputTokens+outputTokens) > 0 {
		if token, tokErr := s.tokenRepo.GetByID(ctx, reqCtx.TokenID); tokErr == nil {
			if _, decErr := s.tokenRepo.DecrementQuota(ctx, reqCtx.TokenID, int64(inputTokens+outputTokens), token.Version); decErr != nil {
				logger.L.Warn("failed to decrement token quota", zap.String("token_id", reqCtx.TokenID), zap.Error(decErr))
			}
		}
	}
}

// LogRequestWithBody 记录请求日志（包含请求/响应体）
func (s *GatewayService) LogRequestWithBody(ctx context.Context, reqCtx *RequestContext, req *protocol.ProviderRequest, resp *protocol.ProviderResponse, requestBody []byte, requestHeaders map[string]any, responseBody []byte, responseHeaders map[string]any, err error) {
	duration := time.Since(reqCtx.StartTime)
	success := err == nil && resp != nil && resp.StatusCode < 500

	// 解析缓存信息
	inputTokens, outputTokens, cacheReadTokens := 0, 0, 0
	if responseBody != nil {
		inputTokens, outputTokens, cacheReadTokens = cache.ExtractUsageFromResponse(string(reqCtx.TargetFormat), responseBody)
	}

	// 截断过长的请求/响应体
	reqBodyStr := string(requestBody)
	if len(reqBodyStr) > 100000 {
		reqBodyStr = reqBodyStr[:100000] + "...(truncated)"
	}
	respBodyStr := string(responseBody)
	if len(respBodyStr) > 100000 {
		respBodyStr = respBodyStr[:100000] + "...(truncated)"
	}

	logEntry := &domain.RequestLog{
		ID:                   utils.GenerateRequestID(),
		RequestID:            reqCtx.RequestID,
		Timestamp:            time.Now(),
		ClientIP:             reqCtx.ClientIP,
		APIKeyID:             reqCtx.APIKeyID,
		APIKeyName:           reqCtx.APIKeyName,
		Route:                reqCtx.Route,
		ModelName:            reqCtx.ModelName,
		ProviderID:           reqCtx.ProviderID,
		ProviderName:         reqCtx.ProviderName,
		TokenID:              reqCtx.TokenID,
		InterfaceType:        string(reqCtx.SourceFormat),
		SourceFormat:         string(reqCtx.SourceFormat),
		TargetFormat:         string(reqCtx.TargetFormat),
		ProtocolConverted:    reqCtx.SourceFormat != reqCtx.TargetFormat,
		IsStreaming:          reqCtx.IsStreaming,
		MaxTokens:            reqCtx.MaxTokens,
		RequestBodySize:      len(requestBody),
		RequestHeaders:       requestHeaders,
		RequestBody:          reqBodyStr,
		ResponseHeaders:      responseHeaders,
		ResponseBody:         respBodyStr,
		InputTokens:          inputTokens,
		OutputTokens:         outputTokens,
		TotalTokens:          inputTokens + outputTokens,
		CacheReadInputTokens: cacheReadTokens,
		CacheHit:             cacheReadTokens > 0,
		TotalLatencyMs:       int(duration.Milliseconds()),
		Success:              success,
	}

	if resp != nil {
		logEntry.StatusCode = resp.StatusCode
	}

	if err != nil {
		logEntry.ErrorMessage = err.Error()
	}

	metrics.RequestsTotal.WithLabelValues(reqCtx.ProviderID, reqCtx.ModelName, fmt.Sprint(success)).Inc()
	metrics.RequestDuration.WithLabelValues(reqCtx.ProviderID, reqCtx.ModelName).Observe(duration.Seconds())

	// Increment tokens_total metrics
	if inputTokens > 0 {
		metrics.TokensTotal.WithLabelValues(reqCtx.ProviderName, reqCtx.ModelName, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		metrics.TokensTotal.WithLabelValues(reqCtx.ProviderName, reqCtx.ModelName, "output").Add(float64(outputTokens))
	}

	if err := s.logRepo.Create(ctx, logEntry); err != nil {
		logger.L.Warn("failed to save request log", zap.Error(err))
	}

	// Decrement token quota by actual usage
	if success && reqCtx.TokenID != "" && (inputTokens+outputTokens) > 0 {
		if token, tokErr := s.tokenRepo.GetByID(ctx, reqCtx.TokenID); tokErr == nil {
			if _, decErr := s.tokenRepo.DecrementQuota(ctx, reqCtx.TokenID, int64(inputTokens+outputTokens), token.Version); decErr != nil {
				logger.L.Warn("failed to decrement token quota", zap.String("token_id", reqCtx.TokenID), zap.Error(decErr))
			}
		}
	}
}

func (s *GatewayService) getOrCreateBreaker(providerID string) *resilience.CircuitBreaker {
	if cb, ok := s.circuitBreakers[providerID]; ok {
		return cb
	}

	cb := resilience.NewCircuitBreaker(
		providerID,
		s.cfg.CircuitBreaker.FailureThreshold,
		s.cfg.CircuitBreaker.SuccessThreshold,
		s.cfg.CircuitBreaker.CooldownDuration,
		s.cbRepo,
		s.eventBus,
	)
	s.circuitBreakers[providerID] = cb
	return cb
}

// FetchProviderModels 调上游 /v1/models 获取服务商可用模型列表
func (s *GatewayService) FetchProviderModels(ctx context.Context, providerID string) ([]string, error) {
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	tokens, _ := s.tokenRepo.GetAvailableByProvider(ctx, providerID)

	var authHeader string
	if len(tokens) > 0 {
		authHeader = "Bearer " + tokens[0].TokenValue
	}

	// 先尝试从静态配置获取
	staticModels := provider.Models

	baseURL := strings.TrimRight(provider.BaseURL, "/")
	httpReq, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		// 上游不可用时回退到静态配置
		if len(staticModels) > 0 {
			return staticModels, nil
		}
		return staticModels, nil // 返回空列表而非报错
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// OpenAI 格式: {"object":"list","data":[{"id":"gpt-4o","object":"model"},...]}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// 解析失败时回退到静态配置
		return staticModels, nil
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}

	if len(models) == 0 {
		return staticModels, nil
	}

	return models, nil
}
