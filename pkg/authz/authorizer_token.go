// Token-specific authorization logic
package authz

import (
	"fmt"

	"github.com/fulcrumproject/core/pkg/auth"
)

// TokenCreationScope wraps an ObjectScope and carries the target role for token creation
type TokenCreationScope struct {
	ObjectScope // Embed to delegate Matches() automatically
	targetRole  auth.Role
}

// NewTokenCreationScope creates a scope for token creation with target role information
func NewTokenCreationScope(targetRole auth.Role, scope ObjectScope) *TokenCreationScope {
	return &TokenCreationScope{
		ObjectScope: scope,
		targetRole:  targetRole,
	}
}

// TargetRole returns the role of the token being created
func (s *TokenCreationScope) TargetRole() auth.Role {
	return s.targetRole
}

// TokenAuthorizer wraps the default authorizer and adds token-specific role validation
type TokenAuthorizer struct {
	wrapped Authorizer
}

// NewTokenAuthorizer creates a new token creation authorizer
func NewTokenAuthorizer(wrapped Authorizer) *TokenAuthorizer {
	return &TokenAuthorizer{
		wrapped: wrapped,
	}
}

// Authorize performs authorization with token-specific role validation
// For token creation, it first validates that the identity can create tokens with the target role,
// then delegates to the wrapped authorizer for scope validation
func (a *TokenAuthorizer) Authorize(
	identity *auth.Identity,
	action Action,
	objectType ObjectType,
	objectScope ObjectScope,
) error {
	// Only apply custom logic for token creation
	if objectType == ObjectTypeToken && action == ActionCreate {
		if identity == nil {
			return fmt.Errorf("access denied: no identity provided")
		}

		// Extract target role from TokenCreationScope if available
		var targetRole auth.Role
		if tcs, ok := objectScope.(*TokenCreationScope); ok {
			targetRole = tcs.TargetRole()
		} else {
			return fmt.Errorf("access denied: no target role provided")
		}

		// Validate role permissions
		if !canCreateTokenWithRole(identity, targetRole) {
			return fmt.Errorf(
				"access denied: role %s cannot create tokens with role %s",
				identity.Role,
				targetRole,
			)
		}
	}

	// Delegate to wrapped authorizer for standard authorization
	return a.wrapped.Authorize(identity, action, objectType, objectScope)
}

// canCreateTokenWithRole checks if the identity can create a token with the given target role
func canCreateTokenWithRole(identity *auth.Identity, targetRole auth.Role) bool {
	if identity == nil {
		return false
	}

	// Admins can create tokens with any role (admin, participant, or agent)
	if identity.Role == auth.RoleAdmin {
		return true
	}

	// Participants can create tokens with role participant or agent (but not admin)
	if identity.Role == auth.RoleParticipant {
		return targetRole == auth.RoleParticipant || targetRole == auth.RoleAgent
	}

	// Agents cannot create tokens
	return false
}
