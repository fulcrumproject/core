package provider

import (
	"fulcrumproject.org/core/internal/domain/common"
)

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	common.BaseEntity
	Name                string          `gorm:"not null;unique" json:"name"`
	ResourceDefinitions common.GormJSON `gorm:"type:jsonb" json:"resourceDefinitions"`
	AgentTypes          []AgentType     `gorm:"many2many:agent_type_service_types" json:"agentTypes,omitempty"`
}

// NewServiceType creates a new ServiceType with the given parameters
func NewServiceType(name string, resourceDefinitions common.JSON) (*ServiceType, error) {
	if err := common.ValidateName(name); err != nil {
		return nil, err
	}
	if err := common.ValidateJSON(resourceDefinitions); err != nil {
		return nil, err
	}

	gormJSON, err := resourceDefinitions.ToGormJSON()
	if err != nil {
		return nil, err
	}

	return &ServiceType{
		Name:                name,
		ResourceDefinitions: gormJSON,
	}, nil
}

// Validate checks if the service type is valid
func (st *ServiceType) Validate() error {
	if err := common.ValidateName(st.Name); err != nil {
		return err
	}
	return nil
}

// GetResourceDefinitions returns the resource definitions as a JSON object
func (st *ServiceType) GetResourceDefinitions() (common.JSON, error) {
	return st.ResourceDefinitions.ToJSON()
}

// UpdateResourceDefinitions updates the resource definitions
func (st *ServiceType) UpdateResourceDefinitions(definitions common.JSON) error {
	if err := common.ValidateJSON(definitions); err != nil {
		return err
	}

	gormJSON, err := definitions.ToGormJSON()
	if err != nil {
		return err
	}

	st.ResourceDefinitions = gormJSON
	return nil
}

// TableName returns the table name for the service type
func (ServiceType) TableName() string {
	return "service_types"
}
