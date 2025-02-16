package domain

import "context"

// ServiceGroupRepository defines the interface for the ServiceGroup repository
type ServiceGroupRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *ServiceGroup) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *ServiceGroup) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceGroup, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filters map[string]interface{}) ([]ServiceGroup, error)

	// Count returns the number of entities matching the provided filters
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
}
