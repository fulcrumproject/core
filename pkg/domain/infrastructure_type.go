package domain

import (
	"fmt"

	"github.com/fulcrumproject/core/pkg/schema"
)

const (
	EventTypeInfrastructureTypeCreated EventType = "agent_type.created"
	EventTypeInfrastructureTypeUpdated EventType = "agent_type.updated"
	EventTypeInfrastructureTypeDeleted EventType = "agent_type.deleted"
)

type InfrastructureType struct {
	BaseEntity
	Name string `json:"name" gorm:"not null;unique"`
	TemplateValidation
}

func NewInfrastructureType(params CreateInfrastructureTypeParams) *InfrastructureType {
	return &InfrastructureType{
		Name: params.Name,
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: params.ConfigurationSchema,
			ConfigTemplate:      params.ConfigTemplate,
			CmdTemplate:         params.CmdTemplate,
			ConfigContentType:   params.ConfigContentType,
		},
	}
}

func (InfrastructureType) TableName() string {
	return "infrastructure_types"
}

func (it InfrastructureType) Validate() error {
	if it.Name == "" {
		return fmt.Errorf("infrastructure type name cannot be empty")
	}
	return nil
}

type CreateInfrastructureTypeParams struct {
	Name                string        `json:"name"`
	ConfigurationSchema schema.Schema `json:"configurationSchema"`
	ConfigTemplate      string        `json:"configTemplate"`
	CmdTemplate         string        `json:"cmdTemplate"`
	ConfigContentType   string        `json:"configContentType"`
}
