package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeConfigPoolCreated EventType = "config_pool.created"
	EventTypeConfigPoolUpdated EventType = "config_pool.updated"
	EventTypeConfigPoolDeleted EventType = "config_pool.deleted"
)

type ConfigPool struct {
	BaseEntity
	Name            string            `json:"name" gorm:"not null"`
	Type            string            `json:"type" gorm:"not null"`
	PropertyType    string            `json:"propertyType" gorm:"not null"`
	GeneratorType   PoolGeneratorType `json:"generatorType" gorm:"not null"`
	GeneratorConfig *properties.JSON  `json:"generatorConfig,omitempty" gorm:"type:jsonb"`
	ParticipantID   *properties.UUID  `json:"participantId,omitempty" gorm:"index"`
	Participant     *Participant      `json:"-" gorm:"foreignKey:ParticipantID"`
}

// CreateConfigPoolParams defines parameters for creating an ConfigPool
type CreateConfigPoolParams struct {
	Name            string
	Type            string
	PropertyType    string
	GeneratorType   PoolGeneratorType
	GeneratorConfig *properties.JSON
	ParticipantID   *properties.UUID
}

// UpdateConfigPoolParams defines parameters for updating an ConfigPool
type UpdateConfigPoolParams struct {
	Name            *string
	GeneratorConfig *properties.JSON
}

// NewConfigPool creates a new config pool without validation
func NewConfigPool(params CreateConfigPoolParams) *ConfigPool {
	return &ConfigPool{
		Name:            params.Name,
		Type:            params.Type,
		PropertyType:    params.PropertyType,
		GeneratorType:   params.GeneratorType,
		GeneratorConfig: params.GeneratorConfig,
		ParticipantID:   params.ParticipantID,
	}
}

func (ConfigPool) TableName() string {
	return "config_pools"
}

func (ap *ConfigPool) Validate() error {
	if ap.Name == "" {
		return fmt.Errorf("config pool name cannot be empty")
	}

	if ap.Type == "" {
		return fmt.Errorf("config pool type cannot be empty")
	}

	if !slices.Contains(ValidPoolPropertyTypes, ap.PropertyType) {
		return fmt.Errorf("invalid property type: %s (must be one of: %v)", ap.PropertyType, ValidPoolPropertyTypes)
	}

	if ap.GeneratorType != PoolGeneratorList {
		return fmt.Errorf("invalid generator type for config pool: %s (must be %s)", ap.GeneratorType, PoolGeneratorList)
	}

	return nil
}

// Update modifies the ConfigPool with provided parameters
func (ap *ConfigPool) Update(params UpdateConfigPoolParams) {
	if params.Name != nil {
		ap.Name = *params.Name
	}
	if params.GeneratorConfig != nil {
		ap.GeneratorConfig = params.GeneratorConfig
	}
}

type ConfigPoolQuerier interface {
	BaseEntityQuerier[ConfigPool]
	FindByTypeAndParticipant(ctx context.Context, poolType string, participantID *properties.UUID) (*ConfigPool, error)
}

type ConfigPoolRepository interface {
	ConfigPoolQuerier
	Create(ctx context.Context, pool *ConfigPool) error
	Update(ctx context.Context, pool *ConfigPool) error
	Delete(ctx context.Context, id properties.UUID) error
}

// ConfigPoolCommander handles complex ConfigPool operations
type ConfigPoolCommander interface {
	Create(ctx context.Context, params CreateConfigPoolParams) (*ConfigPool, error)
	Update(ctx context.Context, id properties.UUID, params UpdateConfigPoolParams) (*ConfigPool, error)
	Delete(ctx context.Context, id properties.UUID) error
}

type configPoolCommander struct {
	store Store
}

func NewConfigPoolCommander(store Store) ConfigPoolCommander {
	return &configPoolCommander{store: store}
}

func (c *configPoolCommander) Create(
	ctx context.Context,
	params CreateConfigPoolParams,
) (*ConfigPool, error) {
	var pool *ConfigPool
	err := c.store.Atomic(ctx, func(store Store) error {
		pool = NewConfigPool(params)
		if err := pool.Validate(); err != nil {
			return err
		}

		// Type must be globally unique — the schema engine's pool generator resolves pools
		// by type, so two pools sharing a type silently collide at allocation time.
		existing, err := store.ConfigPoolRepo().FindByTypeAndParticipant(ctx, pool.Type, pool.ParticipantID)
		if err != nil {
			var notFound NotFoundError
			if !errors.As(err, &notFound) {
				return err
			}
		} else if existing != nil {
			return NewInvalidInputErrorf("config pool with type %q already exists for this scope", pool.Type)
		}

		if err := store.ConfigPoolRepo().Create(ctx, pool); err != nil {
			return err
		}

		eventEntity, err := NewEvent(EventTypeConfigPoolCreated, WithInitiatorCtx(ctx), WithConfigPool(pool))
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

func (c *configPoolCommander) Update(
	ctx context.Context,
	id properties.UUID,
	params UpdateConfigPoolParams,
) (*ConfigPool, error) {
	var pool *ConfigPool
	err := c.store.Atomic(ctx, func(store Store) error {
		var err error
		pool, err = store.ConfigPoolRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		beforeConfigPool := *pool

		pool.Update(params)

		if err := pool.Validate(); err != nil {
			return err
		}

		if err := store.ConfigPoolRepo().Update(ctx, pool); err != nil {
			return err
		}

		eventEntity, err := NewEvent(EventTypeConfigPoolUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeConfigPool, pool), WithConfigPool(pool))
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

func (c *configPoolCommander) Delete(
	ctx context.Context,
	id properties.UUID,
) error {
	return c.store.Atomic(ctx, func(store Store) error {
		pool, err := store.ConfigPoolRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		poolValues, err := store.ConfigPoolValueRepo().CountByPool(ctx, pool.ID)
		if err != nil {
			return fmt.Errorf("failed to count values for config pool %s: %w", id, err)
		}

		if poolValues > 0 {
			return NewInvalidInputErrorf("cannot delete config pool %s: %d dependent value(s) exist", id, poolValues)
		}

		eventEntity, err := NewEvent(EventTypeConfigPoolDeleted, WithInitiatorCtx(ctx), WithConfigPool(pool))
		if err != nil {
			return err
		}

		if err := store.EventRepo().Create(ctx, eventEntity); err != nil {
			return err
		}

		if err := store.ConfigPoolRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}
