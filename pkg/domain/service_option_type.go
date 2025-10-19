// ServiceOptionType entity and operations
package domain

import (
	"context"
	"fmt"
	"regexp"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeServiceOptionTypeCreated EventType = "service_option_type.created"
	EventTypeServiceOptionTypeUpdated EventType = "service_option_type.updated"
	EventTypeServiceOptionTypeDeleted EventType = "service_option_type.deleted"
)

// ServiceOptionType represents a global category of service options
type ServiceOptionType struct {
	BaseEntity
	Name        string `json:"name" gorm:"not null"`
	Type        string `json:"type" gorm:"not null;unique"`
	Description string `json:"description"`
}

// NewServiceOptionType creates a new service option type without validation
func NewServiceOptionType(params CreateServiceOptionTypeParams) *ServiceOptionType {
	return &ServiceOptionType{
		Name:        params.Name,
		Type:        params.Type,
		Description: params.Description,
	}
}

// TableName returns the table name for the service option type
func (ServiceOptionType) TableName() string {
	return "service_option_types"
}

// Validate ensures all ServiceOptionType fields are valid
func (sot *ServiceOptionType) Validate() error {
	if sot.Name == "" {
		return fmt.Errorf("service option type name cannot be empty")
	}
	if sot.Type == "" {
		return fmt.Errorf("service option type type cannot be empty")
	}

	// Type must be alphanumeric and underscores only
	validType := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validType.MatchString(sot.Type) {
		return fmt.Errorf("service option type type must contain only alphanumeric characters and underscores")
	}

	return nil
}

// Update updates the service option type fields if the pointers are non-nil
func (sot *ServiceOptionType) Update(params UpdateServiceOptionTypeParams) {
	if params.Name != nil {
		sot.Name = *params.Name
	}
	if params.Description != nil {
		sot.Description = *params.Description
	}
	// Type cannot be updated
}

// ServiceOptionTypeRepository defines the interface for the ServiceOptionType repository
type ServiceOptionTypeRepository interface {
	ServiceOptionTypeQuerier
	BaseEntityRepository[ServiceOptionType]
}

// ServiceOptionTypeQuerier defines the interface for the ServiceOptionType read-only queries
type ServiceOptionTypeQuerier interface {
	BaseEntityQuerier[ServiceOptionType]

	// FindByType retrieves a service option type by type
	FindByType(ctx context.Context, typeStr string) (*ServiceOptionType, error)
}

// ServiceOptionTypeCommander defines the interface for the ServiceOptionType commands
type ServiceOptionTypeCommander interface {
	// Create creates a new service option type
	Create(ctx context.Context, params CreateServiceOptionTypeParams) (*ServiceOptionType, error)

	// Update updates a service option type
	Update(ctx context.Context, params UpdateServiceOptionTypeParams) (*ServiceOptionType, error)

	// Delete removes a service option type by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error
}

type CreateServiceOptionTypeParams struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type UpdateServiceOptionTypeParams struct {
	ID          properties.UUID `json:"id"`
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
}

// serviceOptionTypeCommander is the concrete implementation of ServiceOptionTypeCommander
type serviceOptionTypeCommander struct {
	store Store
}

// NewServiceOptionTypeCommander creates a new ServiceOptionTypeCommander
func NewServiceOptionTypeCommander(store Store) ServiceOptionTypeCommander {
	return &serviceOptionTypeCommander{store: store}
}

// Create creates a new service option type
func (c *serviceOptionTypeCommander) Create(
	ctx context.Context,
	params CreateServiceOptionTypeParams,
) (*ServiceOptionType, error) {
	var optionType *ServiceOptionType
	err := c.store.Atomic(ctx, func(store Store) error {
		optionType = NewServiceOptionType(params)
		if err := optionType.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ServiceOptionTypeRepo().Create(ctx, optionType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceOptionTypeCreated, WithInitiatorCtx(ctx), WithServiceOptionType(optionType))
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
	return optionType, nil
}

// Update updates a service option type
func (c *serviceOptionTypeCommander) Update(
	ctx context.Context,
	params UpdateServiceOptionTypeParams,
) (*ServiceOptionType, error) {
	optionType, err := c.store.ServiceOptionTypeRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy before modifications for event diff
	beforeOptionType := *optionType

	// Update and validate
	optionType.Update(params)
	if err := optionType.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceOptionTypeRepo().Save(ctx, optionType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceOptionTypeUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeOptionType, optionType), WithServiceOptionType(optionType))
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
	return optionType, nil
}

// Delete removes a service option type by ID after checking for dependencies
func (c *serviceOptionTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	optionType, err := c.store.ServiceOptionTypeRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		// Check for dependent ServiceOptions
		optionCount, err := store.ServiceOptionRepo().CountByServiceOptionType(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count service options for type %s: %w", id, err)
		}
		if optionCount > 0 {
			return NewInvalidInputErrorf("cannot delete service option type %s: %d dependent service option(s) exist", id, optionCount)
		}

		eventEntry, err := NewEvent(EventTypeServiceOptionTypeDeleted, WithInitiatorCtx(ctx), WithServiceOptionType(optionType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		if err := store.ServiceOptionTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}

