package auth

import "context"

type authContextKey string

const (
	identityContextKey = authContextKey("identity")
)

// WithIdentity adds to the context the identity
func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityContextKey, id)
}

// MustGetIdentity retrieves the authenticated identity from the request context
func MustGetIdentity(ctx context.Context) *Identity {
	id, ok := ctx.Value(identityContextKey).(*Identity)
	if !ok || id == nil {
		panic("cannot find identity in context")
	}
	return id
}
