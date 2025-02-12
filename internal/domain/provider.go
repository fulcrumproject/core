package domain

import (
	"errors"
)

var (
	// ErrInvalidProviderState indicates that the provider state is not valid
	ErrInvalidProviderState = errors.New("invalid provider state")
)

// ProviderState represents the possible states of a Provider
type ProviderState string

const (
	// ProviderEnabled represents an enabled provider
	ProviderEnabled ProviderState = "Enabled"
	// ProviderDisabled represents a disabled provider
	ProviderDisabled ProviderState = "Disabled"
)

// IsValid checks if the provider state is valid
func (s ProviderState) IsValid() bool {
	switch s {
	case ProviderEnabled, ProviderDisabled:
		return true
	default:
		return false
	}
}

// Provider represents a cloud service provider
type Provider struct {
	BaseEntity
	Name        string         `gorm:"not null" json:"name"`
	State       ProviderState  `gorm:"not null" json:"state"`
	CountryCode string         `gorm:"size:2" json:"countryCode"`
	Attributes  GormAttributes `gorm:"type:jsonb" json:"attributes"`
	Agents      []Agent        `gorm:"foreignKey:ProviderID" json:"agents,omitempty"`
}

// NewProvider creates a new Provider with the given parameters
func NewProvider(name, countryCode string, attributes Attributes) (*Provider, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := ValidateCountryCode(countryCode); err != nil {
		return nil, err
	}
	if err := ValidateAttributes(attributes); err != nil {
		return nil, err
	}

	gormAttrs, err := attributes.ToGormAttributes()
	if err != nil {
		return nil, err
	}

	return &Provider{
		Name:        name,
		State:       ProviderEnabled, // Default state is enabled
		CountryCode: countryCode,
		Attributes:  gormAttrs,
	}, nil
}

// Validate checks if the provider is valid
func (p *Provider) Validate() error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if err := ValidateCountryCode(p.CountryCode); err != nil {
		return err
	}
	if !p.State.IsValid() {
		return ErrInvalidProviderState
	}
	return nil
}

// GetAttributes returns the attributes as an Attributes map
func (p *Provider) GetAttributes() (Attributes, error) {
	return p.Attributes.ToAttributes()
}

// Enable sets the provider state to enabled
func (p *Provider) Enable() error {
	if p.State == ProviderEnabled {
		return nil
	}
	p.State = ProviderEnabled
	return nil
}

// Disable sets the provider state to disabled
func (p *Provider) Disable() error {
	if p.State == ProviderDisabled {
		return nil
	}
	p.State = ProviderDisabled
	return nil
}

// AddAgent adds an agent to the provider
func (p *Provider) AddAgent(agent *Agent) error {
	if agent == nil {
		return errors.New("agent cannot be nil")
	}
	if agent.ProviderID != p.ID {
		agent.ProviderID = p.ID
	}
	p.Agents = append(p.Agents, *agent)
	return nil
}

// RemoveAgent removes an agent from the provider
func (p *Provider) RemoveAgent(agentID UUID) error {
	for i, agent := range p.Agents {
		if agent.ID == agentID {
			p.Agents = append(p.Agents[:i], p.Agents[i+1:]...)
			return nil
		}
	}
	return errors.New("agent not found")
}

// GetAgent returns an agent by ID
func (p *Provider) GetAgent(agentID UUID) (*Agent, error) {
	for _, agent := range p.Agents {
		if agent.ID == agentID {
			return &agent, nil
		}
	}
	return nil, errors.New("agent not found")
}

// TableName returns the table name for the provider
func (Provider) TableName() string {
	return "providers"
}
