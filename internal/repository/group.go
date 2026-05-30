package repository

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type GroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Create(ctx context.Context, group *domain.Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *GroupRepository) GetByID(ctx context.Context, id string) (*domain.Group, error) {
	var group domain.Group
	err := r.db.WithContext(ctx).First(&group, "id = ?", id).Error
	return &group, err
}

func (r *GroupRepository) GetByName(ctx context.Context, name string) (*domain.Group, error) {
	var group domain.Group
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&group).Error
	return &group, err
}

func (r *GroupRepository) List(ctx context.Context, enabled bool, page, pageSize int) ([]*domain.Group, int64, error) {
	var groups []*domain.Group
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Group{})
	if enabled {
		query = query.Where("enabled = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := query.Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

func (r *GroupRepository) Update(ctx context.Context, group *domain.Group) error {
	result := r.db.WithContext(ctx).Model(&domain.Group{}).
		Where("id = ? AND version = ?", group.ID, group.Version).
		Updates(map[string]any{
			"name":                  group.Name,
			"description":           group.Description,
			"load_balance_strategy": group.LoadBalanceStrategy,
			"rate_limit_config":     toJSON(group.RateLimitConfig),
			"quota_config":          toJSON(group.QuotaConfig),
			"enabled":               group.Enabled,
			"metadata":              toJSON(group.Metadata),
			"version":               gorm.Expr("version + 1"),
		})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GroupRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Group{}, "id = ?", id).Error
}

// GetModels 获取分组绑定的模型列表
func (r *GroupRepository) GetModels(ctx context.Context, groupID string) ([]*domain.GroupModel, error) {
	var models []*domain.GroupModel
	err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&models).Error
	return models, err
}

// AddModel 添加模型到分组
func (r *GroupRepository) AddModel(ctx context.Context, gm *domain.GroupModel) error {
	return r.db.WithContext(ctx).Create(gm).Error
}

// RemoveModel 从分组移除模型
func (r *GroupRepository) RemoveModel(ctx context.Context, groupID, modelID string) error {
	return r.db.WithContext(ctx).Where("group_id = ? AND model_id = ?", groupID, modelID).Delete(&domain.GroupModel{}).Error
}

// RemoveAllModels 清空分组的所有模型绑定
func (r *GroupRepository) RemoveAllModels(ctx context.Context, groupID string) error {
	return r.db.WithContext(ctx).Where("group_id = ?", groupID).Delete(&domain.GroupModel{}).Error
}

// GetByModelID 根据模型ID查找该模型所属的分组
func (r *GroupRepository) GetByModelID(ctx context.Context, modelID string) (*domain.Group, error) {
	var group domain.Group
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN group_models ON group_models.group_id = groups.id").
		Where("group_models.model_id = ? AND group_models.enabled = ? AND groups.enabled = ?", modelID, true, true).
		First(&group).Error
	return &group, err
}
