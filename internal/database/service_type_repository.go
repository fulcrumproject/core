package database

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type serviceTypeRepository struct {
	db *gorm.DB
}

// NewServiceTypeRepository creates a new instance of ServiceTypeRepository
func NewServiceTypeRepository(db *gorm.DB) domain.ServiceTypeRepository {
	return &serviceTypeRepository{db: db}
}

func (r *serviceTypeRepository) Create(ctx context.Context, serviceType *domain.ServiceType) error {
	result := r.db.WithContext(ctx).Create(serviceType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceTypeRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
	var serviceType domain.ServiceType
	err := r.db.WithContext(ctx).
		First(&serviceType, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &serviceType, nil
}

func (r *serviceTypeRepository) List(ctx context.Context, filters domain.Filters, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.ServiceType], error) {
	var serviceTypes []domain.ServiceType
	var totalItems int64

	query := r.db.WithContext(ctx)

	// Apply filters
	for key, value := range filters {
		query = query.Where(key, value)
	}

	// Get total count for pagination
	if err := query.Model(&domain.ServiceType{}).Count(&totalItems).Error; err != nil {
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

	if err := query.Find(&serviceTypes).Error; err != nil {
		return nil, err
	}

	totalPages := int(totalItems) / pagination.PageSize
	if int(totalItems)%pagination.PageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.ServiceType]{
		Items:       serviceTypes,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: pagination.Page,
		HasNext:     pagination.Page < totalPages,
		HasPrev:     pagination.Page > 1,
	}, nil
}

func (r *serviceTypeRepository) Count(ctx context.Context, filters domain.Filters) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&domain.ServiceType{})
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
