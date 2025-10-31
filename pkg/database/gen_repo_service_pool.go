// ServicePool repository implementation using GORM Gen
// Provides type-safe database operations for ServicePool entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type GenServicePoolRepository struct {
	q *Query
}

func NewGenServicePoolRepository(db *gorm.DB) *GenServicePoolRepository {
	return &GenServicePoolRepository{q: Use(db)}
}

func (r *GenServicePoolRepository) Create(ctx context.Context, entity *domain.ServicePool) error {
	return r.q.ServicePool.WithContext(ctx).Create(entity)
}

func (r *GenServicePoolRepository) Update(ctx context.Context, entity *domain.ServicePool) error {
	result, err := r.q.ServicePool.WithContext(ctx).Where(r.q.ServicePool.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServicePoolRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServicePool.WithContext(ctx).Where(r.q.ServicePool.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServicePoolRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServicePool, error) {
	entity, err := r.q.ServicePool.WithContext(ctx).Where(r.q.ServicePool.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServicePoolRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServicePool.WithContext(ctx).Where(r.q.ServicePool.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServicePoolRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServicePool.WithContext(ctx).Count()
}

func (r *GenServicePoolRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServicePool], error) {
	query := r.q.ServicePool.WithContext(ctx)
	query = applyGenServicePoolAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenServicePoolFilters,
		applyGenServicePoolSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ServicePool, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.ServicePool]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServicePoolRepository) ListByPoolSet(ctx context.Context, poolSetID properties.UUID) ([]*domain.ServicePool, error) {
	return r.q.ServicePool.WithContext(ctx).
		Where(r.q.ServicePool.ServicePoolSetID.Eq(poolSetID)).
		Order(r.q.ServicePool.Name).
		Find()
}

func (r *GenServicePoolRepository) FindByPoolSetAndType(ctx context.Context, poolSetID properties.UUID, poolType string) (*domain.ServicePool, error) {
	entity, err := r.q.ServicePool.WithContext(ctx).
		Where(r.q.ServicePool.ServicePoolSetID.Eq(poolSetID)).
		Where(r.q.ServicePool.Type.Eq(poolType)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServicePoolRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	pool, err := r.q.ServicePool.WithContext(ctx).
		Select(r.q.ServicePool.ServicePoolSetID).
		Where(r.q.ServicePool.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
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

func applyGenServicePoolAuthz(query IServicePoolDo, scope *auth.IdentityScope) IServicePoolDo {
	q := Use(nil).ServicePool
	qps := Use(nil).ServicePoolSet
	if scope.ParticipantID != nil {
		poolSetIDs, _ := qps.WithContext(context.Background()).
			Select(qps.ID).
			Where(qps.ProviderID.Eq(*scope.ParticipantID)).
			Find()
		if len(poolSetIDs) > 0 {
			ids := make([]properties.UUID, len(poolSetIDs))
			for i, ps := range poolSetIDs {
				ids[i] = ps.ID
			}
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ServicePoolSetID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
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
				ids := make([]properties.UUID, len(poolSetIDs))
				for i, ps := range poolSetIDs {
					ids[i] = ps.ID
				}
				conditions := make([]gen.Condition, len(ids))
				for i, id := range ids {
					conditions[i] = q.ServicePoolSetID.Eq(id)
				}
				query = query.Where(conditions[0])
				if len(conditions) > 1 {
					query = query.Or(conditions[1:]...)
				}
			}
		}
	}
	return query
}

func applyGenServicePoolFilters(query IServicePoolDo, pageReq *domain.PageReq) IServicePoolDo {
	q := Use(nil).ServicePool
	if values, ok := pageReq.Filters["servicePoolSetId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ServicePoolSetID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	if values, ok := pageReq.Filters["type"]; ok && len(values) > 0 {
		query = query.Where(q.Type.In(values...))
	}
	if values, ok := pageReq.Filters["generatorType"]; ok && len(values) > 0 {
		query = query.Where(q.GeneratorType.In(values...))
	}
	return query
}

func applyGenServicePoolSort(query IServicePoolDo, pageReq *domain.PageReq) IServicePoolDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).ServicePool
	switch pageReq.SortBy {
	case "name":
		if pageReq.SortAsc {
			return query.Order(q.Name)
		}
		return query.Order(q.Name.Desc())
	case "type":
		if pageReq.SortAsc {
			return query.Order(q.Type)
		}
		return query.Order(q.Type.Desc())
	case "createdAt":
		if pageReq.SortAsc {
			return query.Order(q.CreatedAt)
		}
		return query.Order(q.CreatedAt.Desc())
	}
	return query
}

