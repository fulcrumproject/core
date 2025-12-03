package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormAgentRepository struct {
	*GormRepository[domain.Agent]
}

var applyAgentFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":        StringContainsInsensitiveFilterFieldApplier("name"),
	"status":      ParserInFilterFieldApplier("status", domain.ParseAgentStatus),
	"providerId":  ParserInFilterFieldApplier("provider_id", properties.ParseUUID),
	"agentTypeId": ParserInFilterFieldApplier("agent_type_id", properties.ParseUUID),
})

var applyAgentSort = MapSortApplier(map[string]string{
	"name": "name",
})

// NewAgentRepository creates a new instance of AgentRepository
func NewAgentRepository(db *gorm.DB) *GormAgentRepository {
	repo := &GormAgentRepository{
		GormRepository: NewGormRepository[domain.Agent](
			db,
			applyAgentFilter,
			applyAgentSort,
			agentAuthzFilterApplier,
			[]string{"Provider", "AgentType", "AgentType.ServiceTypes", "ServicePoolSet"}, // Find preload paths
			[]string{"Provider"}, // List preload paths (only Provider for list operations)
		),
	}
	return repo
}

func (r *GormAgentRepository) CountByProvider(ctx context.Context, providerID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Agent{}).Where("provider_id = ?", providerID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormAgentRepository) CountByAgentType(ctx context.Context, agentTypeID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Agent{}).Where("agent_type_id = ?", agentTypeID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormAgentRepository) FindByServiceTypeAndTags(ctx context.Context, serviceTypeID properties.UUID, tags []string) ([]*domain.Agent, error) {
	var agents []*domain.Agent

	query := r.db.WithContext(ctx).
		Joins("JOIN agent_types ON agents.agent_type_id = agent_types.id").
		Joins("JOIN agent_type_service_types ON agent_types.id = agent_type_service_types.agent_type_id").
		Where("agent_type_service_types.service_type_id = ?", serviceTypeID)

	if len(tags) > 0 {
		query = query.Where("agents.tags @> ?", pq.StringArray(tags))
	}

	result := query.Preload("Provider").Preload("AgentType").Preload("AgentType.ServiceTypes").Find(&agents)
	if result.Error != nil {
		return nil, result.Error
	}

	return agents, nil
}

func (r *GormAgentRepository) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-inactiveDuration)

	result := r.db.WithContext(ctx).
		Model(&domain.Agent{}).
		Where("status = ?", domain.AgentConnected).
		Where("last_status_update < ? OR last_status_update IS NULL", cutoffTime).
		Updates(map[string]any{
			"status": domain.AgentDisconnected,
		})

	return result.RowsAffected, result.Error
}

// agentAuthzFilterApplier applies authorization scoping to agent queries
func agentAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("provider_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		return q.Where("id = ?", s.AgentID)
	}
	return q
}

// AuthScope returns the auth scope for the agent
func (r *GormAgentRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "id as agent_id", "null")
}
