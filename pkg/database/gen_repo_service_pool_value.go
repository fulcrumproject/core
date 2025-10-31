// ServicePoolValue repository implementation using GORM Gen
// Provides type-safe database operations for ServicePoolValue entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type GenServicePoolValueRepository struct {
	q *Query
}

func NewGenServicePoolValueRepository(db *gorm.DB) *GenServicePoolValueRepository {
	return &GenServicePoolValueRepository{q: Use(db)}
}

func (r *GenServicePoolValueRepository) Create(ctx context.Context, entity *domain.ServicePoolValue) error {
	return r.q.ServicePoolValue.WithContext(ctx).Create(entity)
}

func (r *GenServicePoolValueRepository) Update(ctx context.Context, entity *domain.ServicePoolValue) error {
	result, err := r.q.ServicePoolValue.WithContext(ctx).Where(r.q.ServicePoolValue.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServicePoolValueRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServicePoolValue.WithContext(ctx).Where(r.q.ServicePoolValue.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServicePoolValueRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServicePoolValue, error) {
	entity, err := r.q.ServicePoolValue.WithContext(ctx).Where(r.q.ServicePoolValue.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServicePoolValueRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServicePoolValue.WithContext(ctx).Where(r.q.ServicePoolValue.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServicePoolValueRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServicePoolValue.WithContext(ctx).Count()
}

func (r *GenServicePoolValueRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServicePoolValue], error) {
	query := r.q.ServicePoolValue.WithContext(ctx)
	query = applyGenServicePoolValueAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenServicePoolValueFilters,
		applyGenServicePoolValueSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ServicePoolValue, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.ServicePoolValue]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServicePoolValueRepository) ListByPool(ctx context.Context, poolID properties.UUID) ([]*domain.ServicePoolValue, error) {
	return r.q.ServicePoolValue.WithContext(ctx).
		Where(r.q.ServicePoolValue.ServicePoolID.Eq(poolID)).
		Order(r.q.ServicePoolValue.Name).
		Find()
}

func (r *GenServicePoolValueRepository) ListByService(ctx context.Context, serviceID properties.UUID) ([]*domain.ServicePoolValue, error) {
	return r.q.ServicePoolValue.WithContext(ctx).
		Where(r.q.ServicePoolValue.ServiceID.Eq(serviceID)).
		Find()
}

func (r *GenServicePoolValueRepository) FindByPool(ctx context.Context, poolID properties.UUID) ([]*domain.ServicePoolValue, error) {
	return r.q.ServicePoolValue.WithContext(ctx).
		Where(r.q.ServicePoolValue.ServicePoolID.Eq(poolID)).
		Find()
}

func (r *GenServicePoolValueRepository) FindAvailable(ctx context.Context, poolID properties.UUID) ([]*domain.ServicePoolValue, error) {
	q := r.q.ServicePoolValue
	return q.WithContext(ctx).
		Where(q.ServicePoolID.Eq(poolID)).
		Where(q.ServiceID.IsNull()).
		Order(q.Name).
		Find()
}

func (r *GenServicePoolValueRepository) FindByService(ctx context.Context, serviceID properties.UUID) ([]*domain.ServicePoolValue, error) {
	return r.q.ServicePoolValue.WithContext(ctx).
		Where(r.q.ServicePoolValue.ServiceID.Eq(serviceID)).
		Find()
}

func (r *GenServicePoolValueRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	value, err := r.q.ServicePoolValue.WithContext(ctx).
		Select(r.q.ServicePoolValue.ServicePoolID).
		Where(r.q.ServicePoolValue.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	pool, err := r.q.ServicePool.WithContext(ctx).
		Select(r.q.ServicePool.ServicePoolSetID).
		Where(r.q.ServicePool.ID.Eq(value.ServicePoolID)).First()
	if err != nil {
		return nil, err
	}
	poolSet, err := r.q.ServicePoolSet.WithContext(ctx).
		Select(r.q.ServicePoolSet.ProviderID).
		Where(r.q.ServicePoolSet.ID.Eq(pool.ServicePoolSetID)).First()
	if err != nil {
		return nil, err
	}
	return &auth.DefaultObjectScope{ProviderID: &poolSet.ProviderID}, nil
}

func applyGenServicePoolValueAuthz(query IServicePoolValueDo, scope *auth.IdentityScope) IServicePoolValueDo {
	q := Use(nil).ServicePoolValue
	qp := Use(nil).ServicePool
	qps := Use(nil).ServicePoolSet

	if scope.ParticipantID != nil {
		poolSetIDs, _ := qps.WithContext(context.Background()).
			Select(qps.ID).
			Where(qps.ProviderID.Eq(*scope.ParticipantID)).
			Find()
		if len(poolSetIDs) > 0 {
			setConditions := make([]gen.Condition, len(poolSetIDs))
			for i, ps := range poolSetIDs {
				setConditions[i] = qp.ServicePoolSetID.Eq(ps.ID)
			}
			poolQuery := qp.WithContext(context.Background()).Select(qp.ID).Where(setConditions[0])
			if len(setConditions) > 1 {
				poolQuery = poolQuery.Or(setConditions[1:]...)
			}
			poolIDs, _ := poolQuery.Find()
			if len(poolIDs) > 0 {
				conditions := make([]gen.Condition, len(poolIDs))
				for i, pool := range poolIDs {
					conditions[i] = q.ServicePoolID.Eq(pool.ID)
				}
				query = query.Where(conditions[0])
				if len(conditions) > 1 {
					query = query.Or(conditions[1:]...)
				}
			}
		}
	}
	if scope.AgentID != nil {
		qa := Use(nil).Agent
		providers, _ := qa.WithContext(context.Background()).Select(qa.ProviderID).Where(qa.ID.Eq(*scope.AgentID)).Find()
		if len(providers) > 0 {
			poolSetIDs, _ := qps.WithContext(context.Background()).
				Select(qps.ID).
				Where(qps.ProviderID.Eq(providers[0].ProviderID)).
				Find()
			if len(poolSetIDs) > 0 {
				setConditions := make([]gen.Condition, len(poolSetIDs))
				for i, ps := range poolSetIDs {
					setConditions[i] = qp.ServicePoolSetID.Eq(ps.ID)
				}
				poolQuery := qp.WithContext(context.Background()).Select(qp.ID).Where(setConditions[0])
				if len(setConditions) > 1 {
					poolQuery = poolQuery.Or(setConditions[1:]...)
				}
				poolIDs, _ := poolQuery.Find()
				if len(poolIDs) > 0 {
					conditions := make([]gen.Condition, len(poolIDs))
					for i, pool := range poolIDs {
						conditions[i] = q.ServicePoolID.Eq(pool.ID)
					}
					query = query.Where(conditions[0])
					if len(conditions) > 1 {
						query = query.Or(conditions[1:]...)
					}
				}
			}
		}
	}
	return query
}

func applyGenServicePoolValueFilters(query IServicePoolValueDo, pageReq *domain.PageReq) IServicePoolValueDo {
	q := Use(nil).ServicePoolValue
	if values, ok := pageReq.Filters["servicePoolId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ServicePoolID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	if values, ok := pageReq.Filters["serviceId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ServiceID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	return query
}

func applyGenServicePoolValueSort(query IServicePoolValueDo, pageReq *domain.PageReq) IServicePoolValueDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).ServicePoolValue
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

