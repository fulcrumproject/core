package database

import (
	"errors"
	"fmt"

	"fulcrumproject.org/core/internal/domain"
	"gorm.io/gorm"
)

type FilterValuer func(v string) (interface{}, error)

type FilterConfig struct {
	Query  string
	Valuer FilterValuer
}

func applySimpleFilter(query *gorm.DB, filter *domain.SimpleFilter, configs map[string]FilterConfig) (*gorm.DB, error) {
	if filter == nil {
		return query, nil
	}
	config, exists := configs[filter.Field]
	if !exists {
		return query, domain.NewInvalidInputError(filter.Field, errors.New("invalid filter field"))
	}
	where := filter.Field
	if config.Query != "" {
		where = config.Query
	}
	var (
		value interface{} = filter.Value
		err   error
	)
	if config.Valuer != nil {
		value, err = config.Valuer(filter.Value)
		if err != nil {
			return query, domain.NewInvalidInputError(filter.Field, err)
		}
	}
	return query.Where(where, value), nil
}

type SortingConfig struct {
	Query string
	Value func(value string) interface{}
}

func applySorting(query *gorm.DB, sorting *domain.Sorting) (*gorm.DB, error) {
	if sorting == nil || sorting.Field == "" {
		return query, nil
	}
	order := "asc"
	if sorting.Order == "desc" {
		order = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sorting.Field, order))
	return query, nil
}

func applyPagination(query *gorm.DB, pagination *domain.Pagination) (*gorm.DB, error) {
	if pagination == nil {
		return query, nil
	}
	offset := (pagination.Page - 1) * pagination.PageSize
	query = query.Offset(offset).Limit(pagination.PageSize)
	return query, nil
}
