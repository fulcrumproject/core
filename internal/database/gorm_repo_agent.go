package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"fulcrumproject.org/core/internal/domain"
)

type GormAgentRepository struct {
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
func NewAgentRepository(db *gorm.DB) *GormAgentRepository {
	repo := &GormAgentRepository{
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
func (r *GormAgentRepository) CountByProvider(ctx context.Context, providerID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Agent{}).Where("provider_id = ?", providerID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// FindByTokenHash finds an agent by its token hash
func (r *GormAgentRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Agent, error) {
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

// MarkInactiveAgentsAsDisconnected marks agents that haven't updated their status in the given duration as disconnected
func (r *GormAgentRepository) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-inactiveDuration)

	result := r.db.WithContext(ctx).
		Model(&domain.Agent{}).
		Where("state = ?", domain.AgentConnected).
		Where("last_status_update < ? OR last_status_update IS NULL", cutoffTime).
		Updates(map[string]interface{}{
			"state": domain.AgentDisconnected,
		})

	return result.RowsAffected, result.Error
}
