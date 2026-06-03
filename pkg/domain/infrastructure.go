package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	EventTypeInfrastructureCreated EventType = "infrastructure.created"
	EventTypeInfrastructureUpdated EventType = "infrastructure.updated"
	EventTypeInfrastructureDeleted EventType = "infrastructure.deleted"
)

// Infrastructure is an instance of an InfrastructureType — the substrate
// (network, gateway, VPN, …) an Agent is deployed onto.
type Infrastructure struct {
	BaseEntity

	Name string `json:"name" gorm:"not null"`

	// Tags representing capabilities/labels of this infrastructure
	Tags pq.StringArray `json:"tags" gorm:"type:text[]"`

	// Configuration stores instance-specific configuration parameters as JSON
	Configuration *properties.JSON `json:"configuration,omitempty" gorm:"type:jsonb"`

	// Relationships
	InfrastructureTypeID properties.UUID     `json:"infrastructureTypeId" gorm:"not null"`
	InfrastructureType   *InfrastructureType `json:"infrastructureType,omitempty" gorm:"foreignKey:InfrastructureTypeID"`
	ProviderID           properties.UUID     `json:"providerId" gorm:"not null"`
	Provider             *Participant        `json:"-" gorm:"foreignKey:ProviderID"`
}

// NewInfrastructure creates a new infrastructure without persistence-level validation.
func NewInfrastructure(params CreateInfrastructureParams) *Infrastructure {
	return &Infrastructure{
		Name:                 params.Name,
		ProviderID:           params.ProviderID,
		InfrastructureTypeID: params.InfrastructureTypeID,
		Tags:                 pq.StringArray(params.Tags),
		Configuration:        params.Configuration,
	}
}

// TableName returns the table name for the infrastructure
func (Infrastructure) TableName() string {
	return "infrastructures"
}

// Validate ensures all infrastructure fields are valid
func (i *Infrastructure) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("infrastructure name cannot be empty")
	}
	if i.InfrastructureTypeID == uuid.Nil {
		return fmt.Errorf("infrastructure type ID cannot be empty")
	}
	if i.ProviderID == uuid.Nil {
		return fmt.Errorf("provider ID cannot be empty")
	}
	for idx, tag := range []string(i.Tags) {
		if len(tag) == 0 {
			return fmt.Errorf("tag at index %d cannot be empty", idx)
		}
		if len(tag) > 100 {
			return fmt.Errorf("tag at index %d exceeds maximum length of 100 characters", idx)
		}
	}
	return nil
}

// Update mutates the infrastructure's mutable fields when the pointer args are non-nil.
func (i *Infrastructure) Update(params UpdateInfrastructureParams) {
	if params.Name != nil {
		i.Name = *params.Name
	}
	if params.Tags != nil {
		i.Tags = pq.StringArray(*params.Tags)
	}
	if params.Configuration != nil {
		i.Configuration = params.Configuration
	}
}

// InfrastructureCommander defines the command operations for Infrastructure.
type InfrastructureCommander interface {
	Create(ctx context.Context, params CreateInfrastructureParams) (*Infrastructure, error)
	Update(ctx context.Context, params UpdateInfrastructureParams) (*Infrastructure, error)
	Delete(ctx context.Context, id properties.UUID) error
}

type CreateInfrastructureParams struct {
	Name                 string           `json:"name"`
	ProviderID           properties.UUID  `json:"providerId"`
	InfrastructureTypeID properties.UUID  `json:"infrastructureTypeId"`
	Tags                 []string         `json:"tags"`
	Configuration        *properties.JSON `json:"configuration,omitempty"`
}

type UpdateInfrastructureParams struct {
	ID            properties.UUID  `json:"id"`
	Name          *string          `json:"name,omitempty"`
	Tags          *[]string        `json:"tags,omitempty"`
	Configuration *properties.JSON `json:"configuration,omitempty"`
}

// infrastructureCommander is the concrete implementation of InfrastructureCommander.
type infrastructureCommander struct {
	store        Store
	configEngine *schema.Engine[InfrastructureConfigContext]
}

// NewInfrastructureCommander creates a new InfrastructureCommander.
func NewInfrastructureCommander(
	store Store,
	configEngine *schema.Engine[InfrastructureConfigContext],
) InfrastructureCommander {
	return &infrastructureCommander{
		store:        store,
		configEngine: configEngine,
	}
}

