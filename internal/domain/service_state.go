package domain

import (
	"fmt"
)

// ServiceState represents the possible states of a service
type ServiceState string

const (
	ServiceNew      ServiceState = "New"
	ServiceCreating ServiceState = "Creating"
	ServiceCreated  ServiceState = "Created"
	ServiceUpdating ServiceState = "Updating"
	ServiceUpdated  ServiceState = "Updated"
	ServiceDeleting ServiceState = "Deleting"
	ServiceDeleted  ServiceState = "Deleted"
	ServiceError    ServiceState = "Error"
)

// ParseServiceState parses a string into a ServiceState
func ParseServiceState(s string) (ServiceState, error) {
	state := ServiceState(s)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Validate checks if the service state is valid
func (s ServiceState) Validate() error {
	switch s {
	case ServiceNew, ServiceCreating, ServiceCreated,
		ServiceUpdating, ServiceUpdated,
		ServiceDeleting, ServiceDeleted,
		ServiceError:
		return nil
	}
	return fmt.Errorf("invalid service state: %s", s)
}
