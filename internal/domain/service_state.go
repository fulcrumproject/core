package domain

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
