package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormServicePoolSetRepository struct {
	*GormRepository[domain.ServicePoolSet]
}

var applyServicePoolSetFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"providerId": ParserInFilterFieldApplier("provider_id", properties.ParseUUID),
	"name":       StringContainsInsensitiveFilterFieldApplier("name"),
})

var applyServicePoolSetSort = MapSortApplier(map[string]string{
	"name":      "name",
	"createdAt": "created_at",
})

// servicePoolSetAuthzFilterApplier applies authorization scoping to service pool set queries
func servicePoolSetAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("provider_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		// Agents can only access pool sets for their provider
		return q.Joins("JOIN agents ON agents.provider_id = service_pool_sets.provider_id").
			Where("agents.id = ?", s.AgentID)
	}
	return q
}

// NewServicePoolSetRepository creates a new instance of ServicePoolSetRepository
func NewServicePoolSetRepository(db *gorm.DB) *GormServicePoolSetRepository {
	repo := &GormServicePoolSetRepository{
		GormRepository: NewGormRepository[domain.ServicePoolSet](
			db,
			applyServicePoolSetFilter,
			applyServicePoolSetSort,
			servicePoolSetAuthzFilterApplier,
			[]string{"Provider"}, // No preload paths needed
			[]string{"Provider"}, // No preload paths needed
		),
	}
	return repo
}

// FindByProvider retrieves all service pool sets for a provider
func (r *GormServicePoolSetRepository) FindByProvider(
	ctx context.Context,
	providerID properties.UUID,
) ([]*domain.ServicePoolSet, error) {
	var entities []*domain.ServicePoolSet
	result := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// FindByProviderAndName retrieves a service pool set by provider and name
func (r *GormServicePoolSetRepository) FindByProviderAndName(
	ctx context.Context,
	providerID properties.UUID,
	name string,
) (*domain.ServicePoolSet, error) {
	var entity domain.ServicePoolSet
	result := r.db.WithContext(ctx).
		Where("provider_id = ? AND name = ?", providerID, name).
		First(&entity)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}
	return &entity, nil
}

// Update updates an existing service pool set
func (r *GormServicePoolSetRepository) Update(ctx context.Context, poolSet *domain.ServicePoolSet) error {
	return r.Save(ctx, poolSet)
}

// AuthScope returns the authorization scope for a service pool set
func (r *GormServicePoolSetRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "null", "null")
}
