// MetricType repository implementation using GORM Gen
// Provides type-safe database operations for MetricType entities
package database

import (
	"context"
	
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenMetricTypeRepository struct {
	q *Query
}

func NewGenMetricTypeRepository(db *gorm.DB) *GenMetricTypeRepository {
	return &GenMetricTypeRepository{q: Use(db)}
}

func (r *GenMetricTypeRepository) Create(ctx context.Context, entity *domain.MetricType) error {
	return r.q.MetricType.WithContext(ctx).Create(entity)
}

func (r *GenMetricTypeRepository) Save(ctx context.Context, entity *domain.MetricType) error {
	result, err := r.q.MetricType.WithContext(ctx).Where(r.q.MetricType.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenMetricTypeRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.MetricType.WithContext(ctx).Where(r.q.MetricType.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenMetricTypeRepository) Get(ctx context.Context, id properties.UUID) (*domain.MetricType, error) {
	entity, err := r.q.MetricType.WithContext(ctx).Where(r.q.MetricType.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenMetricTypeRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.MetricType.WithContext(ctx).Where(r.q.MetricType.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenMetricTypeRepository) Count(ctx context.Context) (int64, error) {
	return r.q.MetricType.WithContext(ctx).Count()
}

func (r *GenMetricTypeRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.MetricType], error) {
	result, err := PaginateQuery[domain.MetricType, IMetricTypeDo](ctx, r.q.MetricType.WithContext(ctx), pageReq,
		func(q IMetricTypeDo, pr *domain.PageReq) IMetricTypeDo {
			qt := Use(nil).MetricType
			if values, ok := pr.Filters["name"]; ok && len(values) > 0 {
				q = q.Where(qt.Name.In(values...))
			}
			return q
		},
		func(q IMetricTypeDo, pr *domain.PageReq) IMetricTypeDo {
			if !pr.Sort {
				return q
			}
			qt := Use(nil).MetricType
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
	items := make([]domain.MetricType, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.MetricType]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenMetricTypeRepository) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	entity, err := r.q.MetricType.WithContext(ctx).Where(r.q.MetricType.Name.Eq(name)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenMetricTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return &auth.AllwaysMatchObjectScope{}, nil
}

