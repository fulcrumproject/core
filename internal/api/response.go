package api

import (
	"time"

	"fulcrumproject.org/core/internal/domain"
)

const (
	// ISO8601UTC is the standard time format used across the API
	ISO8601UTC = "2006-01-02T15:04:05Z07:00"
)

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

type JSONUTCTime time.Time

// MarshalJSON implements json.Marshaler interface
func (t JSONUTCTime) MarshalJSON() ([]byte, error) {
	formatted := time.Time(t).UTC().Format(ISO8601UTC)
	return []byte(`"` + formatted + `"`), nil
}
