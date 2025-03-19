package database

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
)

// GormTokenIdentity implements the domain.Identity interface
type GormTokenIdentity struct {
	id         domain.UUID
	name       string
	role       domain.AuthRole
	providerID *domain.UUID
	brokerID   *domain.UUID
	agentID    *domain.UUID
}

// ID returns the identity's ID
func (i *GormTokenIdentity) ID() domain.UUID {
	return i.id
}

// Name returns the identity's name
func (i *GormTokenIdentity) Name() string {
	return i.name
}

// Role returns the identity's role
func (i *GormTokenIdentity) Role() domain.AuthRole {
	return i.role
}

// Scope returns the identity's authorization scope
func (i *GormTokenIdentity) Scope() *domain.AuthScope {
	return &domain.AuthScope{
		ProviderID: i.providerID,
		BrokerID:   i.brokerID,
		AgentID:    i.agentID,
	}
}

// IsRole checks if the identity has the specified role
func (i *GormTokenIdentity) IsRole(role domain.AuthRole) bool {
	return i.role == role
}

// GormTokenAuthenticator implements domain.Authenticator using GORM database
type GormTokenAuthenticator struct {
	store domain.Store
}

// NewTokenAuthenticator creates a new token authenticator
func NewTokenAuthenticator(store domain.Store) *GormTokenAuthenticator {
	return &GormTokenAuthenticator{
		store: store,
	}
}

// Authenticate extracts and validates the token from the HTTP request
// Returns nil if authentication fails
func (a *GormTokenAuthenticator) Authenticate(ctx context.Context, tokenValue string) domain.AuthIdentity {
	// Hash the token value
	hashedValue := domain.HashTokenValue(tokenValue)

	// Look up the token in the database
	token, err := a.store.TokenRepo().FindByHashedValue(ctx, hashedValue)
	if err != nil {
		return nil
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil
	}

	// Create a new identity
	return &GormTokenIdentity{
		id:         token.ID,
		name:       token.Name,
		role:       token.Role,
		providerID: token.ProviderID,
		brokerID:   token.BrokerID,
		agentID:    token.AgentID,
	}
}
