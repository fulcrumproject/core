package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
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
		return NewInvalidInputError("invalid agent status: '{status}'", MsgData{"status": s})
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

	// Configuration stores instance-specific configuration parameters as JSON
	Configuration *properties.JSON `json:"configuration,omitempty" gorm:"type:jsonb"`

	// Relationships
	AgentTypeID      properties.UUID  `json:"agentTypeId" gorm:"not null"`
	AgentType        *AgentType       `json:"agentType,omitempty" gorm:"foreignKey:AgentTypeID"`
	ProviderID       properties.UUID  `json:"providerId" gorm:"not null"`
	Provider         *Participant     `json:"-" gorm:"foreignKey:ProviderID"`
	ServicePoolSetID *properties.UUID `json:"servicePoolSetId,omitempty"`
	ServicePoolSet   *ServicePoolSet  `json:"-" gorm:"foreignKey:ServicePoolSetID"`
	InfrastructureID *properties.UUID `json:"infrastructureId,omitempty"`
	Infrastructure   *Infrastructure  `json:"infrastructure,omitempty" gorm:"foreignKey:InfrastructureID"`
}

// NewAgent creates a new agent with proper validation
func NewAgent(params CreateAgentParams) *Agent {
	return &Agent{
		Name:             params.Name,
		Status:           AgentDisconnected,
		LastStatusUpdate: time.Now(),
		ProviderID:       params.ProviderID,
		AgentTypeID:      params.AgentTypeID,
		Configuration:    params.Configuration,
		ServicePoolSetID: params.ServicePoolSetID,
		InfrastructureID: params.InfrastructureID,
	}
}

// TableName returns the table name for the agent
func (Agent) TableName() string {
	return "agents"
}

