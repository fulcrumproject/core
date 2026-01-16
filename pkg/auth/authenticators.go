package auth

import (
	"context"
	"errors"
	"fmt"
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
			continue
		}
		if identity != nil {
			return identity, nil
		}
	}

	// All authenticators failed
	return nil, errors.New("authentication failed: no valid identity found")
}

// Health checks the health of all underlying authenticators
func (c *CompositeAuthenticator) Health(ctx context.Context) error {
	if len(c.authenticators) == 0 {
		return errors.New("no authenticators configured")
	}

	var healthErrors []error
	for i, authenticator := range c.authenticators {
		if err := authenticator.Health(ctx); err != nil {
			healthErrors = append(healthErrors, fmt.Errorf("authenticator %d: %w", i, err))
		}
	}

	if len(healthErrors) > 0 {
		return fmt.Errorf("authenticator health check failed: %v", healthErrors)
	}

	return nil
}
