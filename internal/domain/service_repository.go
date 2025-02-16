package domain

import "context"

// ServiceRepository defines the interface for the Service repository
type ServiceRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *Service) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Service) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Service, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filters Filters, sorting *Sorting, pagination *Pagination) (*PaginatedResult[Service], error)

	// Count returns the number of entities matching the provided filters
	Count(ctx context.Context, filters Filters) (int64, error)
}
