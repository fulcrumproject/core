package database

import (
	"context"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormAgentTypeRepository struct {
	*GormRepository[domain.AgentType]
}

var applyAgentTypeFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyAgentTypeSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewAgentTypeRepository creates a new instance of AgentTypeRepository
func NewAgentTypeRepository(db *gorm.DB) *GormAgentTypeRepository {
	repo := &GormAgentTypeRepository{
		GormRepository: NewGormRepository[domain.AgentType](
			db,
			applyAgentTypeFilter,
			applyAgentTypeSort,
			nil,                      // No authz filters
			[]string{"ServiceTypes"}, // Find preload paths
			[]string{"ServiceTypes"}, // List preload paths
		),
	}
	return repo
}

// Count returns the total number of agent types
func (r *GormAgentTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.GormRepository.Count(ctx)
}

// AuthScope returns the auth scope for the agent type
func (r *GormAgentTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Agent types don't have scoping IDs as they are global resources
	return &auth.AllwaysMatchObjectScope{}, nil
}
