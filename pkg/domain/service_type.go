package domain

import (
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
	BaseEntityRepository[ServiceType]
}

// ServiceTypeQuerier defines the interface for the ServiceType read-only queries
type ServiceTypeQuerier interface {
	BaseEntityQuerier[ServiceType]
}
