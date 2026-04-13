package domain

import (
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

type AgentPool struct {
	BaseEntity
	Name         string `json:"name" gorm:"not null"`
	Type         string `json:"type" gorm:"not null"`
	PropertyType string `json:"propertyType" gorm:"not null"`
	// TODO: Check wether we need to define only one type since now used in only the list
	GeneratorType   PoolGeneratorType `json:"generatorType" gorm:"not null"`
	GeneratorConfig *properties.JSON  `json:"generatorConfig,omitempty" gorm:"type:jsonb"`
	ProviderID      properties.UUID   `json:"providerId" gorm:"not null;index"`
	Provider        *Participant      `json:"-" gorm:"foreignKey;ProviderID"`
}

func (AgentPool) TableName() string {
	return "agent_pools"
}

func (ap *AgentPool) Validate() error {
	if ap.Name == "" {
		return fmt.Errorf("agent pool name cannot be empty")
	}

	if ap.Type == "" {
		return fmt.Errorf("agent pool type cannot be empty")
	}
	return nil
}
