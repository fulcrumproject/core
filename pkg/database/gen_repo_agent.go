// Agent repository implementation using GORM Gen
// Provides type-safe database operations for Agent entities
package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/lib/pq"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type GenAgentRepository struct {
	db *gorm.DB
	q  *Query
}

func NewGenAgentRepository(db *gorm.DB) *GenAgentRepository {
	return &GenAgentRepository{
		db: db,
		q:  Use(db),
	}
}

// Create creates a new agent
func (r *GenAgentRepository) Create(ctx context.Context, entity *domain.Agent) error {
	return r.q.Agent.WithContext(ctx).Create(entity)
}

// Save updates an existing agent
func (r *GenAgentRepository) Save(ctx context.Context, entity *domain.Agent) error {
	result, err := r.q.Agent.WithContext(ctx).
		Where(r.q.Agent.ID.Eq(entity.ID)).
		Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Delete deletes an agent
func (r *GenAgentRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.Agent.WithContext(ctx).
		Where(r.q.Agent.ID.Eq(id)).
		Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Get retrieves an agent by ID with preloading
func (r *GenAgentRepository) Get(ctx context.Context, id properties.UUID) (*domain.Agent, error) {
	agent, err := r.q.Agent.WithContext(ctx).
		Preload(r.q.Agent.Provider).
		Preload(r.q.Agent.AgentType).
		Preload(r.q.Agent.AgentType.ServiceTypes).
		Preload(r.q.Agent.ServicePoolSet).
		Where(r.q.Agent.ID.Eq(id)).
		First()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return agent, nil
}

// Exists checks if an agent exists
func (r *GenAgentRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.Agent.WithContext(ctx).
		Where(r.q.Agent.ID.Eq(id)).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count returns total count of all agents
func (r *GenAgentRepository) Count(ctx context.Context) (int64, error) {
	return r.q.Agent.WithContext(ctx).Count()
}

// List returns paginated agents with authorization, filters, and sorting
func (r *GenAgentRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.Agent], error) {
	query := r.q.Agent.WithContext(ctx).Preload(r.q.Agent.Provider)
	query = applyGenAgentAuthz(query, scope)

	result, err := PaginateQuery(
		ctx,
		query,
		pageReq,
		applyGenAgentFilters,
		applyGenAgentSort,
	)
	if err != nil {
		return nil, err
	}

	// Convert []*Agent to []Agent to match interface
	items := make([]domain.Agent, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}

	return &domain.PageRes[domain.Agent]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}, nil
}

// CountByProvider returns the number of agents for a specific provider
func (r *GenAgentRepository) CountByProvider(ctx context.Context, providerID properties.UUID) (int64, error) {
	return r.q.Agent.WithContext(ctx).
		Where(r.q.Agent.ProviderID.Eq(providerID)).
		Count()
}

// CountByAgentType returns the number of agents for a specific agent type
func (r *GenAgentRepository) CountByAgentType(ctx context.Context, agentTypeID properties.UUID) (int64, error) {
	return r.q.Agent.WithContext(ctx).
		Where(r.q.Agent.AgentTypeID.Eq(agentTypeID)).
		Count()
}

// FindByServiceTypeAndTags finds agents that support a service type and have all required tags
func (r *GenAgentRepository) FindByServiceTypeAndTags(ctx context.Context, serviceTypeID properties.UUID, tags []string) ([]*domain.Agent, error) {
	query := r.db.WithContext(ctx).
		Joins("JOIN agent_types ON agents.agent_type_id = agent_types.id").
		Joins("JOIN agent_type_service_types ON agent_types.id = agent_type_service_types.agent_type_id").
		Where("agent_type_service_types.service_type_id = ?", serviceTypeID)

	if len(tags) > 0 {
		query = query.Where("agents.tags @> ?", pq.StringArray(tags))
	}

	var agents []*domain.Agent
	if err := query.Preload("Provider").Preload("AgentType").Preload("AgentType.ServiceTypes").Find(&agents).Error; err != nil {
		return nil, err
	}

	return agents, nil
}

// MarkInactiveAgentsAsDisconnected marks agents that haven't updated their status in the given duration as disconnected
func (r *GenAgentRepository) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-inactiveDuration)

	result, err := r.q.Agent.WithContext(ctx).
		Where(r.q.Agent.Status.Eq(string(domain.AgentConnected))).
		Where(r.q.Agent.LastStatusUpdate.Lt(cutoffTime)).
		Or(r.q.Agent.LastStatusUpdate.IsNull()).
		UpdateSimple(r.q.Agent.Status.Value(string(domain.AgentDisconnected)))

	if err != nil {
		return 0, err
	}

	return result.RowsAffected, nil
}

// AuthScope returns the auth scope for the agent
func (r *GenAgentRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	agent, err := r.q.Agent.WithContext(ctx).
		Select(r.q.Agent.ProviderID, r.q.Agent.ID).
		Where(r.q.Agent.ID.Eq(id)).
		First()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &auth.DefaultObjectScope{
		ParticipantID: &agent.ProviderID,
		AgentID:       &agent.ID,
	}, nil
}

// Authorization helper - applies authorization scope to queries
func applyGenAgentAuthz(query IAgentDo, scope *auth.IdentityScope) IAgentDo {
	q := Use(nil).Agent

	if scope.ParticipantID != nil {
		return query.Where(q.ProviderID.Eq(*scope.ParticipantID))
	}
	if scope.AgentID != nil {
		return query.Where(q.ID.Eq(*scope.AgentID))
	}
	return query
}

// Filter helper - applies filters to queries
func applyGenAgentFilters(query IAgentDo, pageReq *domain.PageReq) IAgentDo {
	q := Use(nil).Agent

	if values, ok := pageReq.Filters["name"]; ok && len(values) > 0 {
		query = query.Where(q.Name.In(values...))
	}

	if values, ok := pageReq.Filters["status"]; ok && len(values) > 0 {
		statuses := make([]string, 0, len(values))
		for _, v := range values {
			if status, err := domain.ParseAgentStatus(v); err == nil {
				statuses = append(statuses, string(status))
			}
		}
		if len(statuses) > 0 {
			query = query.Where(q.Status.In(statuses...))
		}
	}

	if values, ok := pageReq.Filters["providerId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			// Build OR conditions for each ID
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ProviderID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}

	if values, ok := pageReq.Filters["agentTypeId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			// Build OR conditions for each ID
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.AgentTypeID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}

	return query
}

// Sort helper - applies sorting to queries
func applyGenAgentSort(query IAgentDo, pageReq *domain.PageReq) IAgentDo {
	if !pageReq.Sort {
		return query
	}

	q := Use(nil).Agent

	switch pageReq.SortBy {
	case "name":
		if pageReq.SortAsc {
			query = query.Order(q.Name)
		} else {
			query = query.Order(q.Name.Desc())
		}
	}

	return query
}
