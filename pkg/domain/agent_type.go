package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

const (
	EventTypeAgentTypeCreated EventType = "agent_type.created"
	EventTypeAgentTypeUpdated EventType = "agent_type.updated"
	EventTypeAgentTypeDeleted EventType = "agent_type.deleted"
)

// AgentType represents a type of service manager agent
type AgentType struct {
	BaseEntity
	Name                string        `json:"name" gorm:"not null;unique"`
	ServiceTypes        []ServiceType `json:"-" gorm:"many2many:agent_type_service_types;"`
	ConfigurationSchema schema.Schema `json:"configurationSchema" gorm:"type:jsonb;not null"`
}

// NewAgentType creates a new agent type without validation
func NewAgentType(params CreateAgentTypeParams) *AgentType {
	// Convert service type IDs to ServiceType entities for GORM relationship handling
	serviceTypes := make([]ServiceType, 0, len(params.ServiceTypeIds))
	for _, id := range params.ServiceTypeIds {
		serviceTypes = append(serviceTypes, ServiceType{
			BaseEntity: BaseEntity{ID: id},
		})
	}

	return &AgentType{
		Name:                params.Name,
		ServiceTypes:        serviceTypes,
		ConfigurationSchema: params.ConfigurationSchema,
	}
}

// TableName returns the table name for the agent type
func (AgentType) TableName() string {
	return "agent_types"
}

// Validate ensures all AgentType fields are valid (without schema validation)
func (at *AgentType) Validate() error {
	if at.Name == "" {
		return fmt.Errorf("agent type name cannot be empty")
	}
	return nil
}

// ValidateWithEngine validates the agent type including its configuration schema
func (at *AgentType) ValidateWithEngine(engine *schema.Engine[AgentConfigContext]) error {
	if at.Name == "" {
		return fmt.Errorf("agent type name cannot be empty")
	}

	// Always validate schema (required, not nullable)
	if err := engine.ValidateSchema(at.ConfigurationSchema); err != nil {
		return fmt.Errorf("configurationSchema: %w", err)
	}

	return nil
}

// Update updates the agent type fields if the pointers are non-nil
func (at *AgentType) Update(params UpdateAgentTypeParams) {
	if params.Name != nil {
		at.Name = *params.Name
	}
	if params.ServiceTypeIds != nil {
		// Convert service type IDs to ServiceType entities for GORM relationship handling
		serviceTypes := make([]ServiceType, 0, len(*params.ServiceTypeIds))
		for _, id := range *params.ServiceTypeIds {
			serviceTypes = append(serviceTypes, ServiceType{
				BaseEntity: BaseEntity{ID: id},
			})
		}
		at.ServiceTypes = serviceTypes
	}
	if params.ConfigurationSchema != nil {
		at.ConfigurationSchema = *params.ConfigurationSchema
	}
}

// AgentTypeCommander defines the interface for agent type command operations
type AgentTypeCommander interface {
	// Create creates a new agent type
	Create(ctx context.Context, params CreateAgentTypeParams) (*AgentType, error)

	// Update updates an agent type
	Update(ctx context.Context, params UpdateAgentTypeParams) (*AgentType, error)

	// Delete removes an agent type by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error
}

type CreateAgentTypeParams struct {
	Name                string            `json:"name"`
	ServiceTypeIds      []properties.UUID `json:"serviceTypeIds,omitempty"`
	ConfigurationSchema schema.Schema     `json:"configurationSchema"`
}

type UpdateAgentTypeParams struct {
	ID                  properties.UUID    `json:"id"`
	Name                *string            `json:"name"`
	ServiceTypeIds      *[]properties.UUID `json:"serviceTypeIds,omitempty"`
	ConfigurationSchema *schema.Schema     `json:"configurationSchema,omitempty"`
}

// agentTypeCommander is the concrete implementation of AgentTypeCommander
type agentTypeCommander struct {
	store        Store
	configEngine *schema.Engine[AgentConfigContext]
}

// NewAgentTypeCommander creates a new AgentTypeCommander
func NewAgentTypeCommander(
	store Store,
	configEngine *schema.Engine[AgentConfigContext],
) AgentTypeCommander {
	return &agentTypeCommander{
		store:        store,
		configEngine: configEngine,
	}
}

// Create creates a new agent type
func (c *agentTypeCommander) Create(
	ctx context.Context,
	params CreateAgentTypeParams,
) (*AgentType, error) {
	var agentType *AgentType
	err := c.store.Atomic(ctx, func(store Store) error {
		agentType = NewAgentType(params)

		// Use engine to validate (includes schema validation)
		if err := agentType.ValidateWithEngine(c.configEngine); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.AgentTypeRepo().Create(ctx, agentType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeAgentTypeCreated, WithInitiatorCtx(ctx), WithAgentType(agentType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return agentType, nil
}

// Update updates an agent type
func (c *agentTypeCommander) Update(
	ctx context.Context,
	params UpdateAgentTypeParams,
) (*AgentType, error) {
	agentType, err := c.store.AgentTypeRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy of the agent type before modifications for event diff
	beforeAgentType := *agentType

	// Update and validate
	agentType.Update(params)
	if err := agentType.ValidateWithEngine(c.configEngine); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.AgentTypeRepo().Save(ctx, agentType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeAgentTypeUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeAgentType, agentType), WithAgentType(agentType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return agentType, nil
}

// Delete removes an agent type by ID after checking for dependencies
func (c *agentTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	agentType, err := c.store.AgentTypeRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		// Check for dependent Agents
		agentCount, err := store.AgentRepo().CountByAgentType(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count agents for agent type %s: %w", id, err)
		}
		if agentCount > 0 {
			return NewInvalidInputErrorf("cannot delete agent type %s: %d dependent agent(s) exist", id, agentCount)
		}

		eventEntry, err := NewEvent(EventTypeAgentTypeDeleted, WithInitiatorCtx(ctx), WithAgentType(agentType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		if err := store.AgentTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}

// AgentTypeRepository defines the interface for the AgentType repository
type AgentTypeRepository interface {
	AgentTypeQuerier
	BaseEntityRepository[AgentType]
}

// AgentTypeQuerier defines the interface for the AgentType read-only queries
type AgentTypeQuerier interface {
	BaseEntityQuerier[AgentType]
}
