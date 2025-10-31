// ServicePoolSet repository implementation using GORM Gen
// Provides type-safe database operations for ServicePoolSet entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type GenServicePoolSetRepository struct {
	q *Query
}

func NewGenServicePoolSetRepository(db *gorm.DB) *GenServicePoolSetRepository {
	return &GenServicePoolSetRepository{q: Use(db)}
}

func (r *GenServicePoolSetRepository) Create(ctx context.Context, entity *domain.ServicePoolSet) error {
	return r.q.ServicePoolSet.WithContext(ctx).Create(entity)
}

func (r *GenServicePoolSetRepository) Update(ctx context.Context, entity *domain.ServicePoolSet) error {
	result, err := r.q.ServicePoolSet.WithContext(ctx).Where(r.q.ServicePoolSet.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServicePoolSetRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServicePoolSet.WithContext(ctx).Where(r.q.ServicePoolSet.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServicePoolSetRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServicePoolSet, error) {
	entity, err := r.q.ServicePoolSet.WithContext(ctx).Where(r.q.ServicePoolSet.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServicePoolSetRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServicePoolSet.WithContext(ctx).Where(r.q.ServicePoolSet.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServicePoolSetRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServicePoolSet.WithContext(ctx).Count()
}

func (r *GenServicePoolSetRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServicePoolSet], error) {
	query := r.q.ServicePoolSet.WithContext(ctx)
	query = applyGenServicePoolSetAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenServicePoolSetFilters,
		applyGenServicePoolSetSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ServicePoolSet, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.ServicePoolSet]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServicePoolSetRepository) FindByProvider(ctx context.Context, providerID properties.UUID) ([]*domain.ServicePoolSet, error) {
	return r.q.ServicePoolSet.WithContext(ctx).
		Where(r.q.ServicePoolSet.ProviderID.Eq(providerID)).
		Order(r.q.ServicePoolSet.Name).
		Find()
}

func (r *GenServicePoolSetRepository) FindByProviderAndName(ctx context.Context, providerID properties.UUID, name string) (*domain.ServicePoolSet, error) {
	entity, err := r.q.ServicePoolSet.WithContext(ctx).
		Where(r.q.ServicePoolSet.ProviderID.Eq(providerID)).
		Where(r.q.ServicePoolSet.Name.Eq(name)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServicePoolSetRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	entity, err := r.q.ServicePoolSet.WithContext(ctx).
		Select(r.q.ServicePoolSet.ProviderID).
		Where(r.q.ServicePoolSet.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &auth.DefaultObjectScope{ProviderID: &entity.ProviderID}, nil
}

func applyGenServicePoolSetAuthz(query IServicePoolSetDo, scope *auth.IdentityScope) IServicePoolSetDo {
	q := Use(nil).ServicePoolSet
	if scope.ParticipantID != nil {
		return query.Where(q.ProviderID.Eq(*scope.ParticipantID))
	}
	if scope.AgentID != nil {
		qa := Use(nil).Agent
		providers, _ := qa.WithContext(context.Background()).Select(qa.ProviderID).Where(qa.ID.Eq(*scope.AgentID)).Find()
		if len(providers) > 0 {
			return query.Where(q.ProviderID.Eq(providers[0].ProviderID))
		}
	}
	return query
}

func applyGenServicePoolSetFilters(query IServicePoolSetDo, pageReq *domain.PageReq) IServicePoolSetDo {
	q := Use(nil).ServicePoolSet
	if values, ok := pageReq.Filters["providerId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
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
	if values, ok := pageReq.Filters["name"]; ok && len(values) > 0 {
		query = query.Where(q.Name.In(values...))
	}
	return query
}

func applyGenServicePoolSetSort(query IServicePoolSetDo, pageReq *domain.PageReq) IServicePoolSetDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).ServicePoolSet
	switch pageReq.SortBy {
	case "name":
		if pageReq.SortAsc {
			return query.Order(q.Name)
		}
		return query.Order(q.Name.Desc())
	case "createdAt":
		if pageReq.SortAsc {
			return query.Order(q.CreatedAt)
		}
		return query.Order(q.CreatedAt.Desc())
	}
	return query
}

