// ServiceOption entity and operations
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

const (
	EventTypeServiceOptionCreated EventType = "service_option.created"
	EventTypeServiceOptionUpdated EventType = "service_option.updated"
	EventTypeServiceOptionDeleted EventType = "service_option.deleted"
)

// ServiceOption represents a provider-specific option value
type ServiceOption struct {
	BaseEntity
	ProviderID          properties.UUID `json:"providerId" gorm:"type:uuid;not null;index:idx_service_option_provider"`
	ServiceOptionTypeID properties.UUID `json:"serviceOptionTypeId" gorm:"type:uuid;not null;index:idx_service_option_type"`
	Name                string          `json:"name" gorm:"not null"`
	Value               any             `json:"value" gorm:"type:jsonb;not null"`
	Enabled             bool            `json:"enabled" gorm:"not null;default:true"`
	DisplayOrder        int             `json:"displayOrder" gorm:"default:0"`
}

// NewServiceOption creates a new service option without validation
func NewServiceOption(params CreateServiceOptionParams) *ServiceOption {
	return &ServiceOption{
		ProviderID:          params.ProviderID,
		ServiceOptionTypeID: params.ServiceOptionTypeID,
		Name:                params.Name,
		Value:               params.Value,
		Enabled:             params.Enabled,
		DisplayOrder:        params.DisplayOrder,
	}
}

// TableName returns the table name for the service option
func (ServiceOption) TableName() string {
	return "service_options"
}

// Validate ensures all ServiceOption fields are valid
func (so *ServiceOption) Validate() error {
	if so.ProviderID == properties.UUID(uuid.Nil) {
		return fmt.Errorf("service option providerId cannot be empty")
	}
	if so.ServiceOptionTypeID == properties.UUID(uuid.Nil) {
		return fmt.Errorf("service option serviceOptionTypeId cannot be empty")
	}
	if so.Name == "" {
		return fmt.Errorf("service option name cannot be empty")
	}
	if so.Value == nil {
		return fmt.Errorf("service option value cannot be nil")
	}
	return nil
}

// Update updates the service option fields if the pointers are non-nil
func (so *ServiceOption) Update(params UpdateServiceOptionParams) {
	if params.Name != nil {
		so.Name = *params.Name
	}
	if params.Value != nil {
		so.Value = *params.Value
	}
	if params.Enabled != nil {
		so.Enabled = *params.Enabled
	}
	if params.DisplayOrder != nil {
		so.DisplayOrder = *params.DisplayOrder
	}
	// ProviderID and ServiceOptionTypeID cannot be updated
}

// ServiceOptionRepository defines the interface for the ServiceOption repository
type ServiceOptionRepository interface {
	ServiceOptionQuerier
	BaseEntityRepository[ServiceOption]

	// CountByServiceOptionType returns the count of options for a given type
	CountByServiceOptionType(ctx context.Context, typeID properties.UUID) (int64, error)
}

// ServiceOptionQuerier defines the interface for the ServiceOption read-only queries
type ServiceOptionQuerier interface {
	BaseEntityQuerier[ServiceOption]

	// FindByProviderAndTypeAndValue retrieves a service option by provider, type, and value
	FindByProviderAndTypeAndValue(ctx context.Context, providerID, typeID properties.UUID, value any) (*ServiceOption, error)

	// ListByProvider retrieves all service options for a provider
	ListByProvider(ctx context.Context, providerID properties.UUID) ([]*ServiceOption, error)

	// ListByProviderAndType retrieves all service options for a provider and type
	ListByProviderAndType(ctx context.Context, providerID, typeID properties.UUID) ([]*ServiceOption, error)

	// ListEnabledByProviderAndType retrieves enabled service options for a provider and type
	ListEnabledByProviderAndType(ctx context.Context, providerID, typeID properties.UUID) ([]*ServiceOption, error)
}

// ServiceOptionCommander defines the interface for the ServiceOption commands
type ServiceOptionCommander interface {
	// Create creates a new service option
	Create(ctx context.Context, params CreateServiceOptionParams) (*ServiceOption, error)

	// Update updates a service option
	Update(ctx context.Context, params UpdateServiceOptionParams) (*ServiceOption, error)

	// Delete removes a service option by ID
	Delete(ctx context.Context, id properties.UUID) error
}

type CreateServiceOptionParams struct {
	ProviderID          properties.UUID `json:"providerId"`
	ServiceOptionTypeID properties.UUID `json:"serviceOptionTypeId"`
	Name                string          `json:"name"`
	Value               any             `json:"value"`
	Enabled             bool            `json:"enabled"`
	DisplayOrder        int             `json:"displayOrder"`
}

type UpdateServiceOptionParams struct {
	ID           properties.UUID `json:"id"`
	Name         *string         `json:"name"`
	Value        *any            `json:"value"`
	Enabled      *bool           `json:"enabled"`
	DisplayOrder *int            `json:"displayOrder"`
}

// serviceOptionCommander is the concrete implementation of ServiceOptionCommander
type serviceOptionCommander struct {
	store Store
}

// NewServiceOptionCommander creates a new ServiceOptionCommander
func NewServiceOptionCommander(store Store) ServiceOptionCommander {
	return &serviceOptionCommander{store: store}
}

// Create creates a new service option
func (c *serviceOptionCommander) Create(
	ctx context.Context,
	params CreateServiceOptionParams,
) (*ServiceOption, error) {
	var option *ServiceOption
	err := c.store.Atomic(ctx, func(store Store) error {
		// Validate that the provider exists
		exists, err := store.ParticipantRepo().Exists(ctx, params.ProviderID)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("provider %s not found", params.ProviderID)
		}

		// Validate that the service option type exists
		exists, err = store.ServiceOptionTypeRepo().Exists(ctx, params.ServiceOptionTypeID)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("service option type %s not found", params.ServiceOptionTypeID)
		}

		option = NewServiceOption(params)
		if err := option.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ServiceOptionRepo().Create(ctx, option); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceOptionCreated, WithInitiatorCtx(ctx), WithServiceOption(option))
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
	return option, nil
}

// Update updates a service option
func (c *serviceOptionCommander) Update(
	ctx context.Context,
	params UpdateServiceOptionParams,
) (*ServiceOption, error) {
	option, err := c.store.ServiceOptionRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy before modifications for event diff
	beforeOption := *option

	// Update and validate
	option.Update(params)
	if err := option.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceOptionRepo().Save(ctx, option); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceOptionUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeOption, option), WithServiceOption(option))
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
	return option, nil
}

// Delete removes a service option by ID
func (c *serviceOptionCommander) Delete(ctx context.Context, id properties.UUID) error {
	option, err := c.store.ServiceOptionRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		eventEntry, err := NewEvent(EventTypeServiceOptionDeleted, WithInitiatorCtx(ctx), WithServiceOption(option))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		if err := store.ServiceOptionRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}
