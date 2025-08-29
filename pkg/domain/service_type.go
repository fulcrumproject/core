package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeServiceTypeCreated EventType = "service_type.created"
	EventTypeServiceTypeUpdated EventType = "service_type.updated"
	EventTypeServiceTypeDeleted EventType = "service_type.deleted"
)

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	BaseEntity
	Name           string         `json:"name" gorm:"not null;unique"`
	PropertySchema *ServiceSchema `json:"propertySchema,omitempty" gorm:"type:jsonb"`
}

// NewServiceType creates a new service type without validation
func NewServiceType(params CreateServiceTypeParams) *ServiceType {
	return &ServiceType{
		Name:           params.Name,
		PropertySchema: params.PropertySchema,
	}
}

// TableName returns the table name for the service type
func (ServiceType) TableName() string {
	return "service_types"
}

// Validate ensures all ServiceType fields are valid
func (st *ServiceType) Validate() error {
	if st.Name == "" {
		return fmt.Errorf("service type name cannot be empty")
	}
	return nil
}

// Update updates the service type fields if the pointers are non-nil
func (st *ServiceType) Update(params UpdateServiceTypeParams) {
	if params.Name != nil {
		st.Name = *params.Name
	}
	if params.PropertySchema != nil {
		st.PropertySchema = params.PropertySchema
	}
}

// ServiceTypeRepository defines the interface for the ServiceType repository
type ServiceTypeRepository interface {
	ServiceTypeQuerier
	BaseEntityRepository[ServiceType]
}

// ServiceTypeQuerier defines the interface for the ServiceType read-only queries
type ServiceTypeQuerier interface {
	BaseEntityQuerier[ServiceType]
}

// ServiceTypeCommander defines the interface for the ServiceType commands
type ServiceTypeCommander interface {
	// Create creates a new service type
	Create(ctx context.Context, params CreateServiceTypeParams) (*ServiceType, error)

	// Update updates a service type
	Update(ctx context.Context, params UpdateServiceTypeParams) (*ServiceType, error)

	// Delete removes a service type by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error

	// ValidateServiceProperties validates properties against a service type's schema
	ValidateServiceProperties(ctx context.Context, params *ServicePropertyValidationParams) (map[string]any, error)
}

// ServicePropertyValidationParams provides the parameters for validating service properties
type ServicePropertyValidationParams struct {
	ServiceTypeID properties.UUID
	GroupID       properties.UUID
	Properties    map[string]any
}

type CreateServiceTypeParams struct {
	Name           string         `json:"name"`
	PropertySchema *ServiceSchema `json:"propertySchema,omitempty"`
}

type UpdateServiceTypeParams struct {
	ID             properties.UUID `json:"id"`
	Name           *string         `json:"name"`
	PropertySchema *ServiceSchema  `json:"propertySchema,omitempty"`
}

// serviceTypeCommander is the concrete implementation of ServiceTypeCommander
type serviceTypeCommander struct {
	store Store
}

// NewServiceTypeCommander creates a new ServiceTypeCommander
func NewServiceTypeCommander(store Store) ServiceTypeCommander {
	return &serviceTypeCommander{store: store}
}

// Create creates a new service type
func (c *serviceTypeCommander) Create(
	ctx context.Context,
	params CreateServiceTypeParams,
) (*ServiceType, error) {
	var serviceType *ServiceType
	err := c.store.Atomic(ctx, func(store Store) error {
		serviceType = NewServiceType(params)
		if err := serviceType.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ServiceTypeRepo().Create(ctx, serviceType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceTypeCreated, WithInitiatorCtx(ctx), WithServiceType(serviceType))
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
	return serviceType, nil
}

// Update updates a service type
func (c *serviceTypeCommander) Update(
	ctx context.Context,
	params UpdateServiceTypeParams,
) (*ServiceType, error) {
	serviceType, err := c.store.ServiceTypeRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy of the service type before modifications for event diff
	beforeServiceType := *serviceType

	// Update and validate
	serviceType.Update(params)
	if err := serviceType.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceTypeRepo().Save(ctx, serviceType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceTypeUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeServiceType, serviceType), WithServiceType(serviceType))
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
	return serviceType, nil
}

// Delete removes a service type by ID after checking for dependencies
func (c *serviceTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	serviceType, err := c.store.ServiceTypeRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		// Check for dependent Services
		serviceCount, err := store.ServiceRepo().CountByServiceType(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count services for service type %s: %w", id, err)
		}
		if serviceCount > 0 {
			return NewInvalidInputErrorf("cannot delete service type %s: %d dependent service(s) exist", id, serviceCount)
		}

		eventEntry, err := NewEvent(EventTypeServiceTypeDeleted, WithInitiatorCtx(ctx), WithServiceType(serviceType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		if err := store.ServiceTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}

// ValidateServiceProperties validates the properties against the service type schema
func (c *serviceTypeCommander) ValidateServiceProperties(ctx context.Context, params *ServicePropertyValidationParams) (map[string]any, error) {
	return ValidateServiceProperties(ctx, c.store, params)
}

// ValidateProperties validates the properties against the service type schema
func ValidateServiceProperties(ctx context.Context, store Store, params *ServicePropertyValidationParams) (map[string]any, error) {
	// Fetch the service type to get its schema
	serviceType, err := store.ServiceTypeRepo().Get(ctx, params.ServiceTypeID)
	if err != nil {
		return nil, err
	}

	// If no schema, return properties as-is
	if serviceType.PropertySchema == nil {
		return params.Properties, nil
	}

	// Apply defaults to properties
	propertiesWithDefaults := applyServicePropertiesDefaults(params.Properties, *serviceType.PropertySchema)

	// Create validation context
	validationCtx := &ServicePropertyValidationCtx{
		Context:    ctx,
		Store:      store,
		Schema:     *serviceType.PropertySchema,
		GroupID:    params.GroupID,
		Properties: propertiesWithDefaults,
	}

	// Validate properties against schema
	validationErrors, err := validateServiceProperties(validationCtx)
	if err != nil {
		return nil, err
	}
	if len(validationErrors) > 0 {
		return nil, NewValidationError(validationErrors)
	}

	return propertiesWithDefaults, nil
}
