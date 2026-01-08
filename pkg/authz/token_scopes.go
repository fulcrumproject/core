package authz

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
)

// TokenCreationScope represents the authorization context for creating a new token
type TokenCreationScope struct {
	ctx 				 context.Context
	store 			 domain.Store
	tokenRole 	 auth.Role
	tokenScopeId *properties.UUID
}

// NewTokenCreationScope creates a new TokenCreationScope for authorization checks.
func NewTokenCreationScope(
	ctx context.Context,
	store domain.Store,
	tokenRole auth.Role,
	tokenScopeId *properties.UUID,
) *TokenCreationScope {
	return &TokenCreationScope{
		ctx: ctx,
		store: store,
		tokenRole: tokenRole,
		tokenScopeId: tokenScopeId,
	} 
}

// Matches checks if the given identity has permission to create the token.
// Authorization rules:
//   - Admins can create any token (admin, participant, or agent)
//   - Non-admins cannot create admin tokens
//   - Participants can create agent tokens only for agents they own
//   - Participants can create participant tokens only for themselves
func(s *TokenCreationScope) Matches(id *auth.Identity) bool {

	// Reject if no identity provided
	if id == nil {
		return false
	}

	// Admins have unrestricted access to create any token
	if id.Role == auth.RoleAdmin {
		return true
	}

	// Non-admins cannot create admin tokens
	if s.tokenRole == auth.RoleAdmin {
		return false
	}

	// For agent tokens: verify the caller owns the agent's provider
	if s.tokenRole == auth.RoleAgent && s.tokenScopeId != nil {
		agent, err := s.store.AgentRepo().Get(s.ctx, *s.tokenScopeId)
		
		// Reject if agent not found or database error
		if err != nil {
			return false
		}

		// Verify the caller's participant ID matches the agent's provider ID
		if id.Scope.ParticipantID != nil && agent.ProviderID != *id.Scope.ParticipantID {
			return false
		}
	}

	// For participant tokens: verify the caller is creating a token for themselves
	if s.tokenRole == auth.RoleParticipant && s.tokenScopeId != nil {
		if id.Scope.ParticipantID != nil && *s.tokenScopeId != *id.Scope.ParticipantID{
			return false
		}
	}

	// All checks passed	
	return true
}
