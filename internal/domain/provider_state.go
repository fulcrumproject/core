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
		return nil
	default:
		return ErrInvalidProviderState
	}
}

func ParseProviderState(value string) (ProviderState, error) {
	state := ProviderState(value)
	return state, state.Validate()
}
