// ServicePoolSet entity - collection of resource pools per provider
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeServicePoolSetCreated EventType = "service_pool_set.created"
	EventTypeServicePoolSetUpdated EventType = "service_pool_set.updated"
	EventTypeServicePoolSetDeleted EventType = "service_pool_set.deleted"
)

// ServicePoolSet represents a collection of resource pools for a provider
type ServicePoolSet struct {
	BaseEntity

	Name       string          `json:"name" gorm:"not null"`
	ProviderID properties.UUID `json:"providerId" gorm:"not null;index"`
	Provider   *Participant    `json:"-" gorm:"foreignKey:ProviderID"`
}

// CreateServicePoolSetParams defines parameters for creating a ServicePoolSet
type CreateServicePoolSetParams struct {
	Name       string
	ProviderID properties.UUID
}

// UpdateServicePoolSetParams defines parameters for updating a ServicePoolSet
type UpdateServicePoolSetParams struct {
	Name *string
}

// NewServicePoolSet creates a new service pool set without validation
func NewServicePoolSet(params CreateServicePoolSetParams) *ServicePoolSet {
	return &ServicePoolSet{
		Name:       params.Name,
		ProviderID: params.ProviderID,
	}
}

// TableName returns the table name for the service pool set
func (ServicePoolSet) TableName() string {
	return "service_pool_sets"
}

// Validate ensures all ServicePoolSet fields are valid
func (sps *ServicePoolSet) Validate() error {
	if sps.Name == "" {
		return fmt.Errorf("pool set name cannot be empty")
	}
	if sps.ProviderID == (properties.UUID{}) {
		return fmt.Errorf("provider ID cannot be empty")
	}
	return nil
}

// Update modifies the ServicePoolSet with provided parameters
func (sps *ServicePoolSet) Update(params UpdateServicePoolSetParams) {
	if params.Name != nil {
		sps.Name = *params.Name
	}
}

// ServicePoolSetRepository manages ServicePoolSet entities
type ServicePoolSetRepository interface {
	ServicePoolSetQuerier
	Create(ctx context.Context, poolSet *ServicePoolSet) error
	Update(ctx context.Context, poolSet *ServicePoolSet) error
	Delete(ctx context.Context, id properties.UUID) error
}

// ServicePoolSetQuerier provides read-only access to ServicePoolSet entities
type ServicePoolSetQuerier interface {
	BaseEntityQuerier[ServicePoolSet]

	FindByProvider(ctx context.Context, providerID properties.UUID) ([]*ServicePoolSet, error)
	FindByProviderAndName(ctx context.Context, providerID properties.UUID, name string) (*ServicePoolSet, error)
}

// ServicePoolSetCommander handles complex ServicePoolSet operations
type ServicePoolSetCommander interface {
	Create(ctx context.Context, params CreateServicePoolSetParams) (*ServicePoolSet, error)
	Update(ctx context.Context, id properties.UUID, params UpdateServicePoolSetParams) (*ServicePoolSet, error)
	Delete(ctx context.Context, id properties.UUID) error
}

// servicePoolSetCommander is the concrete implementation of ServicePoolSetCommander
type servicePoolSetCommander struct {
	store Store
}

// NewServicePoolSetCommander creates a new ServicePoolSetCommander
func NewServicePoolSetCommander(store Store) ServicePoolSetCommander {
	return &servicePoolSetCommander{store: store}
}

// Create creates a new service pool set
func (c *servicePoolSetCommander) Create(
	ctx context.Context,
	params CreateServicePoolSetParams,
) (*ServicePoolSet, error) {
	var poolSet *ServicePoolSet
	err := c.store.Atomic(ctx, func(store Store) error {
		// Validate that the provider exists
		exists, err := store.ParticipantRepo().Exists(ctx, params.ProviderID)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("provider with id %s not found", params.ProviderID)
		}

		// Create the pool set
		poolSet = NewServicePoolSet(params)
		if err := poolSet.Validate(); err != nil {
			return err
		}

		// Save to database
		if err := store.ServicePoolSetRepo().Create(ctx, poolSet); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return poolSet, nil
}

// Update updates an existing service pool set
func (c *servicePoolSetCommander) Update(
	ctx context.Context,
	id properties.UUID,
	params UpdateServicePoolSetParams,
) (*ServicePoolSet, error) {
	var poolSet *ServicePoolSet
	err := c.store.Atomic(ctx, func(store Store) error {
		// Get the existing pool set
		var err error
		poolSet, err = store.ServicePoolSetRepo().Get(ctx, id)
		if err != nil {
			return err
		}

		// Update fields
		poolSet.Update(params)

		// Validate
		if err := poolSet.Validate(); err != nil {
			return err
		}

		// Save changes
		if err := store.ServicePoolSetRepo().Update(ctx, poolSet); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return poolSet, nil
}

// Delete deletes a service pool set
func (c *servicePoolSetCommander) Delete(
	ctx context.Context,
	id properties.UUID,
) error {
	return c.store.Atomic(ctx, func(store Store) error {
		// Check if the pool set exists
		exists, err := store.ServicePoolSetRepo().Exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return NewNotFoundErrorf("service pool set with id %s not found", id)
		}

		// Delete the pool set
		return store.ServicePoolSetRepo().Delete(ctx, id)
	})
}
