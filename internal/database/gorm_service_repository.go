package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormServiceRepository struct {
	*GormRepository[domain.Service]
}

var applyServiceFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":  stringInFilterFieldApplier("name"),
	"state": parserInFilterFieldApplier("state", domain.ParseServiceState),
})

var applyServiceSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewServiceRepository creates a new instance of ServiceRepository
func NewServiceRepository(db *gorm.DB) domain.ServiceRepository {
	repo := &gormServiceRepository{
		GormRepository: NewGormRepository[domain.Service](
			db,
			applyServiceFilter,
			applyServiceSort,
			[]string{"Agent", "ServiceType", "Group"}, // Find preload paths
			[]string{"Agent", "ServiceType", "Group"}, // List preload paths
		),
	}
	return repo
}

// CountByGroup returns the number of services in a specific group
func (r *gormServiceRepository) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	return r.Count(ctx, "group_id = ?", groupID)
}
