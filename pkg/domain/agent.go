package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	EventTypeAgentCreated EventType = "agent.created"
	EventTypeAgentUpdated EventType = "agent.updated"
	EventTypeAgentDeleted EventType = "agent.deleted"
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

	Name string `json:"name" gorm:"not null"`

	// Status management
	Status           AgentStatus `json:"status" gorm:"not null"`
	LastStatusUpdate time.Time   `json:"lastStatusUpdate" gorm:"index"`

	// Tags representing capabilities or certifications of this agent
	Tags pq.StringArray `json:"tags" gorm:"type:text[]"`

	// Configuration stores instance-specific configuration parameters as JSON
	Configuration *properties.JSON `json:"configuration,omitempty" gorm:"type:jsonb"`

	// Relationships
	AgentTypeID properties.UUID `json:"agentTypeId" gorm:"not null"`
	AgentType   *AgentType      `json:"agentType,omitempty" gorm:"foreignKey:AgentTypeID"`
	ProviderID  properties.UUID `json:"providerId" gorm:"not null"`
	Provider    *Participant    `json:"-" gorm:"foreignKey:ProviderID"`
}

// NewAgent creates a new agent with proper validation
func NewAgent(params CreateAgentParams) *Agent {
	return &Agent{
		Name:             params.Name,
		Status:           AgentDisconnected,
		LastStatusUpdate: time.Now(),
		ProviderID:       params.ProviderID,
		AgentTypeID:      params.AgentTypeID,
		Tags:             pq.StringArray(params.Tags),
		Configuration:    params.Configuration,
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

	for i, tag := range []string(a.Tags) {
		if len(tag) == 0 {
			return fmt.Errorf("tag at index %d cannot be empty", i)
		}
		if len(tag) > 100 {
			return fmt.Errorf("tag at index %d exceeds maximum length of 100 characters", i)
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

// RegisterMetadata updates the agent's metadata properties (name)
func (a *Agent) RegisterMetadata(name *string) {
	if name != nil {
		a.Name = *name
	}
}

// Update updates the agent's fields
func (a *Agent) Update(name *string, tags *[]string, configuration *properties.JSON) bool {
	updated := false

	if name != nil {
		a.Name = *name
		updated = true
	}

	if tags != nil {
		a.Tags = pq.StringArray(*tags)
		updated = true
	}

	if configuration != nil {
		a.Configuration = configuration
		updated = true
	}

	return updated
}

// AgentCommander defines the interface for agent command operations
type AgentCommander interface {
	// Create creates a new agent
	Create(ctx context.Context, params CreateAgentParams) (*Agent, error)

	// Update updates an agent
	Update(ctx context.Context, params UpdateAgentParams) (*Agent, error)

	// Delete removes an agent by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error

	// UpdateStatus updates the agent status and the related timestamp
	UpdateStatus(ctx context.Context, params UpdateAgentStatusParams) (*Agent, error)
}

type CreateAgentParams struct {
	Name          string           `json:"name"`
	ProviderID    properties.UUID  `json:"providerId"`
	AgentTypeID   properties.UUID  `json:"agentTypeId"`
	Tags          []string         `json:"tags"`
	Configuration *properties.JSON `json:"configuration,omitempty"`
}

type UpdateAgentParams struct {
	ID            properties.UUID  `json:"id"`
	Name          *string          `json:"name,omitempty"`
	Status        *AgentStatus     `json:"status,omitempty"`
	Tags          *[]string        `json:"tags,omitempty"`
	Configuration *properties.JSON `json:"configuration,omitempty"`
}

type UpdateAgentStatusParams struct {
	ID     properties.UUID `json:"id"`
	Status AgentStatus     `json:"status"`
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
	params CreateAgentParams,
) (*Agent, error) {
	// Validate references
	// Assuming store.ParticipantRepo().Exists will be available
	providerExists, err := s.store.ParticipantRepo().Exists(ctx, params.ProviderID)
	if err != nil {
		return nil, err
	}
	if !providerExists {
		return nil, NewInvalidInputErrorf("provider with ID %s does not exist", params.ProviderID)
	}
	agentTypeExists, err := s.store.AgentTypeRepo().Exists(ctx, params.AgentTypeID)
	if err != nil {
		return nil, err
	}
	if !agentTypeExists {
		return nil, NewInvalidInputErrorf("agent type with ID %s does not exist", params.AgentTypeID)
	}

	// Create and save
	var agent *Agent
	err = s.store.Atomic(ctx, func(store Store) error {
		agent = NewAgent(params)
		if err := agent.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.AgentRepo().Create(ctx, agent); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeAgentCreated, WithInitiatorCtx(ctx), WithAgent(agent))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *agentCommander) Update(ctx context.Context,
	params UpdateAgentParams,
) (*Agent, error) {
	// Find it
	agent, err := s.store.AgentRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	beforeAgent := *agent

	// Update and validate
	if params.Status != nil {
		agent.UpdateStatus(*params.Status)
	}
	agent.Update(params.Name, params.Tags, params.Configuration)
	if err := agent.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.AgentRepo().Save(ctx, agent)
		if err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeAgentUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeAgent, agent), WithAgent(agent))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *agentCommander) Delete(ctx context.Context, id properties.UUID) error {
	// Find it
	agent, err := s.store.AgentRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete and event
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
		eventEntry, err := NewEvent(EventTypeAgentDeleted, WithInitiatorCtx(ctx), WithAgent(agent))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
}

func (s *agentCommander) UpdateStatus(ctx context.Context, params UpdateAgentStatusParams) (*Agent, error) {
	// Find it
	agent, err := s.store.AgentRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	beforeAgent := *agent

	// Update and validate
	agent.UpdateStatus(params.Status)
	if err := agent.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.AgentRepo().Save(ctx, agent)
		if err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeAgentUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeAgent, agent), WithAgent(agent))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

type AgentRepository interface {
	AgentQuerier
	BaseEntityRepository[Agent]

	// MarkInactiveAgentsAsDisconnected marks agents that haven't updated their status in the given duration as disconnected
	MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error)
}

type AgentQuerier interface {
	BaseEntityQuerier[Agent]

	// CountByProvider returns the number of agents for a specific provider
	CountByProvider(ctx context.Context, providerID properties.UUID) (int64, error)

	// CountByAgentType returns the number of agents for a specific agent type
	CountByAgentType(ctx context.Context, agentTypeID properties.UUID) (int64, error)

	// FindByServiceTypeAndTags finds agents that support a service type and have all required tags
	FindByServiceTypeAndTags(ctx context.Context, serviceTypeID properties.UUID, tags []string) ([]*Agent, error)
}
