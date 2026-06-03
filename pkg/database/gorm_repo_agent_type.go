package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
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
			[]string{"ServiceTypes", "InfrastructureTypes"}, // Find preload paths
			[]string{"ServiceTypes", "InfrastructureTypes"}, // List preload paths
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

	if agentType.InfrastructureTypes != nil {
		err := r.GormRepository.db.WithContext(ctx).Model(agentType).Association("InfrastructureTypes").Replace(agentType.InfrastructureTypes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *GormAgentTypeRepository) CountByInfrastructureType(ctx context.Context, infrastructureTypeID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.AgentType{}).
		Joins("JOIN agent_type_infrastructure_types ON agent_types.id = agent_type_infrastructure_types.agent_type_id").
		Where("agent_type_infrastructure_types.infrastructure_type_id = ?", infrastructureTypeID).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// AuthScope returns the auth scope for the agent type
func (r *GormAgentTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	// Agent types don't have scoping IDs as they are global resources
	return &authz.AllwaysMatchObjectScope{}, nil
}
