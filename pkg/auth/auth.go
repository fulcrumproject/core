package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

// Role represents a role in the authorization system
type Role string

const (
	RoleAdmin       Role = "admin"
	RoleParticipant Role = "participant"
	RoleAgent       Role = "agent"
)

// Validate ensures the Role is one of the predefined values
func (r Role) Validate() error {
	switch r {
	case RoleAdmin, RoleParticipant, RoleAgent:
		return nil
	default:
		return fmt.Errorf("invalid auth role: %s", r)
	}
}

// Action represents an action that can be performed on an object
type Action string

// ObjectType represents a target object type in the authorization system
type ObjectType string

// ObjectScope defines the target object scope in the authorization system
type ObjectScope interface {
	Matches(identity *Identity) bool
}

// AllwaysMatchObjectScope is a special ObjectScope that always matches any identity
type AllwaysMatchObjectScope struct{}

func (a AllwaysMatchObjectScope) Matches(identity *Identity) bool {
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
func (target *DefaultObjectScope) Matches(id *Identity) bool {
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

// Identity implements the Identifier interface
type Identity struct {
	ID    properties.UUID
	Name  string
	Role  Role
	Scope IdentityScope
}

func (m *Identity) HasRole(role Role) bool {
	return m.Role == role
}

// validateRoleRequirements ensures that role-specific ID requirements are met
func (m *Identity) Validate() error {
	switch m.Role {
	case RoleParticipant:
		if m.Scope.ParticipantID == nil {
			return errors.New("participant role requires participant id")
		}
	case RoleAgent:
		if m.Scope.ParticipantID == nil {
			return errors.New("agent role requires participant id")
		}
		if m.Scope.AgentID == nil {
			return errors.New("agent role requires agent id")
		}
	}

	return nil
}

type IdentityScope struct {
	ParticipantID *properties.UUID
	AgentID       *properties.UUID
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Identity, error)
}

type Authorizer interface {
	Authorize(identity *Identity, action Action, oject ObjectType, objectScope ObjectScope) error
}
