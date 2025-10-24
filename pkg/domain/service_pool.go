// ServicePool entity - type-specific pool with generator configuration
package domain

import (
	"context"
	"fmt"
	"slices"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeServicePoolCreated EventType = "service_pool.created"
	EventTypeServicePoolUpdated EventType = "service_pool.updated"
	EventTypeServicePoolDeleted EventType = "service_pool.deleted"
)

// PoolGeneratorType represents the type of pool generator
type PoolGeneratorType string

const (
	PoolGeneratorList   PoolGeneratorType = "list"
	PoolGeneratorSubnet PoolGeneratorType = "subnet"
)

// Validate checks if the pool generator type is valid
func (t PoolGeneratorType) Validate() error {
	switch t {
	case PoolGeneratorList, PoolGeneratorSubnet:
		return nil
	default:
		return fmt.Errorf("invalid generator type: %s", t)
	}
}

// ServicePool represents a type-specific resource pool
type ServicePool struct {
	BaseEntity

	Name             string            `json:"name" gorm:"not null"`
	Type             string            `json:"type" gorm:"not null"`
	PropertyType     string            `json:"propertyType" gorm:"not null"`
	GeneratorType    PoolGeneratorType `json:"generatorType" gorm:"not null"`
	GeneratorConfig  *properties.JSON  `json:"generatorConfig,omitempty" gorm:"type:jsonb"`
	ServicePoolSetID properties.UUID   `json:"servicePoolSetId" gorm:"not null;index"`
	ServicePoolSet   *ServicePoolSet   `json:"-" gorm:"foreignKey:ServicePoolSetID"`
}

// CreateServicePoolParams defines parameters for creating a ServicePool
type CreateServicePoolParams struct {
	ServicePoolSetID properties.UUID
	Name             string
	Type             string
	PropertyType     string
	GeneratorType    PoolGeneratorType
	GeneratorConfig  *properties.JSON
}

// UpdateServicePoolParams defines parameters for updating a ServicePool
type UpdateServicePoolParams struct {
	Name            *string
	GeneratorConfig *properties.JSON
}

// NewServicePool creates a new service pool without validation
func NewServicePool(params CreateServicePoolParams) *ServicePool {
	return &ServicePool{
		Name:             params.Name,
		Type:             params.Type,
		PropertyType:     params.PropertyType,
		GeneratorType:    params.GeneratorType,
		GeneratorConfig:  params.GeneratorConfig,
		ServicePoolSetID: params.ServicePoolSetID,
	}
}

// TableName returns the table name for the service pool
func (ServicePool) TableName() string {
	return "service_pools"
}

// Validate ensures all ServicePool fields are valid
func (sp *ServicePool) Validate() error {
	if sp.Name == "" {
		return fmt.Errorf("pool name cannot be empty")
	}
	if sp.Type == "" {
		return fmt.Errorf("pool type cannot be empty")
	}

	// Validate PropertyType
	validTypes := []string{"string", "integer", "number", "boolean", "json"}
	if !slices.Contains(validTypes, sp.PropertyType) {
		return fmt.Errorf("invalid property type: %s (must be one of: %v)", sp.PropertyType, validTypes)
	}

	if err := sp.GeneratorType.Validate(); err != nil {
		return err
	}
	if sp.ServicePoolSetID == (properties.UUID{}) {
		return fmt.Errorf("service pool set ID cannot be empty")
	}
	return nil
}

// Update modifies the ServicePool with provided parameters
func (sp *ServicePool) Update(params UpdateServicePoolParams) {
	if params.Name != nil {
		sp.Name = *params.Name
	}
	if params.GeneratorConfig != nil {
		sp.GeneratorConfig = params.GeneratorConfig
	}
}

// ServicePoolRepository manages ServicePool entities
type ServicePoolRepository interface {
	ServicePoolQuerier
	Create(ctx context.Context, pool *ServicePool) error
	Update(ctx context.Context, pool *ServicePool) error
	Delete(ctx context.Context, id properties.UUID) error
}

// ServicePoolQuerier provides read-only access to ServicePool entities
type ServicePoolQuerier interface {
	BaseEntityQuerier[ServicePool]

	ListByPoolSet(ctx context.Context, poolSetID properties.UUID) ([]*ServicePool, error)
	FindByPoolSetAndType(ctx context.Context, poolSetID properties.UUID, poolType string) (*ServicePool, error)
}

// ServicePoolCommander handles complex ServicePool operations
type ServicePoolCommander interface {
	Create(ctx context.Context, params CreateServicePoolParams) (*ServicePool, error)
	Update(ctx context.Context, id properties.UUID, params UpdateServicePoolParams) (*ServicePool, error)
	Delete(ctx context.Context, id properties.UUID) error
}

// servicePoolCommander is the concrete implementation of ServicePoolCommander
type servicePoolCommander struct {
	store Store
}

// NewServicePoolCommander creates a new ServicePoolCommander
func NewServicePoolCommander(store Store) ServicePoolCommander {
	return &servicePoolCommander{store: store}
}

// Create creates a new service pool
func (c *servicePoolCommander) Create(
	ctx context.Context,
	params CreateServicePoolParams,
) (*ServicePool, error) {
	var pool *ServicePool
	err := c.store.Atomic(ctx, func(store Store) error {
		// Validate that the pool set exists
		exists, err := store.ServicePoolSetRepo().Exists(ctx, params.ServicePoolSetID)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("service pool set with id %s not found", params.ServicePoolSetID)
		}

		// Create the pool
		pool = NewServicePool(params)
		if err := pool.Validate(); err != nil {
			return err
		}

		// Save to database
		if err := store.ServicePoolRepo().Create(ctx, pool); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

// Update updates an existing service pool
func (c *servicePoolCommander) Update(
	ctx context.Context,
	id properties.UUID,
	params UpdateServicePoolParams,
) (*ServicePool, error) {
	var pool *ServicePool
	err := c.store.Atomic(ctx, func(store Store) error {
		// Get the existing pool
		var err error
		pool, err = store.ServicePoolRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		// Update fields
		pool.Update(params)

		// Validate
		if err := pool.Validate(); err != nil {
			return err
		}

		// Save changes
		if err := store.ServicePoolRepo().Update(ctx, pool); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

// Delete deletes a service pool
func (c *servicePoolCommander) Delete(
	ctx context.Context,
	id properties.UUID,
) error {
	return c.store.Atomic(ctx, func(store Store) error {
		// Check if the pool exists
		exists, err := store.ServicePoolRepo().Exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("service pool with id %s not found", id)
		}

		// Delete the pool
		return store.ServicePoolRepo().Delete(ctx, id)
	})
}
