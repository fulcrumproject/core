package api

import "fulcrumproject.org/core/internal/domain"

// NewPaginatedResponse creates a new PaginatedResponse from a domain.PaginatedResult
func NewPaginatedResponse[E any, R any](result *domain.PaginatedResult[E], conv func(*E) *R) *PaginatedResponse[R] {
	items := make([]*R, len(result.Items))
	for i, e := range result.Items {
		items[i] = conv(&e)
	}

	return &PaginatedResponse[R]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}
}

// PaginatedResponse represents a generic paginated response
type PaginatedResponse[T any] struct {
	Items       []*T  `json:"items"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int   `json:"totalPages"`
	CurrentPage int   `json:"currentPage"`
	HasNext     bool  `json:"hasNext"`
	HasPrev     bool  `json:"hasPrev"`
}
