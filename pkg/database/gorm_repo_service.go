package database

import (
	"context"
	"errors"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormServiceRepository struct {
	*GormRepository[domain.Service]
}

var applyServiceFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":          stringInFilterFieldApplier("services.name"),
	"currentStatus": parserInFilterFieldApplier("services.current_status", domain.ParseServiceStatus),
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

// FindByExternalID retrieves a service by its external ID and agent ID
func (r *GormServiceRepository) FindByExternalID(ctx context.Context, agentID properties.UUID, externalID string) (*domain.Service, error) {
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

func (r *GormServiceRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.getAuthScope(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}
