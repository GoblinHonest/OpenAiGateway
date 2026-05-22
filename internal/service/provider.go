package service

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/utils"
)

type ProviderService struct {
	repo *repository.ProviderRepository
}

func NewProviderService(repo *repository.ProviderRepository) *ProviderService {
	return &ProviderService{repo: repo}
}

func (s *ProviderService) Create(ctx context.Context, provider *domain.Provider) error {
	if provider.ID == "" {
		provider.ID = utils.GenerateRequestID()
	}
	if provider.Status == "" {
		provider.Status = domain.ProviderStatusActive
	}
	return s.repo.Create(ctx, provider)
}

func (s *ProviderService) GetByID(ctx context.Context, id string) (*domain.Provider, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProviderService) List(ctx context.Context, status string, page, pageSize int) ([]*domain.Provider, int64, error) {
	return s.repo.List(ctx, status, page, pageSize)
}

func (s *ProviderService) Update(ctx context.Context, provider *domain.Provider) error {
	existing, err := s.repo.GetByID(ctx, provider.ID)
	if err != nil {
		return err
	}
	provider.Version = existing.Version
	if provider.Status == "" {
		provider.Status = existing.Status
	}
	return s.repo.Update(ctx, provider)
}

func (s *ProviderService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
