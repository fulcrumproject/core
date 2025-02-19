package domain

type PageRequest struct {
	Filters  map[string][]string // Filters to be applied
	Sort     bool                // Should sort
	SortBy   string              // Field to sort by
	SortAsc  bool                // Sort dir
	Page     int                 // Current page number
	PageSize int                 // Number of items per page
}

type PageResponse[T any] struct {
	Items       []T
	TotalItems  int64
	TotalPages  int
	CurrentPage int
	HasNext     bool
	HasPrev     bool
}

// NewPaginatedResult creates a new PaginatedResult with calculated pagination fields
func NewPaginatedResult[T any](items []T, totalItems int64, page *PageRequest) *PageResponse[T] {
	totalPages := int(totalItems) / page.PageSize
	if int(totalItems)%page.PageSize > 0 {
		totalPages++
	}

	return &PageResponse[T]{
		Items:       items,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: page.Page,
		HasNext:     page.Page < totalPages,
		HasPrev:     page.Page > 1,
	}
}
