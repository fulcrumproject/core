package database

import (
	"context"
	"errors"
	"fmt"

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

func (r *providerRepository) List(ctx context.Context, filters domain.Filters, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.Provider], error) {
	var providers []domain.Provider
	var totalItems int64

	query := r.db.WithContext(ctx)

	// Apply filters
	for key, value := range filters {
		query = query.Where(key, value)
	}

	// Get total count for pagination
	if err := query.Model(&domain.Provider{}).Count(&totalItems).Error; err != nil {
		return nil, err
	}

	// Apply sorting if provided
	if sorting != nil && sorting.SortField != "" {
		order := "asc"
		if sorting.SortOrder == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", sorting.SortField, order))
	}

	// Apply pagination if provided
	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	if err := query.Find(&providers).Error; err != nil {
		return nil, err
	}

	totalPages := int(totalItems) / pagination.PageSize
	if int(totalItems)%pagination.PageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.Provider]{
		Items:       providers,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: pagination.Page,
		HasNext:     pagination.Page < totalPages,
		HasPrev:     pagination.Page > 1,
	}, nil
}

func (r *providerRepository) Count(ctx context.Context, filters domain.Filters) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&domain.Provider{})
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
