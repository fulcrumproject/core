package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type providerRepository struct {
	db *gorm.DB
}

// NewProviderRepository creates a new instance of ProviderRepository
func NewProviderRepository(db *gorm.DB) domain.ProviderRepository {
	return &providerRepository{db: db}
}

func (r *providerRepository) Create(ctx context.Context, provider *domain.Provider) error {
	result := r.db.WithContext(ctx).Create(provider)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *providerRepository) Save(ctx context.Context, provider *domain.Provider) error {
	result := r.db.WithContext(ctx).Save(provider)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *providerRepository) Delete(ctx context.Context, id domain.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Provider{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *providerRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.Provider, error) {
	var provider domain.Provider
	err := r.db.WithContext(ctx).First(&provider, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &provider, nil
}

func (r *providerRepository) List(ctx context.Context, filters map[string]interface{}) ([]domain.Provider, error) {
	var providers []domain.Provider

	query := r.db.WithContext(ctx)
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Find(&providers).Error; err != nil {
		return nil, err
	}

	return providers, nil
}
