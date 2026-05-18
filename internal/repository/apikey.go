package repository

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *domain.APIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.db.WithContext(ctx).First(&key, "id = ?", id).Error
	return &key, err
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&key).Error
	return &key, err
}

func (r *APIKeyRepository) List(ctx context.Context, status string, groupID string, page, pageSize int) ([]*domain.APIKey, int64, error) {
	var keys []*domain.APIKey
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.APIKey{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if groupID != "" {
		query = query.Where("group_id = ?", groupID)
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

	if err := query.Offset(offset).Limit(pageSize).Find(&keys).Error; err != nil {
		return nil, 0, err
	}

	return keys, total, nil
}

func (r *APIKeyRepository) Update(ctx context.Context, apiKey *domain.APIKey) error {
	result := r.db.WithContext(ctx).Model(&domain.APIKey{}).
		Where("id = ? AND version = ?", apiKey.ID, apiKey.Version).
		Updates(map[string]any{
			"name":              apiKey.Name,
			"group_id":          apiKey.GroupID,
			"rate_limit_config": apiKey.RateLimitConfig,
			"quota_config":      apiKey.QuotaConfig,
			"status":            apiKey.Status,
			"expires_at":        apiKey.ExpiresAt,
			"version":           gorm.Expr("version + 1"),
		})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.APIKey{}, "id = ?", id).Error
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
