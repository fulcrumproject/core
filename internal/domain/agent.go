package domain

import (
	"time"
)

// Agent represents a service manager agent
type Agent struct {
	BaseEntity
	Name             string     `gorm:"not null"`
	State            AgentState `gorm:"not null"`
	TokenHash        string     `gorm:"not null"`
	CountryCode      string     `gorm:"size:2"`
	Attributes       Attributes `gorm:"type:jsonb"`
	Properties       JSON       `gorm:"type:jsonb"`
	ProviderID       UUID       `gorm:"not null"`
	AgentTypeID      UUID       `gorm:"not null"`
	Provider         *Provider  `gorm:"foreignKey:ProviderID"`
	AgentType        *AgentType `gorm:"foreignKey:AgentTypeID"`
	LastStatusUpdate time.Time  `gorm:"index"`
}

// TableName returns the table name for the agent
func (Agent) TableName() string {
	return "agents"
}
