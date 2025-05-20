package database

import (
	"context"
	"time"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormAgentRepository struct {
	*GormRepository[domain.Agent]
}

var applyAgentFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":          stringInFilterFieldApplier("name"),
	"state":         parserInFilterFieldApplier("state", domain.ParseAgentState),
	"countryCode":   parserInFilterFieldApplier("country_code", domain.ParseCountryCode),
	"participantId": parserInFilterFieldApplier("participant_id", domain.ParseUUID),
	"agentTypeId":   parserInFilterFieldApplier("agent_type_id", domain.ParseUUID),
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
			agentAuthzFilterApplier,
			[]string{"Participant", "AgentType"}, // Find preload paths
			[]string{"Participant"},              // List preload paths (only Participant for list operations)
		),
	}
	return repo
}

func (r *GormAgentRepository) CountByParticipant(ctx context.Context, participantID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.Agent{}).Where("participant_id = ?", participantID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormAgentRepository) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-inactiveDuration)

	result := r.db.WithContext(ctx).
		Model(&domain.Agent{}).
		Where("state = ?", domain.AgentConnected).
		Where("last_state_update < ? OR last_state_update IS NULL", cutoffTime).
		Updates(map[string]interface{}{
			"state": domain.AgentDisconnected,
		})

	return result.RowsAffected, result.Error
}

// agentAuthzFilterApplier applies authorization scoping to agent queries
func agentAuthzFilterApplier(s *domain.AuthIdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("participant_id = ?", s.ParticipantID)
	}
	if s.AgentID != nil {
		return q.Where("id = ?", s.AgentID)
	}
	return q
}

// AuthScope returns the auth scope for the agent
func (r *GormAgentRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	return r.getAuthScope(ctx, id, "participant_id", "id as agent_id")
}
