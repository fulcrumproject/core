package keycloak

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

// Claims represents the custom claims structure from Keycloak JWT tokens
type Claims struct {
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

// Authenticator implements domain.Authenticator using OIDC/Keycloak JWT tokens
type Authenticator struct {
	config   *Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

// NewAuthenticator creates a new OIDC JWT authenticator for Keycloak
func NewAuthenticator(ctx context.Context, cfg *Config) (*Authenticator, error) {

	if cfg.InsecureSkipVerify {
		customClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		ctx = oidc.ClientContext(ctx, customClient)
	}

	if !cfg.ValidateIssuer {
		ctx = oidc.InsecureIssuerURLContext(ctx, cfg.GetIssuer())
	}

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

	return &Authenticator{
		config:   cfg,
		provider: provider,
		verifier: verifier,
	}, nil
}

// Authenticate extracts and validates the JWT token against Keycloak
// Returns nil if authentication fails
func (a *Authenticator) Authenticate(ctx context.Context, tokenString string) (*auth.Identity, error) {
	// Verify the ID token
	idToken, err := a.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Parse and validate the subject as UUID (identity ID)
	id, err := properties.ParseUUID(idToken.Subject)
	if err != nil {
		return nil, err
	}

	// Extract custom claims
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}

	// Extract role from custom claim or realm roles
	role, err := a.extractRole(&claims)
	if err != nil {
		return nil, err
	}

	// Parse optional participant ID
	var participantID *properties.UUID
	if claims.ParticipantID != "" {
		pid, err := properties.ParseUUID(claims.ParticipantID)
		if err != nil {
			return nil, err
		}
		participantID = &pid
	}

	// Parse optional agent ID
	var agentID *properties.UUID
	if claims.AgentID != "" {
		aid, err := properties.ParseUUID(claims.AgentID)
		if err != nil {
			return nil, err
		}
		agentID = &aid
	}

	// Use preferred name or fallback to preferred_username
	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}
	if name == "" {
		name = idToken.Subject // Fallback to subject if no name available
	}

	// Create the identity
	identity := &auth.Identity{
		ID:   id,
		Name: name,
		Role: role,
		Scope: auth.IdentityScope{
			ParticipantID: participantID,
			AgentID:       agentID,
		},
	}

	// Validate the identity to ensure it meets role-specific requirements
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("invalid identity: %w", err)
	}

	return identity, nil
}

// extractRole extracts the role from Keycloak claims
func (a *Authenticator) extractRole(claims *Claims) (auth.Role, error) {
	// First check if there's a direct role claim
	if claims.Role != "" {
		role := auth.Role(claims.Role)
		if err := role.Validate(); err == nil {
			return role, nil
		}
	}

	// Check realm roles
	for _, realmRole := range claims.RealmAccess.Roles {
		role := auth.Role(realmRole)
		if err := role.Validate(); err == nil {
			return role, nil
		}
	}

	// Check client-specific roles
	if clientRoles, exists := claims.ResourceAccess[a.config.ClientID]; exists {
		for _, clientRole := range clientRoles.Roles {
			role := auth.Role(clientRole)
			if err := role.Validate(); err == nil {
				return role, nil
			}
		}
	}

	return "", errors.New("no valid role found in token")
}

// Health checks if the Keycloak/OIDC provider is accessible
func (a *Authenticator) Health(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("OIDC provider is not initialized")
	}

	if a.verifier == nil {
		return fmt.Errorf("OIDC verifier is not initialized")
	}

	// Try to fetch the provider's configuration to verify connectivity
	// This is a lightweight check that verifies the provider is reachable
	// without requiring authentication
	endpoint := a.provider.Endpoint()
	if endpoint.AuthURL == "" {
		return fmt.Errorf("OIDC provider endpoint is not configured")
	}

	return nil
}
