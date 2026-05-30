package service

import (
	"context"
	"fmt"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/utils"
)

type APIKeyService struct {
	repo *repository.APIKeyRepository
}

func NewAPIKeyService(repo *repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

type CreatedAPIKey struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	KeyPrefix string `json:"key_prefix"`
	Name      string `json:"name"`
	GroupID   string `json:"group_id"`
	Status    string `json:"status"`
}

func (s *APIKeyService) Create(ctx context.Context, name, groupID string, rateLimitConfig, quotaConfig map[string]any, expiresAt string) (*CreatedAPIKey, error) {
	plainKey, keyHash, keyPrefix, err := utils.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	apiKey := &domain.APIKey{
		ID:              "key-" + utils.GenerateID(),
		KeyHash:         keyHash,
		KeyPrefix:       keyPrefix,
		PlainKey:        plainKey,
		Name:            name,
		GroupID:         groupID,
		RateLimitConfig: rateLimitConfig,
		QuotaConfig:     quotaConfig,
		Status:          domain.APIKeyStatusActive,
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	return &CreatedAPIKey{
		ID:        apiKey.ID,
		Key:       plainKey,
		KeyPrefix: keyPrefix,
		Name:      name,
		GroupID:   groupID,
		Status:    string(apiKey.Status),
	}, nil
}

func (s *APIKeyService) List(ctx context.Context, status, groupID string, page, pageSize int) ([]*domain.APIKey, int64, error) {
	return s.repo.List(ctx, status, groupID, page, pageSize)
}

func (s *APIKeyService) Reveal(ctx context.Context, id string) (*CreatedAPIKey, error) {
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CreatedAPIKey{
		ID:        key.ID,
		Key:       key.PlainKey,
		KeyPrefix: key.KeyPrefix,
		Name:      key.Name,
		GroupID:   key.GroupID,
		Status:    string(key.Status),
	}, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, id string) error {
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	key.Status = domain.APIKeyStatusRevoked
	return s.repo.Update(ctx, key)
}

func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Update updates an API key's name, group_id, and rate_limit_config.
func (s *APIKeyService) Update(ctx context.Context, id string, name, groupID string, rateLimitConfig map[string]any) (*domain.APIKey, error) {
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		key.Name = name
	}
	if groupID != "" {
		key.GroupID = groupID
	}
	if rateLimitConfig != nil {
		key.RateLimitConfig = rateLimitConfig
	}

	if err := s.repo.Update(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}
