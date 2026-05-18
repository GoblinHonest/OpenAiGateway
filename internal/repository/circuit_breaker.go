package repository

import (
	"context"

	"github.com/example/aigateway/internal/domain"
	"gorm.io/gorm"
)

type CircuitBreakerRepository struct {
	db *gorm.DB
}

func NewCircuitBreakerRepository(db *gorm.DB) *CircuitBreakerRepository {
	return &CircuitBreakerRepository{db: db}
}

func (r *CircuitBreakerRepository) Save(ctx context.Context, state *domain.CircuitBreakerState) error {
	return r.db.WithContext(ctx).Save(state).Error
}

func (r *CircuitBreakerRepository) GetByProvider(ctx context.Context, providerID string) (*domain.CircuitBreakerState, error) {
	var state domain.CircuitBreakerState
	err := r.db.WithContext(ctx).First(&state, "provider_id = ?", providerID).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *CircuitBreakerRepository) ListAll(ctx context.Context) ([]*domain.CircuitBreakerState, error) {
	var states []*domain.CircuitBreakerState
	err := r.db.WithContext(ctx).Find(&states).Error
	return states, err
}
