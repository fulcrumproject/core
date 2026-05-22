package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeConfigPoolValueCreated EventType = "config_pool_value.created"
	EventTypeConfigPoolValueDeleted EventType = "config_pool_value.deleted"
)

type ConfigPoolValue struct {
	BaseEntity
	Name          string           `json:"name" gorm:"not null"`
	Value         any              `json:"value" gorm:"type:jsonb;serializer:json;not null"`
	ConfigPoolID  properties.UUID  `json:"configPoolId" gorm:"not null;index"`
	ConfigPool    *ConfigPool      `json:"-" gorm:"foreignKey:ConfigPoolID"`
	AgentID       *properties.UUID `json:"agentId,omitempty" gorm:"index"`
	Agent         *Agent           `json:"-" gorm:"foreignKey:AgentID"`
	PropertyName  *string          `json:"propertyName"`
	AllocatedAt   *time.Time       `json:"allocatedAt,omitempty"`
	ParticipantID *properties.UUID `json:"participantId,omitempty" gorm:"index"`
	Participant   *Participant     `json:"-" gorm:"foreignKey:ParticipantID"`
}

func (ConfigPoolValue) TableName() string {
	return "config_pool_values"
}

func (cv *ConfigPoolValue) Validate() error {
	if cv.Name == "" {
		return fmt.Errorf("config pool value name is required")
	}

	if cv.Value == nil {
		return fmt.Errorf("config pool value is required")
	}

	if cv.ConfigPoolID == (properties.UUID{}) {
		return fmt.Errorf("config pool ID cannot be empty")
	}
	return nil
}

func (cv *ConfigPoolValue) IsAllocated() bool {
	return cv.AgentID != nil
}

func (cv *ConfigPoolValue) Allocate(agentID properties.UUID, propertyName string) {
	allocated := time.Now()
	cv.AgentID = &agentID
	cv.AllocatedAt = &allocated
	cv.PropertyName = &propertyName
}

func (cv *ConfigPoolValue) Release() {
	cv.AgentID = nil
	cv.AllocatedAt = nil
	cv.PropertyName = nil
}

func (cv *ConfigPoolValue) PoolID() properties.UUID {
	return cv.ConfigPoolID
}

func (cv *ConfigPoolValue) RawValue() any {
	return cv.Value
}

type CreateConfigPoolValueParams struct {
	Name         string
	Value        any
	ConfigPoolID properties.UUID
}

func NewConfigPoolValue(c CreateConfigPoolValueParams) *ConfigPoolValue {
	return &ConfigPoolValue{
		Name:         c.Name,
		Value:        c.Value,
		ConfigPoolID: c.ConfigPoolID,
	}
}

type ConfigPoolValueQuerier interface {
	BaseEntityQuerier[ConfigPoolValue]
	CountByPool(ctx context.Context, poolID properties.UUID) (int64, error)
	FindAvailable(ctx context.Context, poolID properties.UUID) ([]*ConfigPoolValue, error)
	FindByAgent(ctx context.Context, agentID properties.UUID) ([]*ConfigPoolValue, error)
}

type ConfigPoolValueRepository interface {
	ConfigPoolValueQuerier
	Create(ctx context.Context, value *ConfigPoolValue) error
	Update(ctx context.Context, value *ConfigPoolValue) error
	Delete(ctx context.Context, id properties.UUID) error
}

type ConfigPoolValueCommander interface {
	Create(ctx context.Context, params CreateConfigPoolValueParams) (*ConfigPoolValue, error)
	Delete(ctx context.Context, id properties.UUID) error
}

type configPoolValueCommander struct {
	store Store
}

func NewConfigPoolValueCommander(store Store) ConfigPoolValueCommander {
	return &configPoolValueCommander{store: store}
}

func (c *configPoolValueCommander) Create(ctx context.Context, params CreateConfigPoolValueParams) (*ConfigPoolValue, error) {
	var poolValue *ConfigPoolValue
	err := c.store.Atomic(ctx, func(s Store) error {
		pool, err := s.ConfigPoolRepo().Get(ctx, params.ConfigPoolID)
		if err != nil {
			if errors.As(err, &NotFoundError{}) {
				return NewNotFoundErrorf("config pool with id %s not found", params.ConfigPoolID)
			}
			return err
		}

		poolValue = NewConfigPoolValue(params)
		poolValue.ParticipantID = pool.ParticipantID
		if err := poolValue.Validate(); err != nil {
			return err
		}

		if err := s.ConfigPoolValueRepo().Create(ctx, poolValue); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeConfigPoolValueCreated, WithInitiatorCtx(ctx), WithConfigPoolValue(poolValue))
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

func (c *configPoolValueCommander) Delete(ctx context.Context, id properties.UUID) error {
	return c.store.Atomic(ctx, func(s Store) error {
		value, err := s.ConfigPoolValueRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		if value.IsAllocated() {
			return NewInvalidInputErrorf("cannot delete allocated pool value")
		}

		eventEntry, err := NewEvent(EventTypeConfigPoolValueDeleted, WithInitiatorCtx(ctx), WithConfigPoolValue(value))
		if err != nil {
			return err
		}
		if err := s.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return s.ConfigPoolValueRepo().Delete(ctx, id)
	})
}
