package domain

import (
	"errors"
)

var (
	// ErrNotFound indicates that the requested entity was not found
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that the operation cannot be completed due to a conflict
	ErrConflict = errors.New("entity conflict")

	// ErrInvalidInput indicates that the input data is invalid
	ErrInvalidInput = errors.New("invalid input")
)

type Filters map[string]interface{}

type Sorting struct {
	SortField string // Field to sort by
	SortOrder string // "asc" or "desc"
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
