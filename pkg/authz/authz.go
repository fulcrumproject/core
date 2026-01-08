// Authorization types and interfaces
package authz

import (
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

// Action represents an action that can be performed on an object
type Action string

// ObjectType represents a target object type in the authorization system
type ObjectType string

// ObjectScope defines the target object scope in the authorization system
type ObjectScope interface {
	Matches(identity *auth.Identity) bool
}

// AllwaysMatchObjectScope is a special ObjectScope that always matches any identity
type AllwaysMatchObjectScope struct{}

func (a AllwaysMatchObjectScope) Matches(identity *auth.Identity) bool {
	return true // Always matches, used for global actions
}

// DefaultObjectScope is the default implementation of ObjectScope
type DefaultObjectScope struct {
	ParticipantID *properties.UUID
	ProviderID    *properties.UUID
	ConsumerID    *properties.UUID
	AgentID       *properties.UUID
}

// Matches checks if the given identity matches the object scope
func (target *DefaultObjectScope) Matches(id *auth.Identity) bool {
	if id == nil {
		return false
	}

	// If all fields are nil in the caller scope, it has unrestricted access (admin)
	if id.Scope.ParticipantID == nil && id.Scope.AgentID == nil {
		return true
	}

	// If all fields are nil in the target scope, global access is allowed
	if target.ParticipantID == nil && target.ProviderID == nil && target.ConsumerID == nil && target.AgentID == nil {
		return true
	}

	// Participant check: If identity requires a participant, caller must have same participant or provider or consumer
	if id.Scope.ParticipantID != nil {
		if target.ParticipantID != nil && *target.ParticipantID == *id.Scope.ParticipantID {
			return true
		}
		if target.ProviderID != nil && *target.ProviderID == *id.Scope.ParticipantID {
			return true
		}
		if target.ConsumerID != nil && *target.ConsumerID == *id.Scope.ParticipantID {
			return true
		}
	}

	// Agent check: If source requires an agent, caller must have same agent
	if id.Scope.AgentID != nil && target.AgentID != nil && *target.AgentID == *id.Scope.AgentID {
		return true
	}

	return false
}

// Authorizer interface for checking authorization
type Authorizer interface {
	Authorize(identity *auth.Identity, action Action, object ObjectType, objectScope ObjectScope) error
}
