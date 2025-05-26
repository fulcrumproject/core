package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type contextKey string

const (
	uuidContextKey        = contextKey("uuid")
	decodedBodyContextKey = contextKey("decodedBody")
)

// Auth adds the identity to the context retrieving it from the authenticator
func Auth(auth domain.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" && !strings.HasPrefix(authHeader, "Bearer ") {
				render.Render(w, r, ErrUnauthenticated())
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			id := auth.Authenticate(r.Context(), token)
			if id == nil {
				render.Render(w, r, ErrUnauthenticated())
				return
			}
			ctx := domain.WithAuthIdentity(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ID extracts and validates the UUID from URL paths with /{id} format
func ID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idParam := chi.URLParam(r, "id")
		if idParam != "" {
			id, err := domain.ParseUUID(idParam)
			if err != nil {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}

			ctx := context.WithValue(r.Context(), uuidContextKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustGetID retrieves the UUID from the request context
func MustGetID(ctx context.Context) domain.UUID {
	id, ok := ctx.Value(uuidContextKey).(domain.UUID)
	if !ok {
		panic("UUID not found in request context")
	}
	return id
}

// DecodeBody is middleware that decodes the request body into a struct
// and stores it in the request context for later middlewares and handlers
func DecodeBody[T any]() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new instance of the target type
			v := new(T)

			// Decode the request body into the target
			if err := render.Decode(r, v); err != nil {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}

			// Store the decoded body in the context
			ctx := context.WithValue(r.Context(), decodedBodyContextKey, v)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MustGetBody retrieves and casts the decoded body to a specific type
func MustGetBody[T any](ctx context.Context) T {
	var zero T
	body := ctx.Value(decodedBodyContextKey)
	if body == nil {
		panic("no decoded body found in context")
	}

	// First try direct type assertion
	if typed, ok := body.(T); ok {
		return typed
	}

	// If that fails, try pointer dereferencing (DecodeBody stores *T)
	if ptr, ok := body.(*T); ok {
		return *ptr
	}

	panic(fmt.Sprintf("expected body of type %T or *%T, got %T", zero, zero, body))
}

// AuthTargetScopeExtractor defines a function type that extracts the auth target scope from a request
type AuthTargetScopeExtractor func(r *http.Request) (*domain.AuthTargetScope, error)

// AuthzFromExtractor is the base authorization middleware that uses a scope extractor function
// to get the authorization target scope from the request
func AuthzFromExtractor(
	subject domain.AuthSubject,
	action domain.AuthAction,
	authorizer domain.Authorizer,
	scopeExtractor AuthTargetScopeExtractor,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Get identity from context
			identity := domain.MustGetAuthIdentity(r.Context())

			// Extract scope using the provided extractor
			scope, err := scopeExtractor(r)
			if err != nil {
				render.Render(w, r, ErrUnauthorized(err))
				return
			}

			// Authorize action
			if err := authorizer.Authorize(identity, subject, action, scope); err != nil {
				render.Render(w, r, ErrUnauthorized(err))
				return
			}

			// Continue with updated context
			next.ServeHTTP(w, r)
		})
	}
}

// IDScopeExtractor creates an extractor that gets scope from a resource ID using a retriever
func IDScopeExtractor(scopeRetriever domain.AuthScopeRetriever) AuthTargetScopeExtractor {
	return func(r *http.Request) (*domain.AuthTargetScope, error) {
		// Get resource ID from URL
		id := MustGetID(r.Context())

		// Retrieve authorization scope for this resource
		scope, err := scopeRetriever.AuthScope(r.Context(), id)
		if err != nil {
			return nil, fmt.Errorf("resource not found: %w", err)
		}

		return scope, nil
	}
}

// AuthzFromID authorizes using a resource ID through the extractor pattern
func AuthzFromID(
	subject domain.AuthSubject,
	action domain.AuthAction,
	authorizer domain.Authorizer,
	scopeRetriever domain.AuthScopeRetriever,
) func(http.Handler) http.Handler {
	// Create an extractor that gets scope from the resource ID
	extractor := IDScopeExtractor(scopeRetriever)

	// Use the base AuthzFromExtractor with our specialized extractor
	return AuthzFromExtractor(subject, action, authorizer, extractor)
}

// SimpleScopeExtractor creates an extractor that always returns empty scope
func SimpleScopeExtractor() AuthTargetScopeExtractor {
	return func(r *http.Request) (*domain.AuthTargetScope, error) {
		// Use empty scope for simple operations
		return &domain.EmptyAuthTargetScope, nil
	}
}

// AuthzSimple authorizes without resource-specific scope through the extractor pattern
func AuthzSimple(
	subject domain.AuthSubject,
	action domain.AuthAction,
	authorizer domain.Authorizer,
) func(http.Handler) http.Handler {
	// Create an extractor that always returns empty scope
	extractor := SimpleScopeExtractor()

	// Use the base AuthzFromExtractor with our specialized extractor
	return AuthzFromExtractor(subject, action, authorizer, extractor)
}

// AuthTargetScopeProvider defines the interface for request types that can provide their own auth target scope
type AuthTargetScopeProvider interface {
	AuthTargetScope() (*domain.AuthTargetScope, error)
}

// BodyScopeExtractor creates an extractor that gets scope from the request body
func BodyScopeExtractor[T AuthTargetScopeProvider]() AuthTargetScopeExtractor {
	return func(r *http.Request) (*domain.AuthTargetScope, error) {
		// Get decoded body from context
		body := MustGetBody[T](r.Context())

		// Extract scope from body using its own method
		scope, err := body.AuthTargetScope()
		if err != nil {
			return nil, fmt.Errorf("invalid auth scope in request body: %w", err)
		}

		return scope, nil
	}
}

// AuthzFromBody middleware authorizes using the decoded body through the extractor pattern
// T must implement AuthTargetScopeProvider to provide its own target scope
func AuthzFromBody[T AuthTargetScopeProvider](
	subject domain.AuthSubject,
	action domain.AuthAction,
	authorizer domain.Authorizer,
) func(http.Handler) http.Handler {
	// Create an extractor that gets scope from the request body
	extractor := BodyScopeExtractor[T]()

	// Use the base AuthzFromExtractor with our specialized extractor
	return AuthzFromExtractor(subject, action, authorizer, extractor)
}

// RequireAgentIdentity ensures the request is from an agent
func RequireAgentIdentity() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity := domain.MustGetAuthIdentity(r.Context())

			if !identity.IsRole(domain.RoleAgent) {
				render.Render(w, r, ErrUnauthorized(errors.New("must be authenticated as agent")))
				return
			}

			if identity.Scope().AgentID == nil {
				render.Render(w, r, ErrUnauthorized(errors.New("agent with nil scope")))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MustGetAgentID works with RequireAgentIdentity
func MustGetAgentID(ctx context.Context) domain.UUID {
	id := domain.MustGetAuthIdentity(ctx)
	if !id.IsRole(domain.RoleAgent) {
		panic("must be authenticated as agent")
	}
	if id.Scope().AgentID == nil {
		panic("agent with nil scope")
	}
	return *id.Scope().AgentID
}
