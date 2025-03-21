package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormServiceTypeRepository struct {
	*GormRepository[domain.ServiceType]
}

var applyServiceTypeFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyServiceTypeSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewServiceTypeRepository creates a new instance of ServiceTypeRepository
func NewServiceTypeRepository(db *gorm.DB) *GormServiceTypeRepository {
	repo := &GormServiceTypeRepository{
		GormRepository: NewGormRepository[domain.ServiceType](
			db,
			applyServiceTypeFilter,
			applyServiceTypeSort,
			nil,        // No authz filters
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// Count returns the total number of service types
func (r *GormServiceTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.GormRepository.Count(ctx)
}

// AuthScope returns the auth scope for the service type
func (r *GormServiceTypeRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	// Service types don't have scoping IDs as they are global resources
	return &domain.AuthScope{}, nil
}
