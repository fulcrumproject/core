package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

const (
	EventTypeServiceTypeCreated EventType = "service_type.created"
	EventTypeServiceTypeUpdated EventType = "service_type.updated"
	EventTypeServiceTypeDeleted EventType = "service_type.deleted"
)

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	BaseEntity
	Name            string          `json:"name" gorm:"not null;unique"`
	PropertySchema  schema.Schema   `json:"propertySchema" gorm:"type:jsonb;not null"`
	LifecycleSchema LifecycleSchema `json:"lifecycleSchema" gorm:"type:jsonb;not null"`
}

// NewServiceType creates a new service type without validation
func NewServiceType(params CreateServiceTypeParams) *ServiceType {
	return &ServiceType{
		Name:            params.Name,
		PropertySchema:  params.PropertySchema,
		LifecycleSchema: params.LifecycleSchema,
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

	// Validate lifecycle schema
	if err := st.LifecycleSchema.Validate(); err != nil {
		return fmt.Errorf("lifecycle schema validation failed: %w", err)
	}

	return nil
}

// Update updates the service type fields if the pointers are non-nil
func (st *ServiceType) Update(params UpdateServiceTypeParams) {
	if params.Name != nil {
		st.Name = *params.Name
	}
	if params.PropertySchema != nil {
		st.PropertySchema = *params.PropertySchema
	}
	if params.LifecycleSchema != nil {
		st.LifecycleSchema = *params.LifecycleSchema
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
}

type CreateServiceTypeParams struct {
	Name            string          `json:"name"`
	PropertySchema  schema.Schema   `json:"propertySchema"`
	LifecycleSchema LifecycleSchema `json:"lifecycleSchema"`
}

type UpdateServiceTypeParams struct {
	ID              properties.UUID  `json:"id"`
	Name            *string          `json:"name"`
	PropertySchema  *schema.Schema   `json:"propertySchema,omitempty"`
	LifecycleSchema *LifecycleSchema `json:"lifecycleSchema,omitempty"`
}

// serviceTypeCommander is the concrete implementation of ServiceTypeCommander
type serviceTypeCommander struct {
	store  Store
	engine *schema.Engine[ServicePropertyContext]
}

// NewServiceTypeCommander creates a new ServiceTypeCommander
func NewServiceTypeCommander(store Store, engine *schema.Engine[ServicePropertyContext]) ServiceTypeCommander {
	return &serviceTypeCommander{
		store:  store,
		engine: engine,
	}
}

// Create creates a new service type
func (c *serviceTypeCommander) Create(
	ctx context.Context,
	params CreateServiceTypeParams,
) (*ServiceType, error) {
	var serviceType *ServiceType
	err := c.store.Atomic(ctx, func(store Store) error {
		serviceType = NewServiceType(params)

		// Validate property schema using engine
		if err := c.engine.ValidateSchema(serviceType.PropertySchema); err != nil {
			return InvalidInputError{Err: fmt.Errorf("invalid property schema: %w", err)}
		}

		// Validate service type (includes lifecycle validation)
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

	// Update
	serviceType.Update(params)

	// Validate property schema using engine
	if err := c.engine.ValidateSchema(serviceType.PropertySchema); err != nil {
		return nil, InvalidInputError{Err: fmt.Errorf("invalid property schema: %w", err)}
	}

	// Validate service type (includes lifecycle validation)
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
