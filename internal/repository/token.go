package repository

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Create(ctx context.Context, token *domain.Token) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) GetByID(ctx context.Context, id string) (*domain.Token, error) {
	var token domain.Token
	err := r.db.WithContext(ctx).First(&token, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *TokenRepository) ListByProvider(ctx context.Context, providerID string, status string, page, pageSize int) ([]*domain.Token, int64, error) {
	var tokens []*domain.Token
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Token{})
	if providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
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

	if err := query.Offset(offset).Limit(pageSize).Find(&tokens).Error; err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}

func (r *TokenRepository) Update(ctx context.Context, token *domain.Token) error {
	result := r.db.WithContext(ctx).Model(&domain.Token{}).
		Where("id = ? AND version = ?", token.ID, token.Version).
		Updates(map[string]any{
			"name":                token.Name,
			"token_value":         token.TokenValue,
			"status":              token.Status,
			"quota_total":         token.QuotaTotal,
			"quota_remaining":     token.QuotaRemaining,
			"quota_reset_at":      token.QuotaResetAt,
			"metadata":            token.Metadata,
			"version":             gorm.Expr("version + 1"),
		})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *TokenRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Token{}, "id = ?", id).Error
}

func (r *TokenRepository) DecrementQuota(ctx context.Context, id string, amount int64, expectedVersion int) (bool, error) {
	result := r.db.WithContext(ctx).Exec(
		`UPDATE tokens SET quota_remaining = quota_remaining - ?, 
		 version = version + 1, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ? AND quota_remaining >= ? AND status = ? AND version = ?`,
		amount, id, amount, domain.TokenStatusActive, expectedVersion,
	)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TokenRepository) GetAvailableByProvider(ctx context.Context, providerID string) ([]*domain.Token, error) {
	var tokens []*domain.Token
	// quota_total = 0 表示无配额限制，quota_total > 0 时需要 quota_remaining > 0
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND status = ? AND (quota_total = 0 OR quota_remaining > 0) AND rate_limited = ?",
			providerID, domain.TokenStatusActive, false).
		Find(&tokens).Error
	return tokens, err
}

func (r *TokenRepository) UpdateStatus(ctx context.Context, id string, status domain.TokenStatus) error {
	return r.db.WithContext(ctx).Model(&domain.Token{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *TokenRepository) RecordSuccess(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.Token{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"consecutive_failures": 0,
			"last_success_at":      gorm.Expr("CURRENT_TIMESTAMP"),
			"last_used_at":         gorm.Expr("CURRENT_TIMESTAMP"),
			"total_requests":       gorm.Expr("total_requests + 1"),
			"success_requests":     gorm.Expr("success_requests + 1"),
		}).Error
}

func (r *TokenRepository) RecordFailure(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.Token{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"failure_count":         gorm.Expr("failure_count + 1"),
			"consecutive_failures":  gorm.Expr("consecutive_failures + 1"),
			"last_failure_at":       gorm.Expr("CURRENT_TIMESTAMP"),
			"last_used_at":          gorm.Expr("CURRENT_TIMESTAMP"),
			"total_requests":        gorm.Expr("total_requests + 1"),
		}).Error
}
