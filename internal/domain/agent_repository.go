package domain

import "context"

// AgentRepository defines the interface for the Agent repository
type AgentRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *Agent) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Agent) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Agent, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filters Filters, sorting *Sorting, pagination *Pagination) (*PaginatedResult[Agent], error)

	// Count returns the number of entities matching the provided filters
	Count(ctx context.Context, filters Filters) (int64, error)
}
