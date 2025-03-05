package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormServiceRepository struct {
	*GormRepository[domain.Service]
}

var applyServiceFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":         stringInFilterFieldApplier("name"),
	"currentState": parserInFilterFieldApplier("current_state", domain.ParseServiceState),
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

func (r *gormServiceRepository) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("group_id = ?", groupID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *gormServiceRepository) CountByAgent(ctx context.Context, agentID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Service{}).Where("agent_id = ?", agentID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// FindByExternalID retrieves a service by its external ID and agent ID
func (r *gormServiceRepository) FindByExternalID(ctx context.Context, externalID string, agentID domain.UUID) (*domain.Service, error) {
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
