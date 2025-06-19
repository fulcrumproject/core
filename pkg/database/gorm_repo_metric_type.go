package database

import (
	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"gorm.io/gorm"

	"context"

	"fulcrumproject.org/core/pkg/domain"
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
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}
	return &entity, nil
}

func (r *GormMetricTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Metric types don't have scoping IDs as they are global resources
	return &auth.AllwaysMatchObjectScope{}, nil
}
