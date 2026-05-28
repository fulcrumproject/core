package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormInfrastructureRepository struct {
	*GormRepository[domain.Infrastructure]
}

var applyInfrastructureFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":                 StringContainsInsensitiveFilterFieldApplier("name"),
	"providerId":           ParserInFilterFieldApplier("provider_id", properties.ParseUUID),
	"infrastructureTypeId": ParserInFilterFieldApplier("infrastructure_type_id", properties.ParseUUID),
})

var applyInfrastructureSort = MapSortApplier(map[string]string{
	"name": "name",
})

// NewInfrastructureRepository creates a new InfrastructureRepository.
func NewInfrastructureRepository(db *gorm.DB) *GormInfrastructureRepository {
	return &GormInfrastructureRepository{
		GormRepository: NewGormRepository[domain.Infrastructure](
			db,
			applyInfrastructureFilter,
			applyInfrastructureSort,
			infrastructureAuthzFilterApplier,
			[]string{"Provider", "InfrastructureType"}, // Find preloads
			[]string{"Provider", "InfrastructureType"}, // List preloads
		),
	}
}

// infrastructureAuthzFilterApplier scopes infrastructure queries. The AgentID
// coordinate doubles as the self-reference for Infrastructure (no separate
// InfrastructureID scope field — reused per Phase 2 plan).
func infrastructureAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("provider_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		return q.Where("id = ?", s.AgentID)
	}
	return q
}

// AuthScope returns the auth scope for the infrastructure. Reuses the
// AgentID coordinate to expose self-access by the install-token flow.
func (r *GormInfrastructureRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "id as agent_id", "null")
}
