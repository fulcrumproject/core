package domain

import "context"

// AgentTypeRepository defines the interface for the AgentType repository
type AgentTypeRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *AgentType) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*AgentType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filters map[string]interface{}) ([]AgentType, error)

	// Count returns the number of entities matching the provided filters
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
}
