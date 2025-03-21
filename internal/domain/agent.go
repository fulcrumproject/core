package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

	Name        string      `json:"name" gorm:"not null"`
	Attributes  Attributes  `json:"attributes,omitempty" gorm:"type:jsonb"`
	CountryCode CountryCode `json:"countryCode,omitempty" gorm:"size:2"`

	// State management
	State           AgentState `json:"state" gorm:"not null"`
	LastStateUpdate time.Time  `json:"lastStateUpdate" gorm:"index"`

	// Relationships
	AgentTypeID UUID       `json:"agentTypeId" gorm:"not null"`
	AgentType   *AgentType `json:"agentType,omitempty" gorm:"foreignKey:AgentTypeID"`
	ProviderID  UUID       `json:"providerId" gorm:"not null"`
	Provider    *Provider  `json:"-" gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for the agent
func (Agent) TableName() string {
	return "agents"
}

// Validate ensures all agent fields are valid
func (a *Agent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if err := a.State.Validate(); err != nil {
		return err
	}
	if a.LastStateUpdate.IsZero() {
		return fmt.Errorf("state last update cannot be empty")
	}
	if a.AgentTypeID == uuid.Nil {
		return fmt.Errorf("agent type ID cannot be empty")
	}
	if a.ProviderID == uuid.Nil {
		return fmt.Errorf("provider ID cannot be empty")
	}
	if err := a.CountryCode.Validate(); err != nil {
		return err
	}
	if a.Attributes != nil {
		if err := a.Attributes.Validate(); err != nil {
			return err
		}
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
	store          Store
	auditCommander AuditEntryCommander
}

// NewAgentCommander creates a new default AgentCommander
func NewAgentCommander(
	store Store,
	auditCommander AuditEntryCommander,
) *agentCommander {
	return &agentCommander{
		store:          store,
		auditCommander: auditCommander,
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
	providerExists, err := s.store.ProviderRepo().Exists(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if !providerExists {
		return nil, NewInvalidInputErrorf("provider with ID %s does not exist", providerID)
	}
	agentTypeExists, err := s.store.AgentTypeRepo().Exists(ctx, agentTypeID)
	if err != nil {
		return nil, err
	}
	if !agentTypeExists {
		return nil, NewInvalidInputErrorf("agent type with ID %s does not exist", agentTypeID)
	}

	var agent *Agent
	err = s.store.Atomic(ctx, func(store Store) error {
		agent = &Agent{
			Name:            name,
			State:           AgentDisconnected,
			LastStateUpdate: time.Now(),
			CountryCode:     countryCode,
			Attributes:      attributes,
			ProviderID:      providerID,
			AgentTypeID:     agentTypeID,
		}
		if err := agent.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.AgentRepo().Create(ctx, agent); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtx(
			ctx, EventTypeAgentCreated, JSON{"state": agent},
			&agent.ID, &providerID, nil, nil)
		return err
	})
	if err != nil {
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
	beforeAgent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy of the agent before modifications for audit diff
	beforeAgentCopy := *beforeAgent

	if name != nil {
		beforeAgent.Name = *name
	}
	if countryCode != nil {
		beforeAgent.CountryCode = *countryCode
	}
	if attributes != nil {
		beforeAgent.Attributes = *attributes
	}
	if state != nil {
		beforeAgent.State = *state
	}
	if err := beforeAgent.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.AgentRepo().Save(ctx, beforeAgent)
		if err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtxWithDiff(ctx, EventTypeAgentUpdated,
			&id, &beforeAgent.ProviderID, nil, nil, &beforeAgentCopy, beforeAgent)
		return err
	})
	if err != nil {
		return nil, err
	}
	return beforeAgent, nil
}

func (s *agentCommander) Delete(ctx context.Context, id UUID) error {
	// Get agent before deletion for audit purposes
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Store provider ID for audit entry
	providerID := agent.ProviderID

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

		if err := store.AgentRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtx(ctx, EventTypeAgentDeleted,
			JSON{"state": agent}, &id, &providerID, nil, nil)
		return err
	})
}

func (s *agentCommander) UpdateState(ctx context.Context, id UUID, state AgentState) (*Agent, error) {
	beforeAgent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy of the agent before modifications for audit diff
	beforeAgentCopy := *beforeAgent

	beforeAgent.State = state
	beforeAgent.LastStateUpdate = time.Now()
	if err := beforeAgent.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.AgentRepo().Save(ctx, beforeAgent)
		if err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtxWithDiff(ctx, EventTypeAgentUpdated,
			&id, &beforeAgent.ProviderID, nil, nil, &beforeAgentCopy, beforeAgent)
		return err
	})
	if err != nil {
		return nil, err
	}
	return beforeAgent, nil
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

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[Agent], error)

	// CountByProvider returns the number of agents for a specific provider
	CountByProvider(ctx context.Context, providerID UUID) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
