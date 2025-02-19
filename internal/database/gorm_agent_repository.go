package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormAgentRepository struct {
	*GormRepository[domain.Agent]
}

var applyAgentFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":        stringInFilterFieldApplier("name"),
	"state":       parserInFilterFieldApplier("state", domain.ParseAgentState),
	"countryCode": parserInFilterFieldApplier("country_code", domain.ParseCountryCode),
	"providerId":  parserInFilterFieldApplier("provider_id", domain.ParseUUID),
	"agentTypeId": parserInFilterFieldApplier("agent_type_id", domain.ParseUUID),
})

var applyAgentSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewAgentRepository creates a new instance of AgentRepository
func NewAgentRepository(db *gorm.DB) domain.AgentRepository {
	repo := &gormAgentRepository{
		GormRepository: NewGormRepository[domain.Agent](
			db,
			applyAgentFilter,
			applyAgentSort,
			[]string{"Provider", "AgentType"}, // Find preload paths
			[]string{"Provider"},              // List preload paths (only Provider for list operations)
		),
	}
	return repo
}

// CountByProvider returns the number of agents for a specific provider
func (r *gormAgentRepository) CountByProvider(ctx context.Context, providerID domain.UUID) (int64, error) {
	return r.Count(ctx, "provider_id = ?", providerID)
}
