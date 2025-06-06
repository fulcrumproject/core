package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AgentStatus represents the possible statuss of an Agent
type AgentStatus string

const (
	AgentNew          AgentStatus = "New"
	AgentConnected    AgentStatus = "Connected"
	AgentDisconnected AgentStatus = "Disconnected"
	AgentError        AgentStatus = "Error"
	AgentDisabled     AgentStatus = "Disabled"
)

// Validate checks if the agent status is valid
func (s AgentStatus) Validate() error {
	switch s {
	case AgentNew, AgentConnected, AgentDisconnected, AgentError, AgentDisabled:
		return nil
	default:
		return fmt.Errorf("invalid agent status: %s", s)
	}
}

func ParseAgentStatus(value string) (AgentStatus, error) {
	status := AgentStatus(value)
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

// Agent represents a service manager agent
type Agent struct {
	BaseEntity

	Name        string      `json:"name" gorm:"not null"`
	Attributes  Attributes  `json:"attributes,omitempty" gorm:"type:jsonb"`
	CountryCode CountryCode `json:"countryCode,omitempty" gorm:"size:2"`

	// Status management
	Status           AgentStatus `json:"status" gorm:"not null"`
	LastStatusUpdate time.Time   `json:"lastStatusUpdate" gorm:"index"`

	// Relationships
	AgentTypeID UUID         `json:"agentTypeId" gorm:"not null"`
	AgentType   *AgentType   `json:"agentType,omitempty" gorm:"foreignKey:AgentTypeID"`
	ProviderID  UUID         `json:"providerId" gorm:"not null"`
	Provider    *Participant `json:"-" gorm:"foreignKey:ProviderID"`
}

// NewAgent creates a new agent with proper validation
func NewAgent(name string, countryCode CountryCode, attributes Attributes, providerID UUID, agentTypeID UUID) *Agent {
	return &Agent{
		Name:             name,
		Status:           AgentDisconnected,
		LastStatusUpdate: time.Now(),
		CountryCode:      countryCode,
		Attributes:       attributes,
		ProviderID:       providerID,
		AgentTypeID:      agentTypeID,
	}
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
	if err := a.Status.Validate(); err != nil {
		return err
	}
	if a.LastStatusUpdate.IsZero() {
		return fmt.Errorf("status last update cannot be empty")
	}
	if a.AgentTypeID == uuid.Nil {
		return fmt.Errorf("agent type ID cannot be empty")
	}
	if a.ProviderID == uuid.Nil {
		return fmt.Errorf("provider ID cannot be empty")
	}
	if err := a.CountryCode.Validate(); err != nil {
		// Allow empty country code
		if string(a.CountryCode) != "" {
			return err
		}
	}
	if a.Attributes != nil {
		if err := a.Attributes.Validate(); err != nil {
			return err
		}
	}
	if a.Attributes != nil {
		if err := a.Attributes.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// UpdateStatus updates the agent's status and last update timestamp
func (a *Agent) UpdateStatus(newStatus AgentStatus) {
	a.Status = newStatus
	a.LastStatusUpdate = time.Now()
}

// UpdateHeartbeat updates the last status update timestamp without changing the status
func (a *Agent) UpdateHeartbeat() {
	a.LastStatusUpdate = time.Now()
}

// RegisterMetadata updates the agent's metadata properties (name, country code, attributes)
func (a *Agent) RegisterMetadata(name *string, countryCode *CountryCode, attributes *Attributes) {
	if name != nil {
		a.Name = *name
	}

	if countryCode != nil {
		a.CountryCode = *countryCode
	}

	if attributes != nil {
		a.Attributes = *attributes
	}
}

// AgentCommander defines the interface for agent command operations
type AgentCommander interface {
	// Create creates a new agent
	Create(ctx context.Context, name string, countryCode CountryCode, attributes Attributes, providerID UUID, agentTypeID UUID) (*Agent, error)

	// Update updates an agent
	Update(ctx context.Context, id UUID, name *string, countryCode *CountryCode, attributes *Attributes, status *AgentStatus) (*Agent, error)

	// Delete removes an agent by ID after checking for dependencies
	Delete(ctx context.Context, id UUID) error

	// UpdateStatus updates the agent status and the related timestamp
	UpdateStatus(ctx context.Context, id UUID, status AgentStatus) (*Agent, error)
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
	// Validate references
	// Assuming store.ParticipantRepo().Exists will be available
	providerExists, err := s.store.ParticipantRepo().Exists(ctx, providerID)
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

	// Create and save
	var agent *Agent
	err = s.store.Atomic(ctx, func(store Store) error {
		agent = NewAgent(name, countryCode, attributes, providerID, agentTypeID)
		if err := agent.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.AgentRepo().Create(ctx, agent); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtx(
			ctx, EventTypeAgentCreated, JSON{"status": agent},
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
	status *AgentStatus,
) (*Agent, error) {
	// Find it
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	beforeAgent := *agent

	// Update and validate
	if status != nil {
		agent.UpdateStatus(*status)
	}
	if name != nil || countryCode != nil || attributes != nil {
		agent.RegisterMetadata(name, countryCode, attributes)
	}
	if err := agent.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.AgentRepo().Save(ctx, agent)
		if err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtxWithDiff(ctx, EventTypeAgentUpdated,
			&id, &agent.ProviderID, nil, nil, &beforeAgent, agent)
		return err
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *agentCommander) Delete(ctx context.Context, id UUID) error {
	// Find it
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}
	providerID := agent.ProviderID

	// Delete and audit
	return s.store.Atomic(ctx, func(store Store) error {
		// Check dependencies
		numOfServices, err := store.ServiceRepo().CountByAgent(ctx, id)
		if err != nil {
			return err
		}
		if numOfServices > 0 {
			return errors.New("cannot delete agent with associated services")
		}

		if err := store.TokenRepo().DeleteByAgentID(ctx, id); err != nil {
			return err
		}
		if err := store.AgentRepo().Delete(ctx, id); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtx(ctx, EventTypeAgentDeleted,
			JSON{"status": agent}, &id, &providerID, nil, nil)
		return err
	})
}

func (s *agentCommander) UpdateStatus(ctx context.Context, id UUID, status AgentStatus) (*Agent, error) {
	// Find it
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	beforeAgent := *agent

	// Update and validate
	agent.UpdateStatus(status)
	if err := agent.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.AgentRepo().Save(ctx, agent)
		if err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtxWithDiff(ctx, EventTypeAgentUpdated,
			&id, &agent.ProviderID, nil, nil, &beforeAgent, agent)
		return err
	})
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

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Agent], error)

	// CountByProvider returns the number of agents for a specific provider
	CountByProvider(ctx context.Context, providerID UUID) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}
