package oauth

import (
	"context"
	"errors"
	"fmt"

	"fulcrumproject.org/core/internal/config"
	"fulcrumproject.org/core/internal/domain"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// KeycloakClaims represents the custom claims structure from Keycloak JWT tokens
type KeycloakClaims struct {
	Role              string `json:"role,omitempty"`
	ParticipantID     string `json:"participant_id,omitempty"`
	AgentID           string `json:"agent_id,omitempty"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access,omitempty"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access,omitempty"`
}

// JWTIdentity implements the domain.Identity interface for OAuth JWT tokens
type JWTIdentity struct {
	id            domain.UUID
	name          string
	role          domain.AuthRole
	participantID *domain.UUID
	agentID       *domain.UUID
}

// ID returns the identity's ID (from JWT subject)
func (i *JWTIdentity) ID() domain.UUID {
	return i.id
}

// Name returns the identity's name
func (i *JWTIdentity) Name() string {
	return i.name
}

// Role returns the identity's role
func (i *JWTIdentity) Role() domain.AuthRole {
	return i.role
}

// Scope returns the identity's authorization scope
func (i *JWTIdentity) Scope() *domain.AuthIdentityScope {
	return &domain.AuthIdentityScope{
		ParticipantID: i.participantID,
		AgentID:       i.agentID,
	}
}

// IsRole checks if the identity has the specified role
func (i *JWTIdentity) IsRole(role domain.AuthRole) bool {
	return i.role == role
}

// OIDCAuthenticator implements domain.Authenticator using OIDC/Keycloak JWT tokens
type OIDCAuthenticator struct {
	config   config.OAuthConfig
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

// NewOIDCAuthenticator creates a new OIDC JWT authenticator for Keycloak
func NewOIDCAuthenticator(ctx context.Context, cfg config.OAuthConfig) (*OIDCAuthenticator, error) {
	// Create OIDC provider
	provider, err := oidc.NewProvider(ctx, cfg.GetIssuer())
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Configure the ID token verifier
	verifierConfig := &oidc.Config{
		ClientID: cfg.ClientID,
		// Skip audience check since Keycloak may use different audiences
		SkipClientIDCheck: true,
	}

	// Skip issuer validation if configured
	if !cfg.ValidateIssuer {
		verifierConfig.SkipIssuerCheck = true
	}

	verifier := provider.Verifier(verifierConfig)

	return &OIDCAuthenticator{
		config:   cfg,
		provider: provider,
		verifier: verifier,
	}, nil
}

// Authenticate extracts and validates the JWT token against Keycloak
// Returns nil if authentication fails
func (a *OIDCAuthenticator) Authenticate(ctx context.Context, tokenString string) domain.AuthIdentity {
	// Verify the ID token
	idToken, err := a.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil
	}

	// Parse and validate the subject as UUID (identity ID)
	id, err := domain.ParseUUID(idToken.Subject)
	if err != nil {
		return nil
	}

	// Extract custom claims
	var claims KeycloakClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil
	}

	// Extract role from custom claim or realm roles
	role, err := a.extractRole(&claims)
	if err != nil {
		return nil
	}

	// Parse optional participant ID
	var participantID *domain.UUID
	if claims.ParticipantID != "" {
		pid, err := domain.ParseUUID(claims.ParticipantID)
		if err != nil {
			return nil
		}
		participantID = &pid
	}

	// Parse optional agent ID
	var agentID *domain.UUID
	if claims.AgentID != "" {
		aid, err := domain.ParseUUID(claims.AgentID)
		if err != nil {
			return nil
		}
		agentID = &aid
	}

	// Validate role-specific requirements
	if err := a.validateRoleRequirements(role, participantID, agentID); err != nil {
		return nil
	}

	// Use preferred name or fallback to preferred_username
	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}
	if name == "" {
		name = idToken.Subject // Fallback to subject if no name available
	}

	// Create and return the identity
	return &JWTIdentity{
		id:            id,
		name:          name,
		role:          role,
		participantID: participantID,
		agentID:       agentID,
	}
}

// extractRole extracts the role from Keycloak claims
func (a *OIDCAuthenticator) extractRole(claims *KeycloakClaims) (domain.AuthRole, error) {
	// First check if there's a direct role claim
	if claims.Role != "" {
		role := domain.AuthRole(claims.Role)
		if err := role.Validate(); err == nil {
			return role, nil
		}
	}

	// Check realm roles
	for _, realmRole := range claims.RealmAccess.Roles {
		role := domain.AuthRole(realmRole)
		if err := role.Validate(); err == nil {
			return role, nil
		}
	}

	// Check client-specific roles
	if clientRoles, exists := claims.ResourceAccess[a.config.ClientID]; exists {
		for _, clientRole := range clientRoles.Roles {
			role := domain.AuthRole(clientRole)
			if err := role.Validate(); err == nil {
				return role, nil
			}
		}
	}

	return "", errors.New("no valid role found in token")
}

// validateRoleRequirements ensures that role-specific ID requirements are met
func (a *OIDCAuthenticator) validateRoleRequirements(role domain.AuthRole, participantID, agentID *domain.UUID) error {
	switch role {
	case domain.RoleAdmin:
		// Admin should not have participant or agent IDs
		if participantID != nil || agentID != nil {
			return errors.New("admin role should not have participant_id or agent_id")
		}
	case domain.RoleParticipant:
		// Participant must have participant ID but not agent ID
		if participantID == nil {
			return errors.New("participant role requires participant_id")
		}
		if agentID != nil {
			return errors.New("participant role should not have agent_id")
		}
	case domain.RoleAgent:
		// Agent must have both participant ID and agent ID
		if participantID == nil {
			return errors.New("agent role requires participant_id")
		}
		if agentID == nil {
			return errors.New("agent role requires agent_id")
		}
	default:
		return fmt.Errorf("unknown role: %s", role)
	}
	return nil
}

// GetOAuth2Config returns the OAuth2 configuration for this OIDC provider
// This can be used for OAuth2 flows if needed
func (a *OIDCAuthenticator) GetOAuth2Config(redirectURL string, scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.config.ClientID,
		ClientSecret: a.config.ClientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     a.provider.Endpoint(),
		Scopes:       scopes,
	}
}
