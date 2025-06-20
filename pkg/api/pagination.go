package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fulcrumproject/core/pkg/domain"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100

	// Request parameters
	paramPage     = "page"
	paramPageSize = "pageSize"
	paramSort     = "sort"
)

// Reserved parameters that should not be included in filters
var reservedParams = map[string]bool{
	paramPage:     true,
	paramPageSize: true,
	paramSort:     true,
}

func ParsePageRequest(r *http.Request) (*domain.PageRequest, error) {
	q := r.URL.Query()

	// Pagination - strict validation
	page := defaultPage
	if pageStr := q.Get(paramPage); pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, fmt.Errorf("invalid page parameter: %s", pageStr)
		}
		if parsedPage < 1 {
			return nil, fmt.Errorf("page parameter must be greater than 0, got: %d", parsedPage)
		}
		page = parsedPage
	}

	pageSize := defaultPageSize
	if pageSizeStr := q.Get(paramPageSize); pageSizeStr != "" {
		parsedPageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid pageSize parameter: %s", pageSizeStr)
		}
		if parsedPageSize < 1 {
			return nil, fmt.Errorf("pageSize parameter must be greater than 0, got: %d", parsedPageSize)
		}
		if parsedPageSize > maxPageSize {
			return nil, fmt.Errorf("pageSize parameter must not exceed %d, got: %d", maxPageSize, parsedPageSize)
		}
		pageSize = parsedPageSize
	}

	// Sort
	sort := q.Get(paramSort)
	var sortBy string
	var sortAsc bool
	if sort != "" {
		if strings.HasPrefix(sort, "+") {
			sortBy = sort[1:]
			sortAsc = true
		} else if strings.HasPrefix(sort, "-") {
			sortBy = sort[1:]
			sortAsc = false
		} else {
			sortBy = sort
			sortAsc = true // default to ascending if no prefix
		}
	}

	// Collect all non-reserved parameters as filters
	filters := make(map[string][]string)
	for key, values := range q {
		if !reservedParams[key] && len(values) > 0 {
			filters[key] = values
		}
	}

	return &domain.PageRequest{
		Page: page, PageSize: pageSize,
		Sort: sort != "", SortBy: sortBy, SortAsc: sortAsc,
		Filters: filters,
	}, nil
}

// PageResponse represents a generic paginated response
type PageResponse[T any] struct {
	Items       []*T  `json:"items"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int   `json:"totalPages"`
	CurrentPage int   `json:"currentPage"`
	HasNext     bool  `json:"hasNext"`
	HasPrev     bool  `json:"hasPrev"`
}

// NewPageResponse creates a new PaginatedResponse from a domain.PaginatedResult
func NewPageResponse[E any, R any](result *domain.PageResponse[E], conv func(*E) *R) *PageResponse[R] {
	items := make([]*R, len(result.Items))
	for i, e := range result.Items {
		items[i] = conv(&e)
	}

	return &PageResponse[R]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}
}
