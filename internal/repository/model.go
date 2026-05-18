package repository

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type ModelRepository struct {
	db *gorm.DB
}

func NewModelRepository(db *gorm.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

func (r *ModelRepository) Create(ctx context.Context, model *domain.Model) error {
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *ModelRepository) GetByID(ctx context.Context, id string) (*domain.Model, error) {
	var model domain.Model
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	return &model, err
}

func (r *ModelRepository) GetByName(ctx context.Context, name string) (*domain.Model, error) {
	var model domain.Model
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&model).Error
	return &model, err
}

func (r *ModelRepository) List(ctx context.Context, enabled bool, page, pageSize int) ([]*domain.Model, int64, error) {
	var models []*domain.Model
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Model{})
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

	if err := query.Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	return models, total, nil
}

func (r *ModelRepository) Update(ctx context.Context, model *domain.Model) error {
	return r.db.WithContext(ctx).Model(&domain.Model{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"name":                model.Name,
			"display_name":        model.DisplayName,
			"description":         model.Description,
			"model_type":          model.ModelType,
			"context_window":      model.ContextWindow,
			"max_output_tokens":   model.MaxOutputTokens,
			"input_price_per_1k":  model.InputPricePer1K,
			"output_price_per_1k": model.OutputPricePer1K,
			"enabled":             model.Enabled,
			"metadata":            model.Metadata,
		}).Error
}

func (r *ModelRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Model{}, "id = ?", id).Error
}

func (r *ModelRepository) BindProvider(ctx context.Context, binding *domain.ModelProviderBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *ModelRepository) GetBindings(ctx context.Context, modelID string) ([]*domain.ModelProviderBinding, error) {
	type bindingRow struct {
		ID                string
		ModelID           string
		ProviderID        string
		UpstreamModelName string
		Weight            int
		Priority          int
		Enabled           bool
		Version           int
		ProviderName      string
	}

	var rows []bindingRow
	err := r.db.WithContext(ctx).
		Table("model_provider_bindings").
		Select("model_provider_bindings.*, COALESCE(providers.name, '') AS provider_name").
		Joins("LEFT JOIN providers ON providers.id = model_provider_bindings.provider_id").
		Where("model_provider_bindings.model_id = ? AND model_provider_bindings.enabled = ?", modelID, true).
		Order("model_provider_bindings.priority DESC, model_provider_bindings.weight DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*domain.ModelProviderBinding, 0, len(rows))
	for _, r := range rows {
		bindings = append(bindings, &domain.ModelProviderBinding{
			ID:                r.ID,
			ModelID:           r.ModelID,
			ProviderID:        r.ProviderID,
			UpstreamModelName: r.UpstreamModelName,
			Weight:            r.Weight,
			Priority:          r.Priority,
			Enabled:           r.Enabled,
			Version:           r.Version,
			ProviderName:      r.ProviderName,
		})
	}
	return bindings, nil
}

func (r *ModelRepository) RemoveBinding(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.ModelProviderBinding{}, "id = ?", id).Error
}

func (r *ModelRepository) ListAllBindings(ctx context.Context) ([]*domain.ModelProviderBinding, error) {
	var bindings []*domain.ModelProviderBinding
	err := r.db.WithContext(ctx).Preload("Model").Preload("Provider").Find(&bindings).Error
	return bindings, err
}
