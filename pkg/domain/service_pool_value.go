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
	Get(ctx context.Context, id properties.UUID) (*ServicePoolValue, error)
	List(ctx context.Context) ([]*ServicePoolValue, error)
	ListByPool(ctx context.Context, poolID properties.UUID) ([]*ServicePoolValue, error)
	ListByService(ctx context.Context, serviceID properties.UUID) ([]*ServicePoolValue, error)
	FindByPool(ctx context.Context, poolID properties.UUID) ([]*ServicePoolValue, error)
	FindAvailable(ctx context.Context, poolID properties.UUID) ([]*ServicePoolValue, error)
	FindByService(ctx context.Context, serviceID properties.UUID) ([]*ServicePoolValue, error)
	Exists(ctx context.Context, id properties.UUID) (bool, error)
}

// ServicePoolValueCommander handles complex ServicePoolValue operations
type ServicePoolValueCommander interface {
	CreateServicePoolValue(ctx context.Context, params CreateServicePoolValueParams) (*ServicePoolValue, error)
	DeleteServicePoolValue(ctx context.Context, id properties.UUID) error
}
