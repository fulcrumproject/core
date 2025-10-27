// Context type for service property schema validation and processing
package domain

import (
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

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
// It contains the actor performing the operation and essential service context information.
type ServicePropertyContext struct {
	// Actor identifies who is performing the operation (user, agent, system)
	Actor ActorType

	// Store provides access to repositories within the current transaction.
	// Validators and generators use this to make DB calls within the same transaction.
	Store Store

	// ProviderID is the ID of the provider participant.
	// Required for validators like serviceOption that need to look up provider-specific options.
	ProviderID properties.UUID

	// ServicePoolSetID is the pool set for resource allocation (optional).
	// Required by pool generators to allocate values from pools.
	ServicePoolSetID *properties.UUID

	// ServiceID is the ID of the service being updated (nil during create).
	// Used by validators that need to check current service state.
	ServiceID *properties.UUID

	// ServiceStatus is the current status of the service (empty during create).
	// Used by validators like mutable that check if properties can be updated in current status.
	ServiceStatus string
}
