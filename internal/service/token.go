package service

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/event"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/utils"
)

type TokenService struct {
	repo     *repository.TokenRepository
	eventBus *event.EventBus
}

func NewTokenService(repo *repository.TokenRepository, eventBus *event.EventBus) *TokenService {
	return &TokenService{repo: repo, eventBus: eventBus}
}

func (s *TokenService) Create(ctx context.Context, token *domain.Token) error {
	if token.ID == "" {
		token.ID = utils.GenerateRequestID()
	}
	return s.repo.Create(ctx, token)
}

func (s *TokenService) GetByID(ctx context.Context, id string) (*domain.Token, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TokenService) ListByProvider(ctx context.Context, providerID string, status string, page, pageSize int) ([]*domain.Token, int64, error) {
	return s.repo.ListByProvider(ctx, providerID, status, page, pageSize)
}

func (s *TokenService) Update(ctx context.Context, token *domain.Token) error {
	return s.repo.Update(ctx, token)
}

func (s *TokenService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TokenService) UseToken(ctx context.Context, tokenID string, amount int64) error {
	token, err := s.repo.GetByID(ctx, tokenID)
	if err != nil {
		return err
	}

	success, err := s.repo.DecrementQuota(ctx, tokenID, amount, token.Version)
	if err != nil {
		return err
	}

	if !success {
		if s.eventBus != nil {
			s.eventBus.Publish(event.Event{
				Type:      event.EventTokenExhausted,
				Payload:   map[string]string{"token_id": tokenID, "provider_id": token.ProviderID},
			})
		}
		return &TokenExhaustedError{TokenID: tokenID}
	}

	return nil
}

func (s *TokenService) GetAvailableByProvider(ctx context.Context, providerID string) ([]*domain.Token, error) {
	return s.repo.GetAvailableByProvider(ctx, providerID)
}

type TokenExhaustedError struct {
	TokenID string
}

func (e *TokenExhaustedError) Error() string {
	return "token exhausted: " + e.TokenID
}
