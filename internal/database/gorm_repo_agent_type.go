package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormAgentTypeRepository struct {
	*GormRepository[domain.AgentType]
}

var applyAgentTypeFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyAgentTypeSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewAgentTypeRepository creates a new instance of AgentTypeRepository
func NewAgentTypeRepository(db *gorm.DB) domain.AgentTypeRepository {
	repo := &gormAgentTypeRepository{
		GormRepository: NewGormRepository[domain.AgentType](
			db,
			applyAgentTypeFilter,
			applyAgentTypeSort,
			[]string{"ServiceTypes"}, // Find preload paths
			[]string{"ServiceTypes"}, // List preload paths
		),
	}
	return repo
}

// Count returns the total number of agent types
func (r *gormAgentTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.GormRepository.Count(ctx)
}
