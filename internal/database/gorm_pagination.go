package database

import (
	"context"
	"fmt"

	"fulcrumproject.org/core/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PageApplier func(q *gorm.DB, r *domain.PageRequest) (*gorm.DB, error)

type FilterFieldApplier func(q *gorm.DB, vv []string) (*gorm.DB, error)

func mapFilterApplier(fields map[string]FilterFieldApplier) PageApplier {
	return func(q *gorm.DB, r *domain.PageRequest) (*gorm.DB, error) {
		if len(r.Filters) == 0 {
			return q, nil
		}

		var err error
		for field, values := range r.Filters {
			applier, exists := fields[field]
			if !exists {
				return q, fmt.Errorf("cannot filter by field %s", field)
			}
			if q, err = applier(q, values); err != nil {
				return nil, err
			}
		}
		return q, nil
	}
}

func mapSortApplier(fields map[string]string) PageApplier {
	return func(q *gorm.DB, r *domain.PageRequest) (*gorm.DB, error) {
		if !r.Sort {
			return q, nil
		}
		field, exists := fields[r.SortBy]
		if !exists {
			return q, fmt.Errorf("cannot sort by field %s", field)
		}
		return q.Order(clause.OrderByColumn{Column: clause.Column{Name: field}, Desc: !r.SortAsc}), nil
	}
}

func applyPagination(q *gorm.DB, r *domain.PageRequest) (*gorm.DB, error) {
	offset := (r.Page - 1) * r.PageSize
	q = q.Offset(offset).Limit(r.PageSize)
	return q, nil
}

func parserInFilterFieldApplier[T any](f string, t func(string) (T, error)) FilterFieldApplier {
	return func(q *gorm.DB, vv []string) (*gorm.DB, error) {
		if len(vv) == 0 {
			return q, nil
		}
		values := make([]T, len(vv))
		for _, v := range vv {
			value, err := t(v)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return q.Where(fmt.Sprintf("%s IN ?", f), values), nil
	}
}

func stringInFilterFieldApplier(f string) FilterFieldApplier {
	return parserInFilterFieldApplier(f, func(v string) (string, error) { return v, nil })
}

// list implements a generic list operation for any model type
func list[T any](
	ctx context.Context,
	db *gorm.DB,
	page *domain.PageRequest,
	filterApplier PageApplier,
	sortApplier PageApplier,
	preloadPaths []string,
) (*domain.PageResponse[T], error) {
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
