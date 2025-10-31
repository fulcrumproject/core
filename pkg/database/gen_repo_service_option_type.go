// ServiceOptionType repository implementation using GORM Gen
// Provides type-safe database operations for ServiceOptionType entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenServiceOptionTypeRepository struct {
	q *Query
}

func NewGenServiceOptionTypeRepository(db *gorm.DB) *GenServiceOptionTypeRepository {
	return &GenServiceOptionTypeRepository{q: Use(db)}
}

func (r *GenServiceOptionTypeRepository) Create(ctx context.Context, entity *domain.ServiceOptionType) error {
	return r.q.ServiceOptionType.WithContext(ctx).Create(entity)
}

func (r *GenServiceOptionTypeRepository) Save(ctx context.Context, entity *domain.ServiceOptionType) error {
	result, err := r.q.ServiceOptionType.WithContext(ctx).Where(r.q.ServiceOptionType.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceOptionTypeRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServiceOptionType.WithContext(ctx).Where(r.q.ServiceOptionType.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceOptionTypeRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServiceOptionType, error) {
	entity, err := r.q.ServiceOptionType.WithContext(ctx).Where(r.q.ServiceOptionType.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceOptionTypeRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServiceOptionType.WithContext(ctx).Where(r.q.ServiceOptionType.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServiceOptionTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServiceOptionType.WithContext(ctx).Count()
}

func (r *GenServiceOptionTypeRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServiceOptionType], error) {
	result, err := PaginateQuery(ctx, r.q.ServiceOptionType.WithContext(ctx), pageReq,
		func(q IServiceOptionTypeDo, pr *domain.PageReq) IServiceOptionTypeDo {
			qt := Use(nil).ServiceOptionType
			if values, ok := pr.Filters["name"]; ok && len(values) > 0 {
				q = q.Where(qt.Name.In(values...))
			}
			if values, ok := pr.Filters["type"]; ok && len(values) > 0 {
				q = q.Where(qt.Type.In(values...))
			}
			return q
		},
		func(q IServiceOptionTypeDo, pr *domain.PageReq) IServiceOptionTypeDo {
			if !pr.Sort {
				return q
			}
			qt := Use(nil).ServiceOptionType
			switch pr.SortBy {
			case "name":
				if pr.SortAsc {
					return q.Order(qt.Name)
				}
				return q.Order(qt.Name.Desc())
			case "type":
				if pr.SortAsc {
					return q.Order(qt.Type)
				}
				return q.Order(qt.Type.Desc())
			}
			return q
		},
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ServiceOptionType, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.ServiceOptionType]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServiceOptionTypeRepository) FindByType(ctx context.Context, typeStr string) (*domain.ServiceOptionType, error) {
	entity, err := r.q.ServiceOptionType.WithContext(ctx).Where(r.q.ServiceOptionType.Type.Eq(typeStr)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceOptionTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return &auth.AllwaysMatchObjectScope{}, nil
}
