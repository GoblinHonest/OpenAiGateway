package service

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/utils"
)

type ModelService struct {
	repo *repository.ModelRepository
}

func NewModelService(repo *repository.ModelRepository) *ModelService {
	return &ModelService{repo: repo}
}

func (s *ModelService) Create(ctx context.Context, model *domain.Model) error {
	if model.ID == "" {
		model.ID = utils.GenerateRequestID()
	}
	return s.repo.Create(ctx, model)
}

func (s *ModelService) GetByID(ctx context.Context, id string) (*domain.Model, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ModelService) GetByName(ctx context.Context, name string) (*domain.Model, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *ModelService) List(ctx context.Context, enabled bool, page, pageSize int) ([]*domain.Model, int64, error) {
	return s.repo.List(ctx, enabled, page, pageSize)
}

func (s *ModelService) Update(ctx context.Context, model *domain.Model) error {
	existing, err := s.repo.GetByID(ctx, model.ID)
	if err != nil {
		return err
	}
	// 只覆盖非零值字段，保留原有数据
	if model.Name != "" {
		existing.Name = model.Name
	}
	if model.DisplayName != "" {
		existing.DisplayName = model.DisplayName
	}
	if model.Description != "" {
		existing.Description = model.Description
	}
	if model.ModelType != "" {
		existing.ModelType = model.ModelType
	}
	if model.ContextWindow != 0 {
		existing.ContextWindow = model.ContextWindow
	}
	if model.MaxOutputTokens != 0 {
		existing.MaxOutputTokens = model.MaxOutputTokens
	}
	if model.InputPricePer1K != 0 {
		existing.InputPricePer1K = model.InputPricePer1K
	}
	if model.OutputPricePer1K != 0 {
		existing.OutputPricePer1K = model.OutputPricePer1K
	}
	if model.Metadata != nil {
		existing.Metadata = model.Metadata
	}
	return s.repo.Update(ctx, existing)
}

func (s *ModelService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ModelService) BindProvider(ctx context.Context, binding *domain.ModelProviderBinding) error {
	return s.repo.BindProvider(ctx, binding)
}

func (s *ModelService) GetBindings(ctx context.Context, modelID string) ([]*domain.ModelProviderBinding, error) {
	return s.repo.GetBindings(ctx, modelID)
}

// CreateWithBindings 创建模型并同时绑定服务商（原子操作）
func (s *ModelService) CreateWithBindings(ctx context.Context, model *domain.Model, bindings []domain.ModelProviderBinding) (*domain.Model, error) {
	if model.ID == "" {
		model.ID = utils.GenerateRequestID()
	}

	if err := s.repo.Create(ctx, model); err != nil {
		return nil, err
	}

	for i := range bindings {
		bindings[i].ID = utils.GenerateRequestID()
		bindings[i].ModelID = model.ID
		if err := s.repo.BindProvider(ctx, &bindings[i]); err != nil {
			return nil, err
		}
	}

	return model, nil
}

func (s *ModelService) RemoveBinding(ctx context.Context, id string) error {
	return s.repo.RemoveBinding(ctx, id)
}
