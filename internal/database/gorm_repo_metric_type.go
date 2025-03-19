package database

import (
	"gorm.io/gorm"

	"context"

	"errors"

	"fulcrumproject.org/core/internal/domain"
)

type GormMetricTypeRepository struct {
	*GormRepository[domain.MetricType]
}

var applyMetricTypeFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyMetricTypeSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewMetricTypeRepository creates a new instance of MetricTypeRepository
func NewMetricTypeRepository(db *gorm.DB) *GormMetricTypeRepository {
	repo := &GormMetricTypeRepository{
		GormRepository: NewGormRepository[domain.MetricType](
			db,
			applyMetricTypeFilter,
			applyMetricTypeSort,
			nil,        // No authz filters
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// FindByName retrieves a metric type by name
func (r *GormMetricTypeRepository) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	var entity domain.MetricType
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&entity)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: errors.New("metric type not found")}
		}
		return nil, result.Error
	}
	return &entity, nil
}
