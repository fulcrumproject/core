package domain

import (
	"errors"
)

var (
	// ErrInvalidAgentState indicates that the agent state is not valid
	ErrInvalidAgentState = errors.New("invalid agent state")
)

// AgentState represents the possible states of an Agent
type AgentState string

const (
	// AgentNew represents a newly created agent
	AgentNew AgentState = "New"
	// AgentConnected represents a connected agent
	AgentConnected AgentState = "Connected"
	// AgentDisconnected represents a disconnected agent
	AgentDisconnected AgentState = "Disconnected"
	// AgentError represents an agent in error state
	AgentError AgentState = "Error"
	// AgentDisabled represents a disabled agent
	AgentDisabled AgentState = "Disabled"
)

// IsValid checks if the agent state is valid
func (s AgentState) IsValid() bool {
	switch s {
	case AgentNew, AgentConnected, AgentDisconnected, AgentError, AgentDisabled:
		return true
	default:
		return false
	}
}

// Agent represents a service manager agent
type Agent struct {
	BaseEntity
	Name        string     `gorm:"not null"`
	State       AgentState `gorm:"not null"`
	TokenHash   string     `gorm:"not null"`
	CountryCode string     `gorm:"size:2"`
	Attributes  Attributes `gorm:"type:jsonb"`
	Properties  JSON       `gorm:"type:jsonb"`
	ProviderID  UUID       `gorm:"not null"`
	AgentTypeID UUID       `gorm:"not null"`
	Provider    *Provider  `gorm:"foreignKey:ProviderID"`
	AgentType   *AgentType `gorm:"foreignKey:AgentTypeID"`
}

// TableName returns the table name for the agent
func (Agent) TableName() string {
	return "agents"
}
