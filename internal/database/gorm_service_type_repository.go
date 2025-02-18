package database

import (
	"context"
	"errors"

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
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &serviceType, nil
}

var serviceTypeFilterConfigs = map[string]FilterConfig{
	"name": {},
}

func (r *serviceTypeRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.ServiceType], error) {
	var serviceTypes []domain.ServiceType
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.ServiceType{})
	query, totalItems, err := applyFindAndCount(query, filter, serviceTypeFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&serviceTypes).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(serviceTypes, totalItems, pagination), nil
}

func (r *serviceTypeRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.ServiceType{})
	_, count, err := applyFilterAndCount(query, filter, serviceTypeFilterConfigs)
	if err != nil {
		return 0, err
	}
	return count, nil
}
