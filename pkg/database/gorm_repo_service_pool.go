package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormServicePoolRepository struct {
	*GormRepository[domain.ServicePool]
}

var applyServicePoolFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"servicePoolSetId": ParserInFilterFieldApplier("service_pool_set_id", properties.ParseUUID),
	"type":             StringInFilterFieldApplier("type"),
	"generatorType":    StringInFilterFieldApplier("generator_type"),
})

var applyServicePoolSort = MapSortApplier(map[string]string{
	"name":      "name",
	"type":      "type",
	"createdAt": "created_at",
})

// servicePoolAuthzFilterApplier applies authorization scoping to service pool queries
func servicePoolAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		// Participants can only access pools in their pool sets
		return q.Joins("JOIN service_pool_sets ON service_pool_sets.id = service_pools.service_pool_set_id").
			Where("service_pool_sets.provider_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		// Agents can only access pools for their provider
		return q.Joins("JOIN service_pool_sets ON service_pool_sets.id = service_pools.service_pool_set_id").
			Joins("JOIN agents ON agents.provider_id = service_pool_sets.provider_id").
			Where("agents.id = ?", s.AgentID)
	}
	return q
}

// NewServicePoolRepository creates a new instance of ServicePoolRepository
func NewServicePoolRepository(db *gorm.DB) *GormServicePoolRepository {
	repo := &GormServicePoolRepository{
		GormRepository: NewGormRepository[domain.ServicePool](
			db,
			applyServicePoolFilter,
			applyServicePoolSort,
			servicePoolAuthzFilterApplier,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// ListByPoolSet retrieves all service pools for a pool set
func (r *GormServicePoolRepository) ListByPoolSet(
	ctx context.Context,
	poolSetID properties.UUID,
) ([]*domain.ServicePool, error) {
	var entities []*domain.ServicePool
	result := r.db.WithContext(ctx).
		Where("service_pool_set_id = ?", poolSetID).
		Order("name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// FindByPoolSetAndType retrieves a service pool by pool set and type
func (r *GormServicePoolRepository) FindByPoolSetAndType(
	ctx context.Context,
	poolSetID properties.UUID,
	poolType string,
) (*domain.ServicePool, error) {
	var entity domain.ServicePool
	result := r.db.WithContext(ctx).
		Where("service_pool_set_id = ? AND type = ?", poolSetID, poolType).
		First(&entity)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}
	return &entity, nil
}

// Update updates an existing service pool
func (r *GormServicePoolRepository) Update(ctx context.Context, pool *domain.ServicePool) error {
	return r.Save(ctx, pool)
}

// AuthScope returns the authorization scope for a service pool (via pool set -> provider)
func (r *GormServicePoolRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Join through service_pool_sets to get provider_id
	var result struct {
		ProviderID properties.UUID `gorm:"column:provider_id"`
	}
	
	err := r.db.WithContext(ctx).
		Table("service_pools").
		Select("service_pool_sets.provider_id").
		Joins("JOIN service_pool_sets ON service_pool_sets.id = service_pools.service_pool_set_id").
		Where("service_pools.id = ?", id).
		First(&result).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	
	return &auth.DefaultObjectScope{ProviderID: &result.ProviderID}, nil
}
