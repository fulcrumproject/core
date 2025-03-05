package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormServiceTypeRepository struct {
	*GormRepository[domain.ServiceType]
}

var applyServiceTypeFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyServiceTypeSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewServiceTypeRepository creates a new instance of ServiceTypeRepository
func NewServiceTypeRepository(db *gorm.DB) domain.ServiceTypeRepository {
	repo := &gormServiceTypeRepository{
		GormRepository: NewGormRepository[domain.ServiceType](
			db,
			applyServiceTypeFilter,
			applyServiceTypeSort,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// Count returns the total number of service types
func (r *gormServiceTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.GormRepository.Count(ctx)
}
