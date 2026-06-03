package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

const (
	EventTypeInfrastructureTypeCreated EventType = "infrastructure_type.created"
	EventTypeInfrastructureTypeUpdated EventType = "infrastructure_type.updated"
	EventTypeInfrastructureTypeDeleted EventType = "infrastructure_type.deleted"
)

// InfrastructureType declares the schema and templates for a class of
// Infrastructure instances (the substrate Agents are deployed onto).
type InfrastructureType struct {
	BaseEntity
	Name string `json:"name" gorm:"not null;unique"`
	TemplateValidation
}

// NewInfrastructureType creates a new infrastructure type without validation.
func NewInfrastructureType(params CreateInfrastructureTypeParams) *InfrastructureType {
	configContentType := params.ConfigContentType
	if configContentType == "" {
		configContentType = "text/plain"
	}

	return &InfrastructureType{
		Name: params.Name,
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: params.ConfigurationSchema,
			ConfigTemplate:      params.ConfigTemplate,
			CmdTemplate:         params.CmdTemplate,
			ConfigContentType:   configContentType,
		},
	}
}

func (InfrastructureType) TableName() string {
	return "infrastructure_types"
}

// Validate ensures all InfrastructureType fields are valid (without schema validation).
func (it *InfrastructureType) Validate() error {
	if it.Name == "" {
		return fmt.Errorf("infrastructure type name cannot be empty")
	}
	return it.validateTemplates()
}

// ValidateWithEngine validates the infrastructure type including its configuration schema.
func (it *InfrastructureType) ValidateWithEngine(engine *schema.Engine[InfrastructureConfigContext]) error {
	if it.Name == "" {
		return fmt.Errorf("infrastructure type name cannot be empty")
	}

	if err := engine.ValidateSchema(it.ConfigurationSchema); err != nil {
		return fmt.Errorf("configurationSchema: %w", err)
	}

	return it.validateTemplates()
}

// Update updates the infrastructure type fields if the pointers are non-nil.
func (it *InfrastructureType) Update(params UpdateInfrastructureTypeParams) {
	if params.Name != nil {
		it.Name = *params.Name
	}
	if params.ConfigurationSchema != nil {
		it.ConfigurationSchema = *params.ConfigurationSchema
	}
	if params.ConfigTemplate != nil {
		it.ConfigTemplate = *params.ConfigTemplate
	}
	if params.CmdTemplate != nil {
		it.CmdTemplate = *params.CmdTemplate
	}
	if params.ConfigContentType != nil {
		it.ConfigContentType = *params.ConfigContentType
		if it.ConfigContentType == "" {
			it.ConfigContentType = "text/plain"
		}
	}
}

type CreateInfrastructureTypeParams struct {
	Name                string        `json:"name"`
	ConfigurationSchema schema.Schema `json:"configurationSchema"`
	ConfigTemplate      string        `json:"configTemplate"`
	CmdTemplate         string        `json:"cmdTemplate"`
	ConfigContentType   string        `json:"configContentType"`
}

type UpdateInfrastructureTypeParams struct {
	ID                  properties.UUID `json:"id"`
	Name                *string         `json:"name"`
	ConfigurationSchema *schema.Schema  `json:"configurationSchema"`
	ConfigTemplate      *string         `json:"configTemplate"`
	CmdTemplate         *string         `json:"cmdTemplate"`
	ConfigContentType   *string         `json:"configContentType"`
}

type InfrastructureTypeCommander interface {
	Create(ctx context.Context, params CreateInfrastructureTypeParams) (*InfrastructureType, error)
	Update(ctx context.Context, params UpdateInfrastructureTypeParams) (*InfrastructureType, error)
	Delete(ctx context.Context, id properties.UUID) error
}

// infrastructureTypeCommander is the concrete implementation of InfrastructureTypeCommander.
type infrastructureTypeCommander struct {
	store        Store
	configEngine *schema.Engine[InfrastructureConfigContext]
}

// NewInfrastructureTypeCommander creates a new InfrastructureTypeCommander.
func NewInfrastructureTypeCommander(
	store Store,
	configEngine *schema.Engine[InfrastructureConfigContext],
) InfrastructureTypeCommander {
	return &infrastructureTypeCommander{
		store:        store,
		configEngine: configEngine,
	}
}

// Create creates a new infrastructure type.
func (c *infrastructureTypeCommander) Create(
	ctx context.Context,
	params CreateInfrastructureTypeParams,
) (*InfrastructureType, error) {
	var infraType *InfrastructureType
	err := c.store.Atomic(ctx, func(store Store) error {
		infraType = NewInfrastructureType(params)

		if err := infraType.ValidateWithEngine(c.configEngine); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.InfrastructureTypeRepo().Create(ctx, infraType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeInfrastructureTypeCreated, WithInitiatorCtx(ctx), WithInfrastructureType(infraType))
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
	return infraType, nil
}

// Update updates an infrastructure type.
func (c *infrastructureTypeCommander) Update(
	ctx context.Context,
	params UpdateInfrastructureTypeParams,
) (*InfrastructureType, error) {
	infraType, err := c.store.InfrastructureTypeRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy of the infrastructure type before modifications for event diff
	beforeInfraType := *infraType

	// Update and validate
	infraType.Update(params)
	if err := infraType.ValidateWithEngine(c.configEngine); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.InfrastructureTypeRepo().Save(ctx, infraType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeInfrastructureTypeUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeInfraType, infraType), WithInfrastructureType(infraType))
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
	return infraType, nil
}

// Delete removes an infrastructure type by ID. Blocks if any AgentType still references it.
func (c *infrastructureTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	infraType, err := c.store.InfrastructureTypeRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		infraCount, err := store.InfrastructureRepo().CountByInfrastructureType(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count infrastructures for infrastructure type %s: %w", id, err)
		}
		if infraCount > 0 {
			return NewInvalidInputErrorf("cannot delete infrastructure type %s: %d dependent infrastructure(s) exist", id, infraCount)
		}

		atCount, err := store.AgentTypeRepo().CountByInfrastructureType(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count agent types for infrastructure type %s: %w", id, err)
		}
		if atCount > 0 {
			return NewInvalidInputErrorf("cannot delete infrastructure type %s: %d dependent agent type(s) exist", id, atCount)
		}

		eventEntry, err := NewEvent(EventTypeInfrastructureTypeDeleted, WithInitiatorCtx(ctx), WithInfrastructureType(infraType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		if err := store.InfrastructureTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}

// InfrastructureTypeRepository defines the interface for the InfrastructureType repository.
type InfrastructureTypeRepository interface {
	InfrastructureTypeQuerier
	BaseEntityRepository[InfrastructureType]
}

// InfrastructureTypeQuerier defines the interface for the InfrastructureType read-only queries.
type InfrastructureTypeQuerier interface {
	BaseEntityQuerier[InfrastructureType]
}
