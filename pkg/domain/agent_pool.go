package domain

import (
	"context"
	"fmt"
	"slices"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeAgentPoolCreated EventType = "agent_pool.created"
	EventTypeAgentPoolUpdated EventType = "agent_pool.updated"
	EventTypeAgentPoolDeleted EventType = "agent_pool.deleted"
)

type AgentPool struct {
	BaseEntity
	Name         string `json:"name" gorm:"not null"`
	Type         string `json:"type" gorm:"not null"`
	PropertyType string `json:"propertyType" gorm:"not null"`
	GeneratorType   PoolGeneratorType `json:"generatorType" gorm:"not null"`
	GeneratorConfig *properties.JSON  `json:"generatorConfig,omitempty" gorm:"type:jsonb"`
}

// CreateAgentPoolParams defines parameters for creating an AgentPool
type CreateAgentPoolParams struct {
	Name            string
	Type            string
	PropertyType    string
	GeneratorType   PoolGeneratorType
	GeneratorConfig *properties.JSON
}

// UpdateAgentPoolParams defines parameters for updating an AgentPool
type UpdateAgentPoolParams struct {
	Name            *string
	GeneratorConfig *properties.JSON
}

// NewAgentPool creates a new agent pool without validation
func NewAgentPool(params CreateAgentPoolParams) *AgentPool {
	return &AgentPool{
		Name:            params.Name,
		Type:            params.Type,
		PropertyType:    params.PropertyType,
		GeneratorType:   params.GeneratorType,
		GeneratorConfig: params.GeneratorConfig,
	}
}

func (AgentPool) TableName() string {
	return "agent_pools"
}

func (ap *AgentPool) Validate() error {
	if ap.Name == "" {
		return fmt.Errorf("agent pool name cannot be empty")
	}

	if ap.Type == "" {
		return fmt.Errorf("agent pool type cannot be empty")
	}

	if !slices.Contains(ValidPoolPropertyTypes, ap.PropertyType) {
		return fmt.Errorf("invalid property type: %s (must be one of: %v)", ap.PropertyType, ValidPoolPropertyTypes)
	}

	if ap.GeneratorType != PoolGeneratorList {
		return fmt.Errorf("invalid generator type for agent pool: %s (must be %s)", ap.GeneratorType, PoolGeneratorList)
	}

	return nil
}

// Update modifies the AgentPool with provided parameters
func (ap *AgentPool) Update(params UpdateAgentPoolParams) {
	if params.Name != nil {
		ap.Name = *params.Name
	}
	if params.GeneratorConfig != nil {
		ap.GeneratorConfig = params.GeneratorConfig
	}
}

type AgentPoolQuerier interface {
	BaseEntityQuerier[AgentPool]
}

type AgentPoolRepository interface {
	AgentPoolQuerier
	Create(ctx context.Context, pool *AgentPool) error
	Update(ctx context.Context, pool *AgentPool) error
	Delete(ctx context.Context, id properties.UUID) error
}

// AgentPoolCommander handles complex AgentPool operations
type AgentPoolCommander interface {
	Create(ctx context.Context, params CreateAgentPoolParams) (*AgentPool, error)
	Update(ctx context.Context, id properties.UUID, params UpdateAgentPoolParams) (*AgentPool, error)
	Delete(ctx context.Context, id properties.UUID) error
}

type agentPoolCommander struct {
	store Store
}

func NewAgentPoolCommander(store Store) AgentPoolCommander {
	return &agentPoolCommander{store: store}
}

func (c *agentPoolCommander) Create(
	ctx context.Context,
	params CreateAgentPoolParams,
) (*AgentPool, error) {
	var pool *AgentPool
	err := c.store.Atomic(ctx, func(store Store) error {
		pool = NewAgentPool(params)
		if err := pool.Validate(); err != nil {
			return err
		}

		if err := store.AgentPoolRepo().Create(ctx, pool); err != nil {
			return err
		}

		eventEntity, err := NewEvent(EventTypeAgentPoolCreated, WithInitiatorCtx(ctx), WithAgentPool(pool))
		if err != nil {
			return err
		}

		if err := store.EventRepo().Create(ctx, eventEntity); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (c *agentPoolCommander) Update(
	ctx context.Context,
	id properties.UUID,
	params UpdateAgentPoolParams,
) (*AgentPool, error) {
	var pool *AgentPool
	err := c.store.Atomic(ctx, func(store Store) error {
		var err error
		pool, err = store.AgentPoolRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		beforeAgentPool := *pool

		pool.Update(params)

		if err := pool.Validate(); err != nil {
			return err
		}

		if err := store.AgentPoolRepo().Update(ctx, pool); err != nil {
			return err
		}

		eventEntity, err := NewEvent(EventTypeAgentPoolUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeAgentPool, pool), WithAgentPool(pool))
		if err != nil {
			return err
		}

		if err := store.EventRepo().Create(ctx, eventEntity); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (c *agentPoolCommander) Delete(
	ctx context.Context,
	id properties.UUID,
) error {
	return c.store.Atomic(ctx, func(store Store) error {
		pool, err := store.AgentPoolRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		eventEntity, err := NewEvent(EventTypeAgentPoolDeleted, WithInitiatorCtx(ctx), WithAgentPool(pool))
		if err != nil {
			return err
		}

		if err := store.EventRepo().Create(ctx, eventEntity); err != nil {
			return err
		}

		if err := store.AgentPoolRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}
