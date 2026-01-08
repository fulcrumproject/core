package database

import (
	"context"
	"errors"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormServiceRepository struct {
	*GormRepository[domain.Service]
}

var applyServiceFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":          StringInFilterFieldApplier("services.name"),
	"currentStatus": StringInFilterFieldApplier("services.status"),
})

var applyServiceSort = MapSortApplier(map[string]string{
	"name": "services.name",
})

// NewServiceRepository creates a new instance of ServiceRepository
func NewServiceRepository(db *gorm.DB) *GormServiceRepository {
	repo := &GormServiceRepository{
		GormRepository: NewGormRepository[domain.Service](
			db,
			applyServiceFilter,
			applyServiceSort,
			providerConsumerAgentAuthzFilterApplier,
			[]string{"Agent", "ServiceType", "Group"}, // Find preload paths
			[]string{"Agent", "ServiceType", "Group"}, // List preload paths
		),
	}
	return repo
}

func (r *GormServiceRepository) CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("group_id = ?", groupID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormServiceRepository) CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("agent_id = ?", agentID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormServiceRepository) CountByServiceType(ctx context.Context, serviceTypeID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("service_type_id = ?", serviceTypeID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// FindByAgentInstanceID retrieves a service by its agent instance ID and agent ID
func (r *GormServiceRepository) FindByAgentInstanceID(ctx context.Context, agentID properties.UUID, agentInstanceID string) (*domain.Service, error) {
	var service domain.Service

	result := r.db.WithContext(ctx).
		Where("agent_instance_id = ? AND agent_id = ?", agentInstanceID, agentID).
		Preload("Agent").
		Preload("ServiceType").
		Preload("Group").
		First(&service)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}
	return &service, nil
}

func (r *GormServiceRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}
