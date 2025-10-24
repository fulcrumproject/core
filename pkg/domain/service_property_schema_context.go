// Context type for service property schema validation and processing
package domain

import "github.com/fulcrumproject/core/pkg/auth"

// ActorType identifies the type of actor performing an operation
type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorAgent  ActorType = "agent"
	ActorSystem ActorType = "system"
)

// ActorTypeFromAuthRole converts an auth.Role to domain.ActorType
func ActorTypeFromAuthRole(role auth.Role) ActorType {
	switch role {
	case auth.RoleAgent:
		return ActorAgent
	case auth.RoleAdmin, auth.RoleParticipant:
		return ActorUser
	default:
		return ActorUser // Default to user for unknown roles
	}
}

// ServicePropertyContext provides runtime context for service property validation and generation.
// It contains the actor performing the operation and the service entity being operated on.
type ServicePropertyContext struct {
	// Actor identifies who is performing the operation (user, agent, system)
	Actor ActorType

	// Service is the service entity being operated on.
	// Nil during create operations, populated during update operations.
	// Validators can access service state, group ID, provider ID, etc.
	Service *Service
}