func (c *infrastructureCommander) Create(
	ctx context.Context,
	params CreateInfrastructureParams,
) (*Infrastructure, error) {
	providerExists, err := c.store.ParticipantRepo().Exists(ctx, params.ProviderID)
	if err != nil {
		return nil, err
	}
	if !providerExists {
		return nil, NewInvalidInputErrorf("provider with ID %s does not exist", params.ProviderID)
	}

	infraType, err := c.store.InfrastructureTypeRepo().Get(ctx, params.InfrastructureTypeID)
	if err != nil {
		var nfe NotFoundError
		if errors.As(err, &nfe) {
			return nil, NewInvalidInputErrorf("infrastructure type with ID %s does not exist", params.InfrastructureTypeID)
		}
		return nil, err
	}

	// Pre-generate the infrastructure ID so future pool generators can stamp
	// allocations with it inside the same transaction as the insert.
	infraID := properties.UUID(uuid.New())

	var infra *Infrastructure
	err = c.store.Atomic(ctx, func(store Store) error {
		infra = NewInfrastructure(params)
		infra.ID = infraID

		if infra.Configuration != nil {
			schemaCtx := InfrastructureConfigContext{
				Store:                    store,
				InfrastructureID:         &infraID,
				InfrastructureProviderID: infra.ProviderID,
			}
			configMap := map[string]any(*infra.Configuration)
			processed, err := c.configEngine.ApplyCreate(
				ctx,
				schemaCtx,
				infraType.ConfigurationSchema,
				configMap,
			)
			if err != nil {
				return InvalidInputError{Err: fmt.Errorf("configuration: %w", err)}
			}
			processedJSON := properties.JSON(processed)
			infra.Configuration = &processedJSON
		}

		if err := infra.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.InfrastructureRepo().Create(ctx, infra); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeInfrastructureCreated, WithInitiatorCtx(ctx), WithInfrastructure(infra))
		if err != nil {
			return err
		}
		return store.EventRepo().Create(ctx, eventEntry)
	})
	if err != nil {
		return nil, err
	}
	return infra, nil
}

func (c *infrastructureCommander) Update(
	ctx context.Context,
	params UpdateInfrastructureParams,
) (*Infrastructure, error) {
	infra, err := c.store.InfrastructureRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	beforeInfra := *infra

	infra.Update(params)

	err = c.store.Atomic(ctx, func(store Store) error {
		if params.Configuration != nil && infra.Configuration != nil {
			infraType, err := store.InfrastructureTypeRepo().Get(ctx, infra.InfrastructureTypeID)
			if err != nil {
				return err
			}
			schemaCtx := InfrastructureConfigContext{
				Store:                    store,
				InfrastructureID:         &infra.ID,
				InfrastructureProviderID: infra.ProviderID,
			}

			var oldConfigMap map[string]any
			if beforeInfra.Configuration != nil {
				oldConfigMap = map[string]any(*beforeInfra.Configuration)
			}
			newConfigMap := map[string]any(*infra.Configuration)

			processed, err := c.configEngine.ApplyUpdate(
				ctx,
				schemaCtx,
				infraType.ConfigurationSchema,
				oldConfigMap,
				newConfigMap,
			)
			if err != nil {
				return InvalidInputError{Err: fmt.Errorf("configuration: %w", err)}
			}
			processedJSON := properties.JSON(processed)
			infra.Configuration = &processedJSON
		}

		if err := infra.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.InfrastructureRepo().Save(ctx, infra); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeInfrastructureUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeInfra, infra), WithInfrastructure(infra))
		if err != nil {
			return err
		}
		return store.EventRepo().Create(ctx, eventEntry)
	})
	if err != nil {
		return nil, err
	}
	return infra, nil
}

// Delete removes an infrastructure by ID. Blocks if any Agent still references it.
func (c *infrastructureCommander) Delete(ctx context.Context, id properties.UUID) error {
	infra, err := c.store.InfrastructureRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		agentCount, err := store.AgentRepo().CountByInfrastructure(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count agents for infrastructure %s: %w", id, err)
		}
		if agentCount > 0 {
			return NewInvalidInputErrorf("cannot delete infrastructure %s: %d dependent agent(s) exist", id, agentCount)
		}

		// Release any ConfigPoolValue rows allocated to this infrastructure. Dispatched per pool via
		// the factory so release semantics stay consistent across generator types, mirroring agent Delete.
		allocated, err := store.ConfigPoolValueRepo().FindByInfrastructure(ctx, id)
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

		eventEntry, err := NewEvent(EventTypeInfrastructureDeleted, WithInitiatorCtx(ctx), WithInfrastructure(infra))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return store.InfrastructureRepo().Delete(ctx, id)
	})
}

// InfrastructureRepository defines the persistence operations for Infrastructure.
type InfrastructureRepository interface {
	InfrastructureQuerier
	BaseEntityRepository[Infrastructure]
}

// InfrastructureQuerier defines the read-only operations for Infrastructure.
type InfrastructureQuerier interface {
	BaseEntityQuerier[Infrastructure]

	// CountByInfrastructureType returns the number of infrastructures bound to a specific type
	CountByInfrastructureType(ctx context.Context, infrastructureTypeID properties.UUID) (int64, error)
}
