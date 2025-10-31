// Pagination helper for GORM Gen queries
// Provides generic pagination with filtering and sorting support
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/domain"
)

// PaginateQuery applies pagination, filtering, and sorting to a Gen query
func PaginateQuery[T any, Q interface {
	WithContext(context.Context) Q
	Limit(int) Q
	Offset(int) Q
	Count() (int64, error)
	Find() ([]*T, error)
}](
	ctx context.Context,
	query Q,
	pageReq *domain.PageReq,
	filterApplier func(Q, *domain.PageReq) Q,
	sortApplier func(Q, *domain.PageReq) Q,
) (*domain.PageRes[*T], error) {
	query = query.WithContext(ctx)

	if filterApplier != nil {
		query = filterApplier(query, pageReq)
	}

	count, err := query.Count()
	if err != nil {
		return nil, err
	}

	if sortApplier != nil {
		query = sortApplier(query, pageReq)
	}

	offset := (pageReq.Page - 1) * pageReq.PageSize
	query = query.Limit(pageReq.PageSize).Offset(offset)

	items, err := query.Find()
	if err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(items, count, pageReq), nil
}

