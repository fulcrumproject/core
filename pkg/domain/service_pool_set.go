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
	CreateServicePoolSet(ctx context.Context, params CreateServicePoolSetParams) (*ServicePoolSet, error)
	UpdateServicePoolSet(ctx context.Context, id properties.UUID, params UpdateServicePoolSetParams) (*ServicePoolSet, error)
	DeleteServicePoolSet(ctx context.Context, id properties.UUID) error
}
