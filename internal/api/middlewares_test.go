package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIDMiddleware(t *testing.T) {
	// Create test cases
	tests := []struct {
		name           string
		urlParam       string
		expectedStatus int
		shouldHaveID   bool
	}{
		{
			name:           "Valid UUID",
			urlParam:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			shouldHaveID:   true,
		},
		{
			name:           "Invalid UUID",
			urlParam:       "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			shouldHaveID:   false,
		},
		{
			name:           "Empty UUID",
			urlParam:       "",
			expectedStatus: http.StatusNotFound, // Chi returns 404 when URL param doesn't match
			shouldHaveID:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedUUID domain.UUID
			var idInContext bool

			// Create a test handler that verifies the ID is in the context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Try to extract UUID value from context
				id, ok := r.Context().Value(uuidContextKey).(domain.UUID)
				idInContext = ok
				if ok {
					capturedUUID = id
				}

				// Always return success from the handler itself
				w.WriteHeader(http.StatusOK)
			})

			// Create a router and use the middleware
			r := chi.NewRouter()
			// Only add ID middleware in the route with ID
			r.Route("/{id}", func(r chi.Router) {
				r.Use(IDMiddleware)
				r.Get("/", testHandler)
			})

			// Create the request
			var req *http.Request
			var err error
			if tc.urlParam != "" {
				req, err = http.NewRequest("GET", "/"+tc.urlParam+"/", nil)
			} else {
				req, err = http.NewRequest("GET", "/", nil)
			}
			assert.NoError(t, err)

			// Execute the request
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Verify the expected status
			assert.Equal(t, tc.expectedStatus, w.Code)

			// If we expected a successful extraction of a UUID
			if tc.shouldHaveID {
				assert.True(t, idInContext, "UUID should be in the context")
				assert.Equal(t, tc.urlParam, capturedUUID.String(), "UUID should match expected value")
			}
		})
	}
}

func TestMustGetID(t *testing.T) {
	// Test the happy path
	testUUID := uuid.New()
	testID := domain.UUID(testUUID)
	r := httptest.NewRequest("GET", "/test", nil)

	// Set the ID in the context using the same key as IDMiddleware
	r = r.WithContext(context.WithValue(r.Context(), uuidContextKey, testID))

	// Call MustGetID
	id := MustGetID(r)
	assert.Equal(t, testUUID.String(), id.String())

	// Test the panic case by creating a sub-test to capture the panic
	t.Run("Panic case", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetID did not panic when ID was missing from context")
			}
		}()

		// Create a request without an ID in the context
		r := httptest.NewRequest("GET", "/test", nil)
		// This should panic
		_ = MustGetID(r)
	})
}
