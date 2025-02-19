package database

import (
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormMetricTypeRepository struct {
	*GormRepository[domain.MetricType]
}

var applyMetricTypeFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyMetricTypeSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewMetricTypeRepository creates a new instance of MetricTypeRepository
func NewMetricTypeRepository(db *gorm.DB) domain.MetricTypeRepository {
	repo := &gormMetricTypeRepository{
		GormRepository: NewGormRepository[domain.MetricType](
			db,
			applyMetricTypeFilter,
			applyMetricTypeSort,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}
