package auth

import (
	"context"
	"errors"
)

// CompositeAuthenticator implements Authenticator by trying multiple authenticators in order
type CompositeAuthenticator struct {
	authenticators []Authenticator
}

// NewCompositeAuthenticator creates a new composite authenticator
func NewCompositeAuthenticator(authenticators ...Authenticator) *CompositeAuthenticator {
	return &CompositeAuthenticator{
		authenticators: authenticators,
	}
}

// Authenticate tries each authenticator in order until one succeeds
// Returns nil if all authenticators fail
func (c *CompositeAuthenticator) Authenticate(ctx context.Context, token string) (*Identity, error) {
	// Try each authenticator in order
	for _, authenticator := range c.authenticators {
		identity, err := authenticator.Authenticate(ctx, token)
		if err != nil {
			return nil, err
		}
		if identity != nil {
			return identity, nil
		}
	}

	// All authenticators failed
	return nil, errors.New("authentication failed: no valid identity found")
}
