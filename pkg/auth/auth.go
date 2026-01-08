package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

// Role represents a role in the authentication system
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
	Health(ctx context.Context) error
}
