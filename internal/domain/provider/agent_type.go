package provider

import (
	"errors"

	"fulcrumproject.org/core/internal/domain/common"
)

// AgentType represents a type of service manager agent
type AgentType struct {
	common.BaseEntity
	Name         string        `gorm:"not null;unique" json:"name"`
	ServiceTypes []ServiceType `gorm:"many2many:agent_type_service_types" json:"serviceTypes,omitempty"`
}

// NewAgentType creates a new AgentType with the given parameters
func NewAgentType(name string) (*AgentType, error) {
	if err := common.ValidateName(name); err != nil {
		return nil, err
	}

	return &AgentType{
		Name: name,
	}, nil
}

// Validate checks if the agent type is valid
func (at *AgentType) Validate() error {
	if err := common.ValidateName(at.Name); err != nil {
		return err
	}
	return nil
}

// AddServiceType adds a service type to the agent type
func (at *AgentType) AddServiceType(serviceType *ServiceType) error {
	if serviceType == nil {
		return errors.New("service type cannot be nil")
	}
	at.ServiceTypes = append(at.ServiceTypes, *serviceType)
	return nil
}

// RemoveServiceType removes a service type from the agent type
func (at *AgentType) RemoveServiceType(serviceTypeID common.UUID) error {
	for i, st := range at.ServiceTypes {
		if st.ID == serviceTypeID {
			at.ServiceTypes = append(at.ServiceTypes[:i], at.ServiceTypes[i+1:]...)
			return nil
		}
	}
	return errors.New("service type not found")
}

// HasServiceType checks if the agent type supports a specific service type
func (at *AgentType) HasServiceType(serviceTypeID common.UUID) bool {
	for _, st := range at.ServiceTypes {
		if st.ID == serviceTypeID {
			return true
		}
	}
	return false
}

// TableName returns the table name for the agent type
func (AgentType) TableName() string {
	return "agent_types"
}
