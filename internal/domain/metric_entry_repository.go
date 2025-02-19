package domain

import "context"

// MetricEntryRepository defines the interface for the MetricEntry repository
type MetricEntryRepository interface {
	// Create creates a new metric entry
	Create(ctx context.Context, entity *MetricEntry) error

	// List retrieves a list of metric entries based on the provided filters
	List(ctx context.Context, filter *SimpleFilter, sorting *Sorting, pagination *Pagination) (*PaginatedResult[MetricEntry], error)
}
