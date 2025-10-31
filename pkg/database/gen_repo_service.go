// Service repository implementation using GORM Gen
// Provides type-safe database operations for Service entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenServiceRepository struct {
	q *Query
}

func NewGenServiceRepository(db *gorm.DB) *GenServiceRepository {
	return &GenServiceRepository{q: Use(db)}
}

func (r *GenServiceRepository) Create(ctx context.Context, entity *domain.Service) error {
	return r.q.Service.WithContext(ctx).Create(entity)
}

func (r *GenServiceRepository) Save(ctx context.Context, entity *domain.Service) error {
	result, err := r.q.Service.WithContext(ctx).Where(r.q.Service.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.Service.WithContext(ctx).Where(r.q.Service.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceRepository) Get(ctx context.Context, id properties.UUID) (*domain.Service, error) {
	entity, err := r.q.Service.WithContext(ctx).
		Preload(r.q.Service.Agent).
		Preload(r.q.Service.ServiceType).
		Preload(r.q.Service.Group).
		Where(r.q.Service.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.Service.WithContext(ctx).Where(r.q.Service.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServiceRepository) Count(ctx context.Context) (int64, error) {
	return r.q.Service.WithContext(ctx).Count()
}

func (r *GenServiceRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.Service], error) {
	query := r.q.Service.WithContext(ctx).
		Preload(r.q.Service.Agent).
		Preload(r.q.Service.ServiceType).
		Preload(r.q.Service.Group)
	query = applyGenServiceAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenServiceFilters,
		applyGenServiceSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Service, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.Service]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServiceRepository) FindByAgentInstanceID(ctx context.Context, agentID properties.UUID, agentInstanceID string) (*domain.Service, error) {
	entity, err := r.q.Service.WithContext(ctx).
		Preload(r.q.Service.Agent).
		Preload(r.q.Service.ServiceType).
		Preload(r.q.Service.Group).
		Where(r.q.Service.AgentInstanceID.Eq(agentInstanceID)).
		Where(r.q.Service.AgentID.Eq(agentID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceRepository) CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error) {
	return r.q.Service.WithContext(ctx).Where(r.q.Service.GroupID.Eq(groupID)).Count()
}

func (r *GenServiceRepository) CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error) {
	return r.q.Service.WithContext(ctx).Where(r.q.Service.AgentID.Eq(agentID)).Count()
}

func (r *GenServiceRepository) CountByServiceType(ctx context.Context, serviceTypeID properties.UUID) (int64, error) {
	return r.q.Service.WithContext(ctx).Where(r.q.Service.ServiceTypeID.Eq(serviceTypeID)).Count()
}

func (r *GenServiceRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	entity, err := r.q.Service.WithContext(ctx).
		Select(r.q.Service.ProviderID, r.q.Service.AgentID, r.q.Service.ConsumerID).
		Where(r.q.Service.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &auth.DefaultObjectScope{
		ProviderID: &entity.ProviderID,
		AgentID:    &entity.AgentID,
		ConsumerID: &entity.ConsumerID,
	}, nil
}

func applyGenServiceAuthz(query IServiceDo, scope *auth.IdentityScope) IServiceDo {
	q := Use(nil).Service
	if scope.ParticipantID != nil {
		return query.Where(q.ConsumerID.Eq(*scope.ParticipantID)).
			Or(q.ProviderID.Eq(*scope.ParticipantID))
	}
	if scope.AgentID != nil {
		return query.Where(q.AgentID.Eq(*scope.AgentID))
	}
	return query
}

func applyGenServiceFilters(query IServiceDo, pageReq *domain.PageReq) IServiceDo {
	q := Use(nil).Service
	if values, ok := pageReq.Filters["name"]; ok && len(values) > 0 {
		query = query.Where(q.Name.In(values...))
	}
	if values, ok := pageReq.Filters["currentStatus"]; ok && len(values) > 0 {
		query = query.Where(q.Status.In(values...))
	}
	return query
}

func applyGenServiceSort(query IServiceDo, pageReq *domain.PageReq) IServiceDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).Service
	if pageReq.SortBy == "name" {
		if pageReq.SortAsc {
			return query.Order(q.Name)
		}
		return query.Order(q.Name.Desc())
	}
	return query
}

