package database

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PageFilterApplier func(db *gorm.DB, r *domain.PageReq) (*gorm.DB, error)

type FilterFieldApplier func(db *gorm.DB, vv []string) (*gorm.DB, error)

func MapFilterApplier(fields map[string]FilterFieldApplier) PageFilterApplier {
	return func(db *gorm.DB, r *domain.PageReq) (*gorm.DB, error) {
		if len(r.Filters) == 0 {
			return db, nil
		}

		var err error
		for field, values := range r.Filters {
			applier, exists := fields[field]
			if !exists {
				return db, fmt.Errorf("cannot filter by field %s", field)
			}
			if db, err = applier(db, values); err != nil {
				return nil, err
			}
		}
		return db, nil
	}
}

func MapSortApplier(fields map[string]string) PageFilterApplier {
	return func(db *gorm.DB, r *domain.PageReq) (*gorm.DB, error) {
		if !r.Sort {
			return db, nil
		}
		field, exists := fields[r.SortBy]
		if !exists {
			return db, fmt.Errorf("cannot sort by field %s", field)
		}
		return db.Order(clause.OrderByColumn{Column: clause.Column{Name: field}, Desc: !r.SortAsc}), nil
	}
}

func ParserInFilterFieldApplier[T any](f string, t func(string) (T, error)) FilterFieldApplier {
	return func(db *gorm.DB, vv []string) (*gorm.DB, error) {
		if len(vv) == 0 {
			return db, nil
		}
		values := make([]T, len(vv))
		for _, v := range vv {
			value, err := t(v)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return db.Where(fmt.Sprintf("%s IN ?", f), values), nil
	}
}

func StringInFilterFieldApplier(f string) FilterFieldApplier {
	return ParserInFilterFieldApplier(f, func(v string) (string, error) { return v, nil })
}

// listPaginated implements a generic listPaginated operation for any model type
func listPaginated[T any](
	ctx context.Context,
	db *gorm.DB,
	page *domain.PageReq,
	filterApplier PageFilterApplier,
	sortApplier PageFilterApplier,
	authzFilterApplier AuthzFilterApplier,
	preloadPaths []string,
	authIdentityScope *auth.IdentityScope,
) (*domain.PageRes[T], error) {
	var items []T

	// Start the query with the model type
	q := db.WithContext(ctx).Model(new(T))

	// Apply filters if a filter applier is provided
	if filterApplier != nil {
		var err error
		if q, err = filterApplier(q, page); err != nil {
			return nil, err
		}
	}
	if authzFilterApplier != nil && authIdentityScope != nil {
		q = authzFilterApplier(authIdentityScope, q)
	}

	// Get total count
	var count int64
	q = q.Count(&count)
	if q.Error != nil {
		return nil, q.Error
	}

	// Apply sorting if a sort applier is provided
	if sortApplier != nil {
		var err error
		if q, err = sortApplier(q, page); err != nil {
			return nil, err
		}
	}

	// Apply pagination
	var err error
	if q, err = applyPagination(q, page); err != nil {
		return nil, err
	}

	// Apply preloads
	for _, path := range preloadPaths {
		q = q.Preload(path)
	}

	// Execute the query
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(items, count, page), nil
}

func applyPagination(db *gorm.DB, r *domain.PageReq) (*gorm.DB, error) {
	offset := (r.Page - 1) * r.PageSize
	db = db.Offset(offset).Limit(r.PageSize)
	return db, nil
}
