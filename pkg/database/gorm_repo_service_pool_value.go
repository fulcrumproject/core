package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormServicePoolValueRepository struct {
	*GormRepository[domain.ServicePoolValue]
}

var applyServicePoolValueFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"servicePoolId": ParserInFilterFieldApplier("service_pool_id", properties.ParseUUID),
	"serviceId":     ParserInFilterFieldApplier("service_id", properties.ParseUUID),
})

var applyServicePoolValueSort = MapSortApplier(map[string]string{
	"name":      "name",
	"createdAt": "created_at",
})

// servicePoolValueAuthzFilterApplier applies authorization scoping to service pool value queries
func servicePoolValueAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		// Participants can only access values in their pool sets
		return q.Joins("JOIN service_pools ON service_pools.id = service_pool_values.service_pool_id").
			Joins("JOIN service_pool_sets ON service_pool_sets.id = service_pools.service_pool_set_id").
			Where("service_pool_sets.provider_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		// Agents can only access values for their provider
		return q.Joins("JOIN service_pools ON service_pools.id = service_pool_values.service_pool_id").
			Joins("JOIN service_pool_sets ON service_pool_sets.id = service_pools.service_pool_set_id").
			Joins("JOIN agents ON agents.provider_id = service_pool_sets.provider_id").
			Where("agents.id = ?", s.AgentID)
	}
	return q
}

// NewServicePoolValueRepository creates a new instance of ServicePoolValueRepository
func NewServicePoolValueRepository(db *gorm.DB) *GormServicePoolValueRepository {
	repo := &GormServicePoolValueRepository{
		GormRepository: NewGormRepository[domain.ServicePoolValue](
			db,
			applyServicePoolValueFilter,
			applyServicePoolValueSort,
			servicePoolValueAuthzFilterApplier,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// ListByPool retrieves all values for a pool
func (r *GormServicePoolValueRepository) ListByPool(
	ctx context.Context,
	poolID properties.UUID,
) ([]*domain.ServicePoolValue, error) {
	var entities []*domain.ServicePoolValue
	result := r.db.WithContext(ctx).
		Where("service_pool_id = ?", poolID).
		Order("name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// ListByService retrieves all values allocated to a service
func (r *GormServicePoolValueRepository) ListByService(
	ctx context.Context,
	serviceID properties.UUID,
) ([]*domain.ServicePoolValue, error) {
	var entities []*domain.ServicePoolValue
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// FindByPool retrieves all values for a pool (used for allocation logic)
func (r *GormServicePoolValueRepository) FindByPool(
	ctx context.Context,
	poolID properties.UUID,
) ([]*domain.ServicePoolValue, error) {
	var entities []*domain.ServicePoolValue
	result := r.db.WithContext(ctx).
		Where("service_pool_id = ?", poolID).
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// FindAvailable retrieves available (unallocated) values for a pool
func (r *GormServicePoolValueRepository) FindAvailable(
	ctx context.Context,
	poolID properties.UUID,
) ([]*domain.ServicePoolValue, error) {
	var entities []*domain.ServicePoolValue
	result := r.db.WithContext(ctx).
		Where("service_pool_id = ? AND service_id IS NULL", poolID).
		Order("name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// FindByService retrieves all values allocated to a service
func (r *GormServicePoolValueRepository) FindByService(
	ctx context.Context,
	serviceID properties.UUID,
) ([]*domain.ServicePoolValue, error) {
	var entities []*domain.ServicePoolValue
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// Update updates an existing service pool value
func (r *GormServicePoolValueRepository) Update(ctx context.Context, value *domain.ServicePoolValue) error {
	return r.Save(ctx, value)
}

// AuthScope returns the authorization scope for a service pool value (via pool -> pool set -> provider)
func (r *GormServicePoolValueRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	// Join through service_pools and service_pool_sets to get provider_id
	var result struct {
		ProviderID properties.UUID `gorm:"column:provider_id"`
	}

	err := r.db.WithContext(ctx).
		Table("service_pool_values").
		Select("service_pool_sets.provider_id").
		Joins("JOIN service_pools ON service_pools.id = service_pool_values.service_pool_id").
		Joins("JOIN service_pool_sets ON service_pool_sets.id = service_pools.service_pool_set_id").
		Where("service_pool_values.id = ?", id).
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &authz.DefaultObjectScope{ProviderID: &result.ProviderID}, nil
}
