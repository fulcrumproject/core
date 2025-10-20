// ServicePoolValue entity - individual value records with allocation tracking
package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeServicePoolValueCreated EventType = "service_pool_value.created"
	EventTypeServicePoolValueUpdated EventType = "service_pool_value.updated"
	EventTypeServicePoolValueDeleted EventType = "service_pool_value.deleted"
)

// ServicePoolValue represents an individual allocatable value in a pool
type ServicePoolValue struct {
	BaseEntity

	Name          string          `json:"name" gorm:"not null"`
	Value         any             `json:"value" gorm:"type:jsonb;serializer:json;not null"`
	ServicePoolID properties.UUID `json:"servicePoolId" gorm:"not null;index"`
	ServicePool   *ServicePool    `json:"-" gorm:"foreignKey:ServicePoolID"`

	// Allocation tracking (nullable - null when available)
	ServiceID    *properties.UUID `json:"serviceId,omitempty" gorm:"index"`
	Service      *Service         `json:"-" gorm:"foreignKey:ServiceID"`
	PropertyName *string          `json:"propertyName,omitempty"`
	AllocatedAt  *time.Time       `json:"allocatedAt,omitempty"`
}

// CreateServicePoolValueParams defines parameters for creating a ServicePoolValue
type CreateServicePoolValueParams struct {
	ServicePoolID properties.UUID
	Name          string
	Value         any
}

// NewServicePoolValue creates a new service pool value without validation
func NewServicePoolValue(params CreateServicePoolValueParams) *ServicePoolValue {
	return &ServicePoolValue{
		Name:          params.Name,
		Value:         params.Value,
		ServicePoolID: params.ServicePoolID,
		ServiceID:     nil,
		PropertyName:  nil,
		AllocatedAt:   nil,
	}
}

// TableName returns the table name for the service pool value
func (ServicePoolValue) TableName() string {
	return "service_pool_values"
}

// Validate ensures all ServicePoolValue fields are valid
func (spv *ServicePoolValue) Validate() error {
	if spv.Name == "" {
		return fmt.Errorf("pool value name cannot be empty")
	}
	if spv.Value == nil {
		return fmt.Errorf("pool value cannot be nil")
	}
	if spv.ServicePoolID == (properties.UUID{}) {
		return fmt.Errorf("service pool ID cannot be empty")
	}
	return nil
}

// IsAllocated returns true if this value is currently allocated to a service
func (spv *ServicePoolValue) IsAllocated() bool {
	return spv.ServiceID != nil
}

// Allocate marks this value as allocated to a service
func (spv *ServicePoolValue) Allocate(serviceID properties.UUID, propertyName string) {
	now := time.Now()
	spv.ServiceID = &serviceID
	spv.PropertyName = &propertyName
	spv.AllocatedAt = &now
}

// Release marks this value as available for allocation
func (spv *ServicePoolValue) Release() {
	spv.ServiceID = nil
	spv.PropertyName = nil
	spv.AllocatedAt = nil
}

// ServicePoolValueRepository manages ServicePoolValue entities
type ServicePoolValueRepository interface {
	ServicePoolValueQuerier
	Create(ctx context.Context, value *ServicePoolValue) error
	Update(ctx context.Context, value *ServicePoolValue) error
	Delete(ctx context.Context, id properties.UUID) error
}

// ServicePoolValueQuerier provides read-only access to ServicePoolValue entities
type ServicePoolValueQuerier interface {
	BaseEntityQuerier[ServicePoolValue]

	ListByPool(ctx context.Context, poolID properties.UUID) ([]*ServicePoolValue, error)
	ListByService(ctx context.Context, serviceID properties.UUID) ([]*ServicePoolValue, error)
	FindByPool(ctx context.Context, poolID properties.UUID) ([]*ServicePoolValue, error)
	FindAvailable(ctx context.Context, poolID properties.UUID) ([]*ServicePoolValue, error)
	FindByService(ctx context.Context, serviceID properties.UUID) ([]*ServicePoolValue, error)
}

// ServicePoolValueCommander handles complex ServicePoolValue operations
type ServicePoolValueCommander interface {
	Create(ctx context.Context, params CreateServicePoolValueParams) (*ServicePoolValue, error)
	Delete(ctx context.Context, id properties.UUID) error
}

// servicePoolValueCommander is the concrete implementation of ServicePoolValueCommander
type servicePoolValueCommander struct {
	store Store
}

// NewServicePoolValueCommander creates a new ServicePoolValueCommander
func NewServicePoolValueCommander(store Store) ServicePoolValueCommander {
	return &servicePoolValueCommander{store: store}
}

// Create creates a new service pool value
func (c *servicePoolValueCommander) Create(
	ctx context.Context,
	params CreateServicePoolValueParams,
) (*ServicePoolValue, error) {
	var value *ServicePoolValue
	err := c.store.Atomic(ctx, func(store Store) error {
		// Validate that the service pool exists
		exists, err := store.ServicePoolRepo().Exists(ctx, params.ServicePoolID)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("service pool with id %s not found", params.ServicePoolID)
		}

		// Create the pool value
		value = NewServicePoolValue(params)
		if err := value.Validate(); err != nil {
			return err
		}

		// Save to database
		if err := store.ServicePoolValueRepo().Create(ctx, value); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return value, nil
}

// Delete deletes a service pool value (only if not allocated)
func (c *servicePoolValueCommander) Delete(
	ctx context.Context,
	id properties.UUID,
) error {
	return c.store.Atomic(ctx, func(store Store) error {
		// Get the pool value
		value, err := store.ServicePoolValueRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		// Check if it's allocated
		if value.IsAllocated() {
			return NewInvalidInputErrorf("cannot delete allocated pool value")
		}

		// Delete the pool value
		return store.ServicePoolValueRepo().Delete(ctx, id)
	})
}
