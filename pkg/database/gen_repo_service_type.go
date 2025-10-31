// ServiceType repository implementation using GORM Gen
// Provides type-safe database operations for ServiceType entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenServiceTypeRepository struct {
	q *Query
}

func NewGenServiceTypeRepository(db *gorm.DB) *GenServiceTypeRepository {
	return &GenServiceTypeRepository{
		q: Use(db),
	}
}

func (r *GenServiceTypeRepository) Create(ctx context.Context, entity *domain.ServiceType) error {
	return r.q.ServiceType.WithContext(ctx).Create(entity)
}

func (r *GenServiceTypeRepository) Save(ctx context.Context, entity *domain.ServiceType) error {
	result, err := r.q.ServiceType.WithContext(ctx).
		Where(r.q.ServiceType.ID.Eq(entity.ID)).
		Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceTypeRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServiceType.WithContext(ctx).
		Where(r.q.ServiceType.ID.Eq(id)).
		Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceTypeRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServiceType, error) {
	entity, err := r.q.ServiceType.WithContext(ctx).
		Where(r.q.ServiceType.ID.Eq(id)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceTypeRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServiceType.WithContext(ctx).
		Where(r.q.ServiceType.ID.Eq(id)).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GenServiceTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServiceType.WithContext(ctx).Count()
}

func (r *GenServiceTypeRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServiceType], error) {
	query := r.q.ServiceType.WithContext(ctx)

	result, err := PaginateQuery(
		ctx,
		query,
		pageReq,
		func(q IServiceTypeDo, pr *domain.PageReq) IServiceTypeDo {
			qt := Use(nil).ServiceType
			if values, ok := pr.Filters["name"]; ok && len(values) > 0 {
				q = q.Where(qt.Name.In(values...))
			}
			return q
		},
		func(q IServiceTypeDo, pr *domain.PageReq) IServiceTypeDo {
			if !pr.Sort {
				return q
			}
			qt := Use(nil).ServiceType
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

	items := make([]domain.ServiceType, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}

	return &domain.PageRes[domain.ServiceType]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}, nil
}

func (r *GenServiceTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return &auth.AllwaysMatchObjectScope{}, nil
}
