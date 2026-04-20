package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeAgentPoolValueCreated EventType = "agent_pool_value.created"
	EventTypeAgentPoolValueDeleted EventType = "agent_pool_value.deleted"
)

type AgentPoolValue struct {
	BaseEntity
	Name         string           `json:"name" gorm:"not null"`
	Value        any              `json:"value" gorm:"type:jsonb;serializer:json;not null"`
	AgentPoolID  properties.UUID  `json:"agentPoolId" gorm:"not null;index"`
	AgentPool    *AgentPool       `json:"-" gorm:"foreignKey:AgentPoolID"`
	AgentID      *properties.UUID `json:"agentId,omitempty" gorm:"index"`
	Agent        *Agent           `json:"-" gorm:"foreignKey:AgentID"`
	PropertyName *string          `json:"propertyName"`
	AllocatedAt  *time.Time       `json:"allocatedAt,omitempty"`
}

func (AgentPoolValue) TableName() string {
	return "agent_pool_values"
}

func (ag *AgentPoolValue) Validate() error {
	if ag.Name == "" {
		return fmt.Errorf("agent pool value name is required")
	}

	if ag.Value == nil {
		return fmt.Errorf("agent pool value is required")
	}

	if ag.AgentPoolID == (properties.UUID{}) {
		return fmt.Errorf("agent pool ID cannot be empty")
	}
	return nil
}

func (ag *AgentPoolValue) IsAllocated() bool {
	return ag.AgentID != nil
}

func (ag *AgentPoolValue) Allocate(agentID properties.UUID, propertyName string) {
	allocated := time.Now()
	ag.AgentID = &agentID
	ag.AllocatedAt = &allocated
	ag.PropertyName = &propertyName
}

func (ag *AgentPoolValue) Release() {
	ag.AgentID = nil
	ag.AllocatedAt = nil
	ag.PropertyName = nil
}

func (ag *AgentPoolValue) PoolID() properties.UUID {
	return ag.AgentPoolID
}

func (ag *AgentPoolValue) RawValue() any {
	return ag.Value
}

type CreateAgentPoolValueParams struct {
	Name        string
	Value       any
	AgentPoolID properties.UUID
}

func NewAgentPoolValue(c CreateAgentPoolValueParams) *AgentPoolValue {
	return &AgentPoolValue{
		Name:        c.Name,
		Value:       c.Value,
		AgentPoolID: c.AgentPoolID,
	}
}

type AgentPoolValueQuerier interface {
	BaseEntityQuerier[AgentPoolValue]
	FindAvailable(ctx context.Context, poolID properties.UUID) ([]*AgentPoolValue, error)
	FindByAgent(ctx context.Context, agentID properties.UUID) ([]*AgentPoolValue, error)
}

type AgentPoolValueRepository interface {
	AgentPoolValueQuerier
	Create(ctx context.Context, value *AgentPoolValue) error
	Update(ctx context.Context, value *AgentPoolValue) error
	Delete(ctx context.Context, id properties.UUID) error
}

type AgentPoolValueCommander interface {
	Create(ctx context.Context, params CreateAgentPoolValueParams) (*AgentPoolValue, error)
	Delete(ctx context.Context, id properties.UUID) error
}

type agentPoolValueCommander struct {
	store Store
}

func NewAgentPoolValueCommander(store Store) AgentPoolValueCommander {
	return &agentPoolValueCommander{store: store}
}

func (c *agentPoolValueCommander) Create(ctx context.Context, params CreateAgentPoolValueParams) (*AgentPoolValue, error) {
	var poolValue *AgentPoolValue
	err := c.store.Atomic(ctx, func(s Store) error {
		exists, err := s.AgentPoolRepo().Exists(ctx, params.AgentPoolID)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("agent pool with id %s not found", params.AgentPoolID)
		}

		poolValue = NewAgentPoolValue(params)
		if err := poolValue.Validate(); err != nil {
			return err
		}

		if err := s.AgentPoolValueRepo().Create(ctx, poolValue); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeAgentPoolValueCreated, WithInitiatorCtx(ctx), WithAgentPoolValue(poolValue))
		if err != nil {
			return err
		}

		if err := s.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return poolValue, nil
}

func (c *agentPoolValueCommander) Delete(ctx context.Context, id properties.UUID) error {
	return c.store.Atomic(ctx, func(s Store) error {
		value, err := s.AgentPoolValueRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		if value.IsAllocated() {
			return NewInvalidInputErrorf("cannot delete allocated pool value")
		}

		eventEntry, err := NewEvent(EventTypeAgentPoolValueDeleted, WithInitiatorCtx(ctx), WithAgentPoolValue(value))
		if err != nil {
			return err
		}
		if err := s.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return s.AgentPoolValueRepo().Delete(ctx, id)
	})
}
