package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// AgentState represents the possible states of an Agent
type AgentState string

const (
	AgentNew          AgentState = "New"
	AgentConnected    AgentState = "Connected"
	AgentDisconnected AgentState = "Disconnected"
	AgentError        AgentState = "Error"
	AgentDisabled     AgentState = "Disabled"
)

// Validate checks if the agent state is valid
func (s AgentState) Validate() error {
	switch s {
	case AgentNew, AgentConnected, AgentDisconnected, AgentError, AgentDisabled:
		return nil
	default:
		return fmt.Errorf("invalid agent state: %s", s)
	}
}

func ParseAgentState(value string) (AgentState, error) {
	state := AgentState(value)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Agent represents a service manager agent
type Agent struct {
	BaseEntity

	Name        string      `gorm:"not null"`
	Attributes  Attributes  `gorm:"type:jsonb"`
	CountryCode CountryCode `gorm:"size:2"`

	// State management
	State           AgentState `gorm:"not null"`
	LastStateUpdate time.Time  `gorm:"index"`

	// Relationships
	AgentTypeID UUID       `gorm:"not null"`
	AgentType   *AgentType `gorm:"foreignKey:AgentTypeID"`
	ProviderID  UUID       `gorm:"not null"`
	Provider    *Provider  `gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for the agent
func (Agent) TableName() string {
	return "agents"
}

// Validate ensures all agent fields are valid
func (a *Agent) Validate() error {
	if err := a.State.Validate(); err != nil {
		return err
	}
	return nil
}

// AgentCommander defines the interface for agent command operations
type AgentCommander interface {
	// Create creates a new agent
	Create(ctx context.Context, name string, countryCode CountryCode, attributes Attributes, providerID UUID, agentTypeID UUID) (*Agent, error)

	// Update updates an agent
	Update(ctx context.Context, id UUID, name *string, countryCode *CountryCode, attributes *Attributes, state *AgentState) (*Agent, error)

	// Delete removes an agent by ID after checking for dependencies
	Delete(ctx context.Context, id UUID) error

	// UpdateState updates the agent state and the related timestamp
	UpdateState(ctx context.Context, id UUID, state AgentState) (*Agent, error)
}

// agentCommander is the concrete implementation of AgentCommander
type agentCommander struct {
	store Store
}

// NewAgentCommander creates a new default AgentCommander
func NewAgentCommander(
	store Store,
) *agentCommander {
	return &agentCommander{
		store: store,
	}
}

func (s *agentCommander) Create(
	ctx context.Context,
	name string,
	countryCode CountryCode,
	attributes Attributes,
	providerID UUID,
	agentTypeID UUID,
) (*Agent, error) {
	if err := ValidateAuthScope(ctx, &AuthScope{ProviderID: &providerID}); err != nil {
		return nil, err
	}

	agent := &Agent{
		Name:        name,
		State:       AgentDisconnected,
		CountryCode: countryCode,
		Attributes:  attributes,
		ProviderID:  providerID,
		AgentTypeID: agentTypeID,
	}
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.AgentRepo().Create(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *agentCommander) Update(ctx context.Context,
	id UUID,
	name *string,
	countryCode *CountryCode,
	attributes *Attributes,
	state *AgentState,
) (*Agent, error) {
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &id, ProviderID: &agent.ProviderID}); err != nil {
		return nil, err
	}

	if name != nil {
		agent.Name = *name
	}
	if countryCode != nil {
		agent.CountryCode = *countryCode
	}
	if attributes != nil {
		agent.Attributes = *attributes
	}
	if state != nil {
		agent.State = *state
	}
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	err = s.store.AgentRepo().Save(ctx, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *agentCommander) Delete(ctx context.Context, id UUID) error {
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &id, ProviderID: &agent.ProviderID}); err != nil {
		return err
	}

	return s.store.Atomic(ctx, func(store Store) error {
		// Prevent deletion if it has services present
		numOfServices, err := store.ServiceRepo().CountByAgent(ctx, id)
		if err != nil {
			return err
		}
		if numOfServices > 0 {
			return errors.New("cannot delete agent with associated services")
		}

		// Delete all tokens associated with this agent before deleting the agent
		if err := store.TokenRepo().DeleteByAgentID(ctx, id); err != nil {
			return err
		}

		return store.AgentRepo().Delete(ctx, id)
	})
}

func (s *agentCommander) UpdateState(ctx context.Context, id UUID, state AgentState) (*Agent, error) {
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &id, ProviderID: &agent.ProviderID}); err != nil {
		return nil, err
	}

	agent.State = state
	agent.LastStateUpdate = time.Now()
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	err = s.store.AgentRepo().Save(ctx, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

type AgentRepository interface {
	AgentQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *Agent) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Agent) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// MarkInactiveAgentsAsDisconnected marks agents that haven't updated their status in the given duration as disconnected
	MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error)
}

type AgentQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Agent, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Agent], error)

	// CountByProvider returns the number of agents for a specific provider
	CountByProvider(ctx context.Context, providerID UUID) (int64, error)
}
