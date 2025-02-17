package domain

import "context"

// ServiceTypeRepository defines the interface for the ServiceType repository
type ServiceTypeRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *ServiceType) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filter *SimpleFilter, sorting *Sorting, pagination *Pagination) (*PaginatedResult[ServiceType], error)

	// Count returns the number of entities matching the provided filters
	Count(ctx context.Context, filter *SimpleFilter) (int64, error)
}
