package domain

import (
	"context"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	BaseEntity
	Name           string               `json:"name" gorm:"not null;unique"`
	PropertySchema *schema.CustomSchema `json:"propertySchema,omitempty" gorm:"type:jsonb"`
}

// TableName returns the table name for the service type
func (ServiceType) TableName() string {
	return "service_types"
}

// ServiceTypeRepository defines the interface for the ServiceType repository
type ServiceTypeRepository interface {
	ServiceTypeQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *ServiceType) error
}

// ServiceTypeQuerier defines the interface for the ServiceType read-only queries
type ServiceTypeQuerier interface {
	// Get retrieves an entity by ID
	Get(ctx context.Context, id properties.UUID) (*ServiceType, error)

	// Exists checks if an entity exists by ID
	Exists(ctx context.Context, id properties.UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *auth.IdentityScope, req *PageRequest) (*PageResponse[ServiceType], error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}
