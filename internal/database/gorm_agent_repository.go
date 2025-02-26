package database

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// FindByTokenHash finds an agent by its token hash
func (r *gormAgentRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Agent, error) {
	var agent domain.Agent

	err := r.db.WithContext(ctx).
		Preload(clause.Associations).
		Where("token_hash = ?", tokenHash).
		First(&agent).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: fmt.Errorf("agent with token hash not found")}
		}
		return nil, err
	}

	return &agent, nil
}
