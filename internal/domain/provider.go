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
func (s ProviderState) Validate() error {
	switch s {
	case ProviderEnabled, ProviderDisabled:
		return ErrInvalidProviderState
	default:
		return nil
	}
}

// Provider represents a cloud service provider
type Provider struct {
	BaseEntity
	Name        Name          `gorm:"not null"`
	State       ProviderState `gorm:"not null"`
	CountryCode CountryCode   `gorm:"size:2"`
	Attributes  Attributes    `gorm:"type:jsonb"`
	Agents      []Agent       `gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for the provider
func (Provider) TableName() string {
	return "providers"
}

// Validate ensures all Provider fields are valid
func (p Provider) Validate() error {
	if err := p.Name.Validate(); err != nil {
		return err
	}

	if err := p.State.Validate(); err != nil {
		return err
	}

	if err := p.CountryCode.Validate(); err != nil {
		return err
	}

	if p.Attributes != nil {
		return p.Attributes.Validate()
	}
	return nil
}
