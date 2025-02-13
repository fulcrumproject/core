package domain

import "context"

// ServiceTypeRepository defines the interface for the ServiceType repository
type ServiceTypeRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *ServiceType) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filters map[string]interface{}) ([]ServiceType, error)
}
