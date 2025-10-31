// AgentType repository implementation using GORM Gen
// Provides type-safe database operations for AgentType entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenAgentTypeRepository struct {
	q *Query
}

func NewGenAgentTypeRepository(db *gorm.DB) *GenAgentTypeRepository {
	return &GenAgentTypeRepository{
		q: Use(db),
	}
}

// Create inserts a new agent type
func (r *GenAgentTypeRepository) Create(ctx context.Context, agentType *domain.AgentType) error {
	return r.q.AgentType.WithContext(ctx).Create(agentType)
}

// Save updates an existing agent type
func (r *GenAgentTypeRepository) Save(ctx context.Context, agentType *domain.AgentType) error {
	result, err := r.q.AgentType.WithContext(ctx).
		Where(r.q.AgentType.ID.Eq(agentType.ID)).
		Updates(agentType)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Delete removes an agent type by ID
func (r *GenAgentTypeRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.AgentType.WithContext(ctx).
		Where(r.q.AgentType.ID.Eq(id)).
		Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Get retrieves an agent type by ID with preloads
func (r *GenAgentTypeRepository) Get(ctx context.Context, id properties.UUID) (*domain.AgentType, error) {
	agentType, err := r.q.AgentType.WithContext(ctx).
		Preload(r.q.AgentType.ServiceTypes).
		Where(r.q.AgentType.ID.Eq(id)).
		First()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return agentType, nil
}

// Exists checks if an agent type exists
func (r *GenAgentTypeRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.AgentType.WithContext(ctx).
		Where(r.q.AgentType.ID.Eq(id)).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count returns total count of all agent types
func (r *GenAgentTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.q.AgentType.WithContext(ctx).Count()
}

// List returns paginated agent types with filters and sorting
func (r *GenAgentTypeRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.AgentType], error) {
	query := r.q.AgentType.WithContext(ctx).Preload(r.q.AgentType.ServiceTypes)

	result, err := PaginateQuery(
		ctx,
		query,
		pageReq,
		applyGenAgentTypeFilters,
		applyGenAgentTypeSort,
	)
	if err != nil {
		return nil, err
	}

	// Convert []*AgentType to []AgentType to match interface
	items := make([]domain.AgentType, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}

	return &domain.PageRes[domain.AgentType]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}, nil
}

// AuthScope returns the authorization scope for an agent type (global resource)
func (r *GenAgentTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return &auth.AllwaysMatchObjectScope{}, nil
}

// applyGenAgentTypeFilters applies request filters
func applyGenAgentTypeFilters(query IAgentTypeDo, pageReq *domain.PageReq) IAgentTypeDo {
	q := Use(nil).AgentType

	if values, ok := pageReq.Filters["name"]; ok && len(values) > 0 {
		query = query.Where(q.Name.In(values...))
	}

	return query
}

// applyGenAgentTypeSort applies sorting
func applyGenAgentTypeSort(query IAgentTypeDo, pageReq *domain.PageReq) IAgentTypeDo {
	if !pageReq.Sort {
		return query
	}

	q := Use(nil).AgentType

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
