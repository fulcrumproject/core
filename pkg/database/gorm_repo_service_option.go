package database

import (
	"context"
	"encoding/json"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormServiceOptionRepository struct {
	*GormRepository[domain.ServiceOption]
}

var applyServiceOptionFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"providerId":          ParserInFilterFieldApplier("provider_id", properties.ParseUUID),
	"serviceOptionTypeId": ParserInFilterFieldApplier("service_option_type_id", properties.ParseUUID),
	"enabled":             ParserInFilterFieldApplier("enabled", parseBool),
})

var applyServiceOptionSort = MapSortApplier(map[string]string{
	"name":         "name",
	"displayOrder": "display_order",
})

// serviceOptionAuthzFilterApplier applies authorization scoping to service option queries
func serviceOptionAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("provider_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		// Agents can only access options for their provider
		return q.Joins("JOIN agents ON agents.provider_id = service_options.provider_id").
			Where("agents.id = ?", s.AgentID)
	}
	return q
}

// NewServiceOptionRepository creates a new instance of ServiceOptionRepository
func NewServiceOptionRepository(db *gorm.DB) *GormServiceOptionRepository {
	repo := &GormServiceOptionRepository{
		GormRepository: NewGormRepository[domain.ServiceOption](
			db,
			applyServiceOptionFilter,
			applyServiceOptionSort,
			serviceOptionAuthzFilterApplier,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// FindByProviderAndTypeAndValue retrieves a service option by provider, type, and value
// Only returns enabled options
func (r *GormServiceOptionRepository) FindByProviderAndTypeAndValue(
	ctx context.Context,
	providerID, typeID properties.UUID,
	value any,
) (*domain.ServiceOption, error) {
	// Convert value to JSON for comparison
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var entity domain.ServiceOption
	// Use JSONB equality comparison and filter by enabled
	result := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Where("service_option_type_id = ?", typeID).
		Where("value = ?", valueJSON).
		Where("enabled = ?", true).
		First(&entity)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}
	return &entity, nil
}

// ListByProvider retrieves all service options for a provider
func (r *GormServiceOptionRepository) ListByProvider(
	ctx context.Context,
	providerID properties.UUID,
) ([]*domain.ServiceOption, error) {
	var entities []*domain.ServiceOption
	result := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("display_order ASC, name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// ListByProviderAndType retrieves all service options for a provider and type
func (r *GormServiceOptionRepository) ListByProviderAndType(
	ctx context.Context,
	providerID, typeID properties.UUID,
) ([]*domain.ServiceOption, error) {
	var entities []*domain.ServiceOption
	result := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Where("service_option_type_id = ?", typeID).
		Order("display_order ASC, name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// ListEnabledByProviderAndType retrieves enabled service options for a provider and type
func (r *GormServiceOptionRepository) ListEnabledByProviderAndType(
	ctx context.Context,
	providerID, typeID properties.UUID,
) ([]*domain.ServiceOption, error) {
	var entities []*domain.ServiceOption
	result := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Where("service_option_type_id = ?", typeID).
		Where("enabled = ?", true).
		Order("display_order ASC, name ASC").
		Find(&entities)

	if result.Error != nil {
		return nil, result.Error
	}
	return entities, nil
}

// CountByServiceOptionType returns the count of options for a given type
func (r *GormServiceOptionRepository) CountByServiceOptionType(
	ctx context.Context,
	typeID properties.UUID,
) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.ServiceOption{}).
		Where("service_option_type_id = ?", typeID).
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormServiceOptionRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	var entity domain.ServiceOption
	result := r.db.WithContext(ctx).Select("provider_id").Where("id = ?", id).First(&entity)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}

	return &authz.DefaultObjectScope{
		ProviderID: &entity.ProviderID,
	}, nil
}
