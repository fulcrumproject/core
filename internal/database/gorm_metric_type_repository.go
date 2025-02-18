package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type metricTypeRepository struct {
	db *gorm.DB
}

// NewMetricTypeRepository creates a new instance of MetricTypeRepository
func NewMetricTypeRepository(db *gorm.DB) domain.MetricTypeRepository {
	return &metricTypeRepository{db: db}
}

func (r *metricTypeRepository) Create(ctx context.Context, metricType *domain.MetricType) error {
	result := r.db.WithContext(ctx).Create(metricType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *metricTypeRepository) Save(ctx context.Context, metricType *domain.MetricType) error {
	result := r.db.WithContext(ctx).Save(metricType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *metricTypeRepository) Delete(ctx context.Context, id domain.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.MetricType{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *metricTypeRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
	var metricType domain.MetricType

	err := r.db.WithContext(ctx).First(&metricType, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &metricType, nil
}

var metricTypeFilterConfigs = map[string]FilterConfig{
	"name": {},
}

func (r *metricTypeRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.MetricType], error) {
	var metricTypes []domain.MetricType
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.MetricType{})

	query, totalItems, err := applyFindAndCount(query, filter, metricTypeFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&metricTypes).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(metricTypes, totalItems, pagination), nil
}

func (r *metricTypeRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.MetricType{})
	_, count, err := applyFilterAndCount(query, filter, metricTypeFilterConfigs)
	if err != nil {
		return 0, err
	}
	return count, nil
}
