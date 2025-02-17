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

var providerFilterConfigs = map[string]FilterConfig{
	"name":        {},
	"state":       {Query: "state", Valuer: func(v string) (interface{}, error) { return domain.ParseProviderState(v) }},
	"countryCode": {Query: "country_code", Valuer: func(v string) (interface{}, error) { return domain.ParseCountryCode(v) }},
}

func (r *providerRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.Provider], error) {
	var providers []domain.Provider
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.Provider{})
	query, err := applySimpleFilter(query, filter, providerFilterConfigs)
	if err != nil {
		return nil, err
	}
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, domain.NewInternalError(err)
	}
	query, err = applySorting(query, sorting)
	if err != nil {
		return nil, err
	}
	query, err = applyPagination(query, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&providers).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(providers, totalItems, pagination), nil
}

func (r *providerRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&domain.Provider{})
	query, err := applySimpleFilter(query, filter, providerFilterConfigs)
	if err != nil {
		return 0, err
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, domain.NewInternalError(err)
	}

	return count, nil
}
