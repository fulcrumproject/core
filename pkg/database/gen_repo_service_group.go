// ServiceGroup repository implementation using GORM Gen
// Provides type-safe database operations for ServiceGroup entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenServiceGroupRepository struct {
	q *Query
}

func NewGenServiceGroupRepository(db *gorm.DB) *GenServiceGroupRepository {
	return &GenServiceGroupRepository{q: Use(db)}
}

func (r *GenServiceGroupRepository) Create(ctx context.Context, entity *domain.ServiceGroup) error {
	return r.q.ServiceGroup.WithContext(ctx).Create(entity)
}

func (r *GenServiceGroupRepository) Save(ctx context.Context, entity *domain.ServiceGroup) error {
	result, err := r.q.ServiceGroup.WithContext(ctx).Where(r.q.ServiceGroup.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceGroupRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServiceGroup.WithContext(ctx).Where(r.q.ServiceGroup.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceGroupRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServiceGroup, error) {
	entity, err := r.q.ServiceGroup.WithContext(ctx).
		Preload(r.q.ServiceGroup.Participant).
		Where(r.q.ServiceGroup.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceGroupRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServiceGroup.WithContext(ctx).Where(r.q.ServiceGroup.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServiceGroupRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServiceGroup.WithContext(ctx).Count()
}

func (r *GenServiceGroupRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServiceGroup], error) {
	query := r.q.ServiceGroup.WithContext(ctx).Preload(r.q.ServiceGroup.Participant)
	query = applyGenServiceGroupAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		func(q IServiceGroupDo, pr *domain.PageReq) IServiceGroupDo {
			qt := Use(nil).ServiceGroup
			if values, ok := pr.Filters["name"]; ok && len(values) > 0 {
				q = q.Where(qt.Name.In(values...))
			}
			return q
		},
		func(q IServiceGroupDo, pr *domain.PageReq) IServiceGroupDo {
			if !pr.Sort {
				return q
			}
			qt := Use(nil).ServiceGroup
			if pr.SortBy == "name" {
				if pr.SortAsc {
					return q.Order(qt.Name)
				}
				return q.Order(qt.Name.Desc())
			}
			return q
		},
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ServiceGroup, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.ServiceGroup]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServiceGroupRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	entity, err := r.q.ServiceGroup.WithContext(ctx).
		Select(r.q.ServiceGroup.ConsumerID).
		Where(r.q.ServiceGroup.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &auth.DefaultObjectScope{ConsumerID: &entity.ConsumerID}, nil
}

func applyGenServiceGroupAuthz(query IServiceGroupDo, scope *auth.IdentityScope) IServiceGroupDo {
	q := Use(nil).ServiceGroup
	if scope.ParticipantID != nil {
		return query.Where(q.ConsumerID.Eq(*scope.ParticipantID))
	}
	return query
}
