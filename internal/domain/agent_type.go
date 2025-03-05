package domain

import "context"

// AgentType represents a type of service manager agent
type AgentType struct {
	BaseEntity
	Name         string        `gorm:"not null;unique"`
	ServiceTypes []ServiceType `gorm:"many2many:agent_type_service_types;"`
}

// TableName returns the table name for the agent type
func (*AgentType) TableName() string {
	return "agent_types"
}

// AgentTypeRepository defines the interface for the AgentType repository
type AgentTypeRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *AgentType) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*AgentType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[AgentType], error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)
}

// AgentTypeQuerier defines the interface for the AgentType read-only queries
type AgentTypeQuerier interface {

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*AgentType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[AgentType], error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)
}
