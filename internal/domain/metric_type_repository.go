package domain

import "context"

// MetricTypeRepository defines the interface for the MetricType repository
type MetricTypeRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *MetricType) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *MetricType) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*MetricType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[MetricType], error)
}
