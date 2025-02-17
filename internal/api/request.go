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

// parsePagination extracts and validates pagination parameters from the request
func parsePagination(r *http.Request) *domain.Pagination {
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

// parseSorting extracts and validates sorting parameters from the request
func parseSorting(r *http.Request) *domain.Sorting {
	query := r.URL.Query()

	sortField := query.Get("sortField")
	if sortField == "" {
		return nil
	}

	return &domain.Sorting{
		Field: sortField,
		Order: query.Get("sortOrder"),
	}
}

// ParseFilters extracts filters from the request based on provided field configurations
func parseSimpleFilter(r *http.Request) *domain.SimpleFilter {
	query := r.URL.Query()
	if query.Has("filterField") && query.Has("filterValue") {
		if paramValue := query.Get("filterField"); paramValue != "" {
			return &domain.SimpleFilter{
				Field: paramValue,
				Value: query.Get("filterValue"),
			}
		}
	}
	return nil
}
