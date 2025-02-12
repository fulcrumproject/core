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
	Name        string         `gorm:"not null" json:"name"`
	State       AgentState     `gorm:"not null" json:"state"`
	TokenHash   string         `gorm:"not null" json:"tokenHash"`
	CountryCode string         `gorm:"size:2" json:"countryCode"`
	Attributes  GormAttributes `gorm:"type:jsonb" json:"attributes"`
	Properties  GormJSON       `gorm:"type:jsonb" json:"properties"`
	ProviderID  UUID           `gorm:"not null" json:"providerId"`
	AgentTypeID UUID           `gorm:"not null" json:"agentTypeId"`
	Provider    *Provider      `gorm:"foreignKey:ProviderID" json:"-"`
	AgentType   *AgentType     `gorm:"foreignKey:AgentTypeID" json:"-"`
}

// NewAgent creates a new Agent with the given parameters
func NewAgent(name, countryCode string, attributes Attributes, properties JSON, providerID, agentTypeID UUID) (*Agent, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := ValidateCountryCode(countryCode); err != nil {
		return nil, err
	}
	if err := ValidateAttributes(attributes); err != nil {
		return nil, err
	}
	if err := ValidateJSON(properties); err != nil {
		return nil, err
	}
	if err := ValidateUUID(providerID); err != nil {
		return nil, err
	}
	if err := ValidateUUID(agentTypeID); err != nil {
		return nil, err
	}

	gormAttrs, err := attributes.ToGormAttributes()
	if err != nil {
		return nil, err
	}

	gormProps, err := properties.ToGormJSON()
	if err != nil {
		return nil, err
	}

	return &Agent{
		Name:        name,
		State:       AgentNew,
		CountryCode: countryCode,
		Attributes:  gormAttrs,
		Properties:  gormProps,
		ProviderID:  providerID,
		AgentTypeID: agentTypeID,
	}, nil
}

// Validate checks if the agent is valid
func (a *Agent) Validate() error {
	if err := ValidateName(a.Name); err != nil {
		return err
	}
	if err := ValidateCountryCode(a.CountryCode); err != nil {
		return err
	}
	if !a.State.IsValid() {
		return ErrInvalidAgentState
	}
	if err := ValidateUUID(a.ProviderID); err != nil {
		return err
	}
	if err := ValidateUUID(a.AgentTypeID); err != nil {
		return err
	}
	return nil
}

// GetAttributes returns the attributes as an Attributes map
func (a *Agent) GetAttributes() (Attributes, error) {
	return a.Attributes.ToAttributes()
}

// GetProperties returns the properties as a JSON map
func (a *Agent) GetProperties() (JSON, error) {
	return a.Properties.ToJSON()
}

// UpdateState updates the agent state if the transition is valid
func (a *Agent) UpdateState(newState AgentState) error {
	if !newState.IsValid() {
		return ErrInvalidAgentState
	}

	// Add state transition validation logic here if needed
	a.State = newState
	return nil
}

// TableName returns the table name for the agent
func (Agent) TableName() string {
	return "agents"
}
