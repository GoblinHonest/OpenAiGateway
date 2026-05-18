package repository

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type ProviderRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) Create(ctx context.Context, provider *domain.Provider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

func (r *ProviderRepository) GetByID(ctx context.Context, id string) (*domain.Provider, error) {
	var provider domain.Provider
	err := r.db.WithContext(ctx).First(&provider, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *ProviderRepository) List(ctx context.Context, status string, page, pageSize int) ([]*domain.Provider, int64, error) {
	var providers []*domain.Provider
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Provider{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&providers).Error; err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

func (r *ProviderRepository) Update(ctx context.Context, provider *domain.Provider) error {
	result := r.db.WithContext(ctx).Model(&domain.Provider{}).
		Where("id = ? AND version = ?", provider.ID, provider.Version).
		Updates(map[string]any{
			"name":               provider.Name,
			"description":        provider.Description,
			"base_url":           provider.BaseURL,
			"status":             provider.Status,
			"supported_formats":  provider.SupportedFormats,
			"endpoints":          provider.Endpoints,
			"rate_limit_config":  provider.RateLimitConfig,
			"timeout_config":     provider.TimeoutConfig,
			"retry_config":       provider.RetryConfig,
			"metadata":           provider.Metadata,
			"version":            gorm.Expr("version + 1"),
		})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *ProviderRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Provider{}, "id = ?", id).Error
}

func (r *ProviderRepository) FindByFormat(ctx context.Context, format string) ([]*domain.Provider, error) {
	var providers []*domain.Provider
	// 使用LIKE查询兼容SQLite和MySQL
	err := r.db.WithContext(ctx).Where("status = ? AND supported_formats LIKE ?", domain.ProviderStatusActive, "%"+format+"%").Find(&providers).Error
	return providers, err
}

func (r *ProviderRepository) ListAll(ctx context.Context) ([]*domain.Provider, error) {
	var providers []*domain.Provider
	err := r.db.WithContext(ctx).Find(&providers).Error
	return providers, err
}
