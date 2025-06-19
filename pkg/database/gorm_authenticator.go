package database

import (
	"context"
	"errors"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/core/pkg/domain"
)

var (
	ErrTokenExpired  = errors.New("token is expired")
	ErrTokenNotFound = errors.New("token not found")
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
		return nil, ErrTokenNotFound
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
