package api

import (
	"net/http"
	"strconv"

	"fulcrumproject.org/core/internal/domain"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
)

// ParsePagination extracts and validates pagination parameters from the request
func ParsePagination(r *http.Request) *domain.Pagination {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = defaultPage
	}

	pageSize, _ := strconv.Atoi(query.Get("pageSize"))
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	return &domain.Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// ParseSorting extracts and validates sorting parameters from the request
func ParseSorting(r *http.Request) *domain.Sorting {
	query := r.URL.Query()

	sortField := query.Get("sortField")
	if sortField == "" {
		return nil
	}

	return &domain.Sorting{
		SortField: sortField,
		SortOrder: query.Get("sortOrder"),
	}
}

// FilterConfig defines how a field should be filtered
type FilterConfig struct {
	Param  string
	Query  string
	Valuer func(string) interface{}
}

// ParseFilters extracts filters from the request based on provided field configurations
func ParseFilters(r *http.Request, configs []FilterConfig) domain.Filters {
	query := r.URL.Query()
	filters := make(domain.Filters)

	for _, config := range configs {
		if paramValue := query.Get(config.Param); paramValue != "" {
			query := config.Param
			if config.Query != "" {
				query = config.Query
			}
			var value interface{} = paramValue
			if config.Valuer != nil {
				value = config.Valuer(paramValue)
			}
			filters[query] = value
		}
	}

	return filters
}
