// Vault handler tests
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVaultHandler_GetSecret(t *testing.T) {
	// Setup
	mockVault := schema.NewMockVault(t)
	handler := NewVaultHandler(mockVault)

	agentID := properties.NewUUID()
	agentIdentity := &auth.Identity{
		ID:   agentID,
		Name: "test-agent",
		Role: auth.RoleAgent,
		Scope: auth.IdentityScope{
			ParticipantID: nil,
			AgentID:       &agentID,
		},
	}

	tests := []struct {
		name           string
		reference      string
		setupMock      func()
		expectedStatus int
		expectedValue  any
	}{
		{
			name:      "successful secret retrieval - string value",
			reference: "abc123def456",
			setupMock: func() {
				mockVault.On("Get", mock.Anything, "abc123def456").
					Return("my-secret-password", nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedValue:  "my-secret-password",
		},
		{
			name:      "successful secret retrieval - object value",
			reference: "xyz789",
			setupMock: func() {
				mockVault.On("Get", mock.Anything, "xyz789").
					Return(map[string]any{"username": "admin", "password": "secret"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedValue:  map[string]any{"username": "admin", "password": "secret"},
		},
		{
			name:      "secret not found",
			reference: "nonexistent",
			setupMock: func() {
				mockVault.On("Get", mock.Anything, "nonexistent").
					Return(nil, assert.AnError).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedValue:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.setupMock()

			// Create request with agent identity in context
			req := httptest.NewRequest(http.MethodGet, "/vault/secrets/"+tt.reference, nil)
			ctx := auth.WithIdentity(req.Context(), agentIdentity)
			req = req.WithContext(ctx)

			// Add URL param (simulating chi router)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("reference", tt.reference)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Execute request
			w := httptest.NewRecorder()
			handler.GetSecret(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var res GetSecretRes
				err := json.NewDecoder(w.Body).Decode(&res)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValue, res.Value)
			}

			// Verify mock expectations
			mockVault.AssertExpectations(t)
		})
	}
}

func TestVaultHandler_GetSecret_EmptyReference(t *testing.T) {
	// Setup
	mockVault := schema.NewMockVault(t)
	handler := NewVaultHandler(mockVault)

	agentID := properties.NewUUID()
	agentIdentity := &auth.Identity{
		ID:   agentID,
		Name: "test-agent",
		Role: auth.RoleAgent,
		Scope: auth.IdentityScope{
			ParticipantID: nil,
			AgentID:       &agentID,
		},
	}

	// Create request with empty reference
	req := httptest.NewRequest(http.MethodGet, "/vault/secrets/", nil)
	ctx := auth.WithIdentity(req.Context(), agentIdentity)
	req = req.WithContext(ctx)

	// Add empty URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("reference", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute request
	w := httptest.NewRecorder()
	handler.GetSecret(w, req)

	// Verify bad request response
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNewVaultHandler(t *testing.T) {
	mockVault := schema.NewMockVault(t)
	handler := NewVaultHandler(mockVault)

	assert.NotNil(t, handler)
	assert.Equal(t, mockVault, handler.vault)
}