// Validate ensures all agent fields are valid
func (a *Agent) Validate() error {
	if a.Name == "" {
		return NewInvalidInputError("agent name cannot be empty", nil)
	}

	if err := a.Status.Validate(); err != nil {
		return err
	}

	if a.LastStatusUpdate.IsZero() {
		return NewInvalidInputError("status last update cannot be empty", nil)
	}

	if a.AgentTypeID == uuid.Nil {
		return NewInvalidInputError("agent type ID cannot be empty", nil)
	}
	if a.ProviderID == uuid.Nil {
		return NewInvalidInputError("provider ID cannot be empty", nil)
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
func (a *Agent) Update(name *string, configuration *properties.JSON, servicePoolSetID *properties.UUID) bool {
	updated := false

	if name != nil {
		a.Name = *name
		updated = true
	}

	if configuration != nil {
		a.Configuration = configuration
		updated = true
	}

	if servicePoolSetID != nil {
		a.ServicePoolSetID = servicePoolSetID
		a.ServicePoolSet = nil
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
	Name             string           `json:"name"`
	ProviderID       properties.UUID  `json:"providerId"`
	AgentTypeID      properties.UUID  `json:"agentTypeId"`
	Configuration    *properties.JSON `json:"configuration,omitempty"`
	ServicePoolSetID *properties.UUID `json:"servicePoolSetId,omitempty"`
	InfrastructureID *properties.UUID `json:"infrastructureId,omitempty"`
}

type UpdateAgentParams struct {
	ID               properties.UUID  `json:"id"`
	Name             *string          `json:"name,omitempty"`
	Status           *AgentStatus     `json:"status,omitempty"`
	Configuration    *properties.JSON `json:"configuration,omitempty"`
	ServicePoolSetID *properties.UUID `json:"servicePoolSetId,omitempty"`
}

type UpdateAgentStatusParams struct {
	ID     properties.UUID `json:"id"`
	Status AgentStatus     `json:"status"`
}

// agentCommander is the concrete implementation of AgentCommander
type agentCommander struct {
	store        Store
	configEngine *schema.Engine[AgentConfigContext]
}

// NewAgentCommander creates a new default AgentCommander
func NewAgentCommander(
	store Store,
	configEngine *schema.Engine[AgentConfigContext],
) *agentCommander {
	return &agentCommander{
		store:        store,
		configEngine: configEngine,
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
		return nil, NewInvalidInputError("provider with ID {providerId} does not exist", MsgData{"providerId": params.ProviderID})
	}

	// Get agent type to access configuration schema
	agentType, err := s.store.AgentTypeRepo().Get(ctx, params.AgentTypeID)
	if err != nil {
		return nil, NewInvalidInputError("agent type with ID {agentTypeId} does not exist", MsgData{"agentTypeId": params.AgentTypeID})
	}

	requiredIT := agentType.RequiredInfrastructureType()
	if requiredIT == nil {
		if params.InfrastructureID != nil {
			return nil, NewInvalidInputError("agent type {agentTypeId} does not allow an infrastructure", MsgData{"agentTypeId": params.AgentTypeID})
		}
	} else {
		if params.InfrastructureID == nil {
			return nil, NewInvalidInputError("agent type {agentTypeId} requires an infrastructure of type {infrastructureTypeId}", MsgData{"agentTypeId": params.AgentTypeID, "infrastructureTypeId": requiredIT.ID})
		}
		infra, err := s.store.InfrastructureRepo().Get(ctx, *params.InfrastructureID)
		if err != nil {
			return nil, NewInvalidInputError("infrastructure with ID {infrastructureId} does not exist", MsgData{"infrastructureId": *params.InfrastructureID})
		}
		if infra.ProviderID != params.ProviderID {
			return nil, NewInvalidInputError("infrastructure with ID {infrastructureId} does not belong to provider {providerId}", MsgData{"infrastructureId": *params.InfrastructureID, "providerId": params.ProviderID})
		}
		if infra.InfrastructureTypeID != requiredIT.ID {
			return nil, NewInvalidInputError("infrastructure with ID {infrastructureId} has type {actualType}, expected {expectedType}", MsgData{"infrastructureId": *params.InfrastructureID, "actualType": infra.InfrastructureTypeID, "expectedType": requiredIT.ID})
		}
	}

	if params.ServicePoolSetID != nil {
		servicePoolSet, err := s.store.ServicePoolSetRepo().Get(ctx, *params.ServicePoolSetID)
		if err != nil {
			return nil, NewInvalidInputError("service pool set with ID {servicePoolSetId} does not exist", MsgData{"servicePoolSetId": *params.ServicePoolSetID})
		}

		if servicePoolSet.ProviderID != params.ProviderID {
			return nil, NewInvalidInputError("service pool set with ID {servicePoolSetId} does not belong to provider {providerId}", MsgData{"servicePoolSetId": *params.ServicePoolSetID, "providerId": params.ProviderID})
		}
	}

	// Pre-generate agent ID upfront so pool generators can stamp allocations with it
	// within the same transaction as the agent insert.
	agentID := properties.UUID(uuid.New())

	// Create and save
	var agent *Agent
	err = s.store.Atomic(ctx, func(store Store) error {
		agent = NewAgent(params)
		agent.ID = agentID

		// Validate and process configuration against schema
		if agent.Configuration != nil {
			schemaCtx := AgentConfigContext{
				Store:           store,
				AgentID:         &agentID,
				AgentProviderID: agent.ProviderID,
			}

			// Convert configuration to map
			configMap := map[string]any(*agent.Configuration)

			// Use injected engine to process configuration
			// This validates types, runs validators, applies defaults, processes secrets
			processedConfig, err := s.configEngine.ApplyCreate(
				ctx,
				schemaCtx,
				agentType.ConfigurationSchema,
				configMap,
			)
			if err != nil {
				return InvalidInputError{Err: fmt.Errorf("configuration: %w", err)}
			}

			// Update agent with processed configuration (secrets replaced with vault:// refs)
			processedJSON := properties.JSON(processedConfig)
			agent.Configuration = &processedJSON
		}

		if err := agent.Validate(); err != nil {
			return asInvalidInput(err)
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

	// Get agent type to access configuration schema
	agentType, err := s.store.AgentTypeRepo().Get(ctx, agent.AgentTypeID)
	if err != nil {
		return nil, err
	}

	if params.ServicePoolSetID != nil {
		servicePoolSet, err := s.store.ServicePoolSetRepo().Get(ctx, *params.ServicePoolSetID)
		if err != nil {
			return nil, err
		}

		if servicePoolSet.ProviderID != agent.ProviderID {
			return nil, NewInvalidInputError("service pool set with ID {servicePoolSetId} does not belong to provider {providerId}", MsgData{"servicePoolSetId": *params.ServicePoolSetID, "providerId": agent.ProviderID})
		}
	}

	// Update and validate
	if params.Status != nil {
		agent.UpdateStatus(*params.Status)
	}
	agent.Update(params.Name, params.Configuration, params.ServicePoolSetID)

	// Save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		// Validate and process configuration against schema if configuration is being updated.
		// Done inside the transaction so any pool allocations share the same tx as the save.
		if params.Configuration != nil && agent.Configuration != nil {
			schemaCtx := AgentConfigContext{
				Store:           store,
				AgentID:         &agent.ID,
				AgentProviderID: agent.ProviderID,
			}

			var oldConfigMap map[string]any
			if beforeAgent.Configuration != nil {
				oldConfigMap = map[string]any(*beforeAgent.Configuration)
			}
			newConfigMap := map[string]any(*agent.Configuration)

			processedConfig, err := s.configEngine.ApplyUpdate(
				ctx,
				schemaCtx,
				agentType.ConfigurationSchema,
				oldConfigMap,
				newConfigMap,
			)
			if err != nil {
				return InvalidInputError{Err: fmt.Errorf("configuration: %w", err)}
			}

			processedJSON := properties.JSON(processedConfig)
			agent.Configuration = &processedJSON
		}

		if err := agent.Validate(); err != nil {
			return asInvalidInput(err)
		}

		if err := store.AgentRepo().Save(ctx, agent); err != nil {
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
			return NewInvalidInputError("cannot delete agent with associated services", nil)
		}

		if err := store.TokenRepo().DeleteByAgentID(ctx, id); err != nil {
			return err
		}

		// Release any ConfigPoolValue rows allocated to this agent. Dispatched per pool via
		// the factory so release semantics stay consistent across generator types (list today,
		// potentially subnet later).
		allocated, err := store.ConfigPoolValueRepo().FindByAgent(ctx, id)
		if err != nil {
			return err
		}
		if len(allocated) > 0 {
			factory := NewDefaultConfigPoolGeneratorFactory(store.ConfigPoolValueRepo())
			seen := make(map[properties.UUID]bool, len(allocated))
			for _, v := range allocated {
				if seen[v.ConfigPoolID] {
					continue
				}
				seen[v.ConfigPoolID] = true
				pool, err := store.ConfigPoolRepo().Get(ctx, v.ConfigPoolID)
				if err != nil {
					return err
				}
				gen, err := factory.CreateGenerator(pool)
				if err != nil {
					return err
				}
				if err := gen.Release(ctx, allocated); err != nil {
					return err
				}
			}
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
		return nil, asInvalidInput(err)
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

	// CountByInfrastructure returns the number of agents bound to a specific infrastructure
	CountByInfrastructure(ctx context.Context, infrastructureID properties.UUID) (int64, error)
}
