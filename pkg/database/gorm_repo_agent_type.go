package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormAgentTypeRepository struct {
	*GormRepository[domain.AgentType]
}

var applyAgentTypeFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name": StringContainsInsensitiveFilterFieldApplier("name"),
})

var applyAgentTypeSort = MapSortApplier(map[string]string{
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

// Save overrides the base Save method to handle many-to-many association replacement
func (r *GormAgentTypeRepository) Save(ctx context.Context, agentType *domain.AgentType) error {
	if err := r.GormRepository.Save(ctx, agentType); err != nil {
		return err
	}

	if agentType.ServiceTypes != nil {
		err := r.GormRepository.db.WithContext(ctx).Model(agentType).Association("ServiceTypes").Replace(agentType.ServiceTypes)
		if err != nil {
			return err
		}
	}

	return nil
}

// AuthScope returns the auth scope for the agent type
func (r *GormAgentTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Agent types don't have scoping IDs as they are global resources
	return &auth.AllwaysMatchObjectScope{}, nil
}
