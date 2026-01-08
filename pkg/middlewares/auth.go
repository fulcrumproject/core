package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/response"
	"github.com/go-chi/render"
)

var (
	ErrUnauthorized     = errors.New("invalid token format, expected 'Bearer <token>'")
	ErrIdentityNotFound = errors.New("identity not found")
)

// Auth adds the identity to the context retrieving it from the authenticator
func Auth(authenticator auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				render.Render(w, r, response.ErrUnauthenticated(ErrUnauthorized))
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			id, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				render.Render(w, r, response.ErrUnauthorized(err))
				return
			}
			if id == nil {
				render.Render(w, r, response.ErrUnauthorized(ErrIdentityNotFound))
				return
			}
			ctx := auth.WithIdentity(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ObjectScopeExtractor defines a function type that extracts the auth target scope from a request
type ObjectScopeExtractor func(r *http.Request) (authz.ObjectScope, error)

// ObjectScopeLoader defines a function type that  retrieves the authorization scope for a resource ID
type ObjectScopeLoader func(ctx context.Context, id properties.UUID) (authz.ObjectScope, error)

// ObjectScopeProvider defines an interface for types that can provide their own auth target scope
type ObjectScopeProvider interface {
	ObjectScope() (authz.ObjectScope, error)
}

// AuthzFromExtractor is the base authorization middleware that uses a scope extractor function
// to get the authorization target scope from the request
func AuthzFromExtractor(
	object authz.ObjectType,
	action authz.Action,
	authorizer authz.Authorizer,
	extractor ObjectScopeExtractor,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Get identity from context
			identity := auth.MustGetIdentity(r.Context())

			// Extract scope using the provided extractor
			scope, err := extractor(r)
			if err != nil {
				render.Render(w, r, response.ErrUnauthorized(err))
				return
			}

			// Authorize action
			if err := authorizer.Authorize(identity, action, object, scope); err != nil {
				render.Render(w, r, response.ErrUnauthorized(err))
				return
			}

			// Continue with updated context
			next.ServeHTTP(w, r)
		})
	}
}

// IDScopeExtractor creates an extractor that gets scope from a resource ID using a retriever
func IDScopeExtractor(loader ObjectScopeLoader) ObjectScopeExtractor {
	return func(r *http.Request) (authz.ObjectScope, error) {
		// Get resource ID from URL
		id := MustGetID(r.Context())

		// Retrieve authorization scope for this resource
		scope, err := loader(r.Context(), id)
		if err != nil {
			return nil, fmt.Errorf("cannot load resource: %w", err)
		}

		return scope, nil
	}
}

// AuthzFromID authorizes using a resource ID through the extractor pattern
func AuthzFromID(
	object authz.ObjectType,
	action authz.Action,
	authorizer authz.Authorizer,
	loader ObjectScopeLoader,
) func(http.Handler) http.Handler {
	// Create an extractor that gets scope from the resource ID
	extractor := IDScopeExtractor(loader)

	// Use the base AuthzFromExtractor with our specialized extractor
	return AuthzFromExtractor(object, action, authorizer, extractor)
}

// SimpleScopeExtractor creates an extractor that always returns empty scope
func SimpleScopeExtractor() ObjectScopeExtractor {
	return func(r *http.Request) (authz.ObjectScope, error) {
		// Use empty scope for simple operations
		return &authz.AllwaysMatchObjectScope{}, nil
	}
}

// AuthzSimple authorizes without resource-specific scope through the extractor pattern
func AuthzSimple(
	object authz.ObjectType,
	action authz.Action,
	authorizer authz.Authorizer,
) func(http.Handler) http.Handler {
	// Create an extractor that always returns empty scope
	extractor := SimpleScopeExtractor()

	// Use the base AuthzFromExtractor with our specialized extractor
	return AuthzFromExtractor(object, action, authorizer, extractor)
}

// BodyScopeExtractor creates an extractor that gets scope from the request body
func BodyScopeExtractor[T ObjectScopeProvider]() ObjectScopeExtractor {
	return func(r *http.Request) (authz.ObjectScope, error) {
		// Get decoded body from context
		body := MustGetBody[T](r.Context())

		// Extract scope from body using its own method
		scope, err := body.ObjectScope()
		if err != nil {
			return nil, fmt.Errorf("invalid auth scope in request body: %w", err)
		}

		return scope, nil
	}
}

// AuthzFromBody middleware authorizes using the decoded body through the extractor pattern
// T must implement AuthTargetScopeProvider to provide its own target scope
func AuthzFromBody[T ObjectScopeProvider](
	object authz.ObjectType,
	action authz.Action,
	authorizer authz.Authorizer,
) func(http.Handler) http.Handler {
	// Create an extractor that gets scope from the request body
	extractor := BodyScopeExtractor[T]()

	// Use the base AuthzFromExtractor with our specialized extractor
	return AuthzFromExtractor(object, action, authorizer, extractor)
}

// MustHaveRoles creates a middleware that ensures the authenticated user has at least one of the required roles
func MustHaveRoles(roles ...auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get identity from context
			identity := auth.MustGetIdentity(r.Context())

			// Check if user has any of the required roles
			hasRequiredRole := false
			for _, role := range roles {
				if identity.HasRole(role) {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				err := fmt.Errorf("access denied: user role '%s' is not authorized", identity.Role)
				render.Render(w, r, response.ErrUnauthorized(err))
				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}
