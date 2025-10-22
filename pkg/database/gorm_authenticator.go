package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
)

var (
	ErrTokenExpired = errors.New("token is expired")
	ErrTokenInvalid = errors.New("invalid token")
)

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
func (a *GormTokenAuthenticator) Authenticate(ctx context.Context, tokenValue string) (*auth.Identity, error) {
	// Hash the token value
	hashedValue := domain.HashTokenValue(tokenValue)

	// Look up the token in the database
	token, err := a.store.TokenRepo().FindByHashedValue(ctx, hashedValue)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil, ErrTokenExpired
	}

	// Create a new identity
	return &auth.Identity{
		ID:   token.ID,
		Name: token.Name,
		Role: token.Role,
		Scope: auth.IdentityScope{
			ParticipantID: token.ParticipantID,
			AgentID:       token.AgentID,
		},
	}, nil
}

// Health checks if the token authenticator dependencies are healthy
func (a *GormTokenAuthenticator) Health(ctx context.Context) error {
	if a.store == nil {
		return fmt.Errorf("store is not initialized")
	}

	// Check if we can access the token repository
	tokenRepo := a.store.TokenRepo()
	if tokenRepo == nil {
		return fmt.Errorf("token repository is not available")
	}

	// Try to perform a simple count operation to verify database connectivity
	_, err := tokenRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to access token repository: %w", err)
	}

	return nil
}
