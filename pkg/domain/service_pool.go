// ServicePool entity - type-specific pool with generator configuration
package domain

import (
	"context"
	"fmt"

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
	Get(ctx context.Context, id properties.UUID) (*ServicePool, error)
	ListByPoolSet(ctx context.Context, poolSetID properties.UUID) ([]*ServicePool, error)
	FindByPoolSetAndType(ctx context.Context, poolSetID properties.UUID, poolType string) (*ServicePool, error)
	Exists(ctx context.Context, id properties.UUID) (bool, error)
}

// ServicePoolCommander handles complex ServicePool operations
type ServicePoolCommander interface {
	CreateServicePool(ctx context.Context, params CreateServicePoolParams) (*ServicePool, error)
	UpdateServicePool(ctx context.Context, id properties.UUID, params UpdateServicePoolParams) (*ServicePool, error)
	DeleteServicePool(ctx context.Context, id properties.UUID) error
}
