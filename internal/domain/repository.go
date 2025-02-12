package domain

import (
	"context"
	"errors"
)

var (
	// ErrNotFound indicates that the requested entity was not found
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that the operation cannot be completed due to a conflict
	ErrConflict = errors.New("entity conflict")

	// ErrInvalidInput indicates that the input data is invalid
	ErrInvalidInput = errors.New("invalid input")
)

// Repository defines the base interface for all repositories
type Repository[T any] interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *T) error

	// Update updates an existing entity
	Update(ctx context.Context, entity *T) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*T, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, filters map[string]interface{}) ([]T, error)
}

// ProviderRepository defines the interface for the Provider repository
type ProviderRepository interface {
	Repository[Provider]

	// FindByCountryCode retrieves providers by country code
	FindByCountryCode(ctx context.Context, code string) ([]Provider, error)

	// UpdateState updates the state of a provider
	UpdateState(ctx context.Context, id UUID, state ProviderState) error
}

// AgentRepository defines the interface for the Agent repository
type AgentRepository interface {
	Repository[Agent]

	// FindByProvider retrieves agents for a specific provider
	FindByProvider(ctx context.Context, providerID UUID) ([]Agent, error)

	// FindByAgentType retrieves agents for a specific type
	FindByAgentType(ctx context.Context, agentTypeID UUID) ([]Agent, error)

	// UpdateState updates the state of an agent
	UpdateState(ctx context.Context, id UUID, state AgentState) error
}

// AgentTypeRepository defines the interface for the AgentType repository
type AgentTypeRepository interface {
	Repository[AgentType]

	// FindByServiceType retrieves agent types that support a specific service type
	FindByServiceType(ctx context.Context, serviceTypeID UUID) ([]AgentType, error)

	// AddServiceType adds a service type to an agent type
	AddServiceType(ctx context.Context, agentTypeID, serviceTypeID UUID) error

	// RemoveServiceType removes a service type from an agent type
	RemoveServiceType(ctx context.Context, agentTypeID, serviceTypeID UUID) error
}

// ServiceTypeRepository defines the interface for the ServiceType repository
type ServiceTypeRepository interface {
	Repository[ServiceType]

	// FindByAgentType retrieves service types supported by an agent type
	FindByAgentType(ctx context.Context, agentTypeID UUID) ([]ServiceType, error)

	// UpdateResourceDefinitions updates the resource definitions of a service type
	UpdateResourceDefinitions(ctx context.Context, id UUID, definitions JSON) error
}
