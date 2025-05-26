package domain

import "context"

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	BaseEntity
	Name string `json:"name" gorm:"not null;unique"`
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

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[ServiceType], error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}
