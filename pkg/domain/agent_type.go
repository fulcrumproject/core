package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

// AgentType represents a type of service manager agent
type AgentType struct {
	BaseEntity
	Name         string        `json:"name" gorm:"not null;unique"`
	ServiceTypes []ServiceType `json:"-" gorm:"many2many:agent_type_service_types;"`
}

// TableName returns the table name for the agent type
func (AgentType) TableName() string {
	return "agent_types"
}

// AgentTypeRepository defines the interface for the AgentType repository
type AgentTypeRepository interface {
	AgentTypeQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *AgentType) error
}

// AgentTypeQuerier defines the interface for the AgentType read-only queries
type AgentTypeQuerier interface {

	// Get retrieves an entity by ID
	Get(ctx context.Context, id properties.UUID) (*AgentType, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id properties.UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *auth.IdentityScope, req *PageRequest) (*PageResponse[AgentType], error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}
