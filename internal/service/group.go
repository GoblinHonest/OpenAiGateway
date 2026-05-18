package service

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/utils"
)

type GroupService struct {
	repo *repository.GroupRepository
}

func NewGroupService(repo *repository.GroupRepository) *GroupService {
	return &GroupService{repo: repo}
}

func (s *GroupService) Create(ctx context.Context, group *domain.Group) error {
	if group.ID == "" {
		group.ID = utils.GenerateRequestID()
	}
	return s.repo.Create(ctx, group)
}

func (s *GroupService) GetByID(ctx context.Context, id string) (*domain.Group, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *GroupService) GetByName(ctx context.Context, name string) (*domain.Group, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *GroupService) List(ctx context.Context, enabled bool, page, pageSize int) ([]*domain.Group, int64, error) {
	return s.repo.List(ctx, enabled, page, pageSize)
}

func (s *GroupService) Update(ctx context.Context, group *domain.Group) error {
	return s.repo.Update(ctx, group)
}

func (s *GroupService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// GetModels 获取分组绑定的模型列表
func (s *GroupService) GetModels(ctx context.Context, groupID string) ([]*domain.GroupModel, error) {
	return s.repo.GetModels(ctx, groupID)
}

// SetModels 设置分组绑定的模型列表（全量替换）
func (s *GroupService) SetModels(ctx context.Context, groupID string, modelIDs []string) error {
	// 先删除所有现有绑定
	if err := s.repo.RemoveAllModels(ctx, groupID); err != nil {
		return err
	}
	// 添加新的绑定
	for _, modelID := range modelIDs {
		gm := &domain.GroupModel{
			ID:      utils.GenerateRequestID(),
			GroupID: groupID,
			ModelID: modelID,
			Enabled: true,
		}
		if err := s.repo.AddModel(ctx, gm); err != nil {
			return err
		}
	}
	return nil
}

// AddModel 添加单个模型到分组
func (s *GroupService) AddModel(ctx context.Context, groupID, modelID string) error {
	gm := &domain.GroupModel{
		ID:      utils.GenerateRequestID(),
		GroupID: groupID,
		ModelID: modelID,
		Enabled: true,
	}
	return s.repo.AddModel(ctx, gm)
}

// RemoveModel 从分组移除单个模型
func (s *GroupService) RemoveModel(ctx context.Context, groupID, modelID string) error {
	return s.repo.RemoveModel(ctx, groupID, modelID)
}
