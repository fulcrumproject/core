package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormServiceRepository struct {
	*GormRepository[domain.Service]
}

var applyServiceFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":         stringInFilterFieldApplier("services.name"),
	"currentState": parserInFilterFieldApplier("services.current_state", domain.ParseServiceState),
})

var applyServiceSort = mapSortApplier(map[string]string{
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

func (r *GormServiceRepository) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("group_id = ?", groupID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormServiceRepository) CountByAgent(ctx context.Context, agentID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("agent_id = ?", agentID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// FindByExternalID retrieves a service by its external ID and agent ID
func (r *GormServiceRepository) FindByExternalID(ctx context.Context, agentID domain.UUID, externalID string) (*domain.Service, error) {
	var service domain.Service

	result := r.db.WithContext(ctx).
		Where("external_id = ? AND agent_id = ?", externalID, agentID).
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

func (r *GormServiceRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	return r.getAuthScope(ctx, id, "provider_id", "consumer_id", "agent_id")
}
