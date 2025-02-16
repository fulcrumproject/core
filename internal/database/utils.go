package database

import (
	"fmt"

	"fulcrumproject.org/core/internal/domain"
	"gorm.io/gorm"
)

func applyFilters(query *gorm.DB, filters domain.Filters) *gorm.DB {
	for key, value := range filters {
		query = query.Where(key, value)
	}
	return query
}

func applySorting(query *gorm.DB, sorting *domain.Sorting) *gorm.DB {
	if sorting != nil && sorting.SortField != "" {
		order := "asc"
		if sorting.SortOrder == "desc" {
			order = "desc"
		}
		query = query.Order(fmt.Sprintf("%s %s", sorting.SortField, order))
	}
	return query
}

func applyPagination(query *gorm.DB, pagination *domain.Pagination) *gorm.DB {
	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}
	return query
}
