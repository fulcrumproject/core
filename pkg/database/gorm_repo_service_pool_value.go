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
	"name":          StringContainsInsensitiveFilterFieldApplier("name"),
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
		return q.Where("service_pool_values.participant_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		return q.Joins("JOIN agents ON agents.provider_id = service_pool_values.participant_id").
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
			[]string{"ServicePool", "Service"},
			[]string{"ServicePool", "Service"},
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

func (r *GormServicePoolValueRepository) CountByPool(ctx context.Context, poolID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.ServicePoolValue{}).Where("service_pool_id = ?", poolID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormServicePoolValueRepository) ReleaseByService(ctx context.Context, serviceID properties.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.ServicePoolValue{}).Where("service_id = ?", serviceID).Updates(map[string]any{
		"service_id":    nil,
		"property_name": nil,
		"allocated_at":  nil,
	}).Error
}

// AuthScope returns the authorization scope for a service pool value via its denormalized participant_id.
func (r *GormServicePoolValueRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "participant_id", "null", "null", "null")
}
