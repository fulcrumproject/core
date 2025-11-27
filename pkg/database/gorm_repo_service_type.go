package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormServiceTypeRepository struct {
	*GormRepository[domain.ServiceType]
}

var applyServiceTypeFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name": StringContainsInsensitiveFilterFieldApplier("name"),
})

var applyServiceTypeSort = MapSortApplier(map[string]string{
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
func (r *GormServiceTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Service types don't have scoping IDs as they are global resources
	return &auth.AllwaysMatchObjectScope{}, nil
}
