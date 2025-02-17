package domain

type SimpleFilter struct {
	Field string // Field to filter
	Value string // Value to filter by
}

type Sorting struct {
	Field string // Field to sort by
	Order string // "asc" or "desc"
}

type Pagination struct {
	Page     int // Current page number
	PageSize int // Number of items per page
}

type PaginatedResult[T any] struct {
	Items       []T
	TotalItems  int64
	TotalPages  int
	CurrentPage int
	HasNext     bool
	HasPrev     bool
}

// NewPaginatedResult creates a new PaginatedResult with calculated pagination fields
func NewPaginatedResult[T any](items []T, totalItems int64, pagination *Pagination) *PaginatedResult[T] {
	totalPages := int(totalItems) / pagination.PageSize
	if int(totalItems)%pagination.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult[T]{
		Items:       items,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: pagination.Page,
		HasNext:     pagination.Page < totalPages,
		HasPrev:     pagination.Page > 1,
	}
}
