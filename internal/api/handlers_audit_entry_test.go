package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuditEntryHandleList tests the handleList method
func TestAuditEntryHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return test audit entries
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				providerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				entityID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
					return &domain.PageResponse[domain.AuditEntry]{
						Items: []domain.AuditEntry{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								AuthorityType: domain.AuthorityTypeAdmin,
								AuthorityID:   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
								EventType:     domain.EventTypeParticipantCreated,
								Properties:    domain.JSON{"key": "value"},
								EntityID:      &entityID,
								ProviderID:    &providerID,
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								AuthorityType: domain.AuthorityTypeAdmin,
								AuthorityID:   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
								EventType:     domain.EventTypeParticipantUpdated,
								Properties:    domain.JSON{"key": "updated"},
								EntityID:      &entityID,
								ProviderID:    &providerID,
							},
						},
						TotalItems:  2,
						CurrentPage: 1,
						TotalPages:  1,
						HasNext:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "InvalidPageRequest",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true
			},
			expectedStatus: http.StatusOK, // parsePageRequest doesn't return errors for invalid page params, it uses defaults
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAuditEntryQuerier{}
			commander := &mockAuditEntryCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewAuditEntryHandler(querier, commander, authz)

			// Create request
			var req *http.Request
			if tc.name == "InvalidPageRequest" {
				// Create an invalid page request
				req = httptest.NewRequest("GET", "/audit-entries?page=-1&pageSize=invalid", nil)
			} else {
				req = httptest.NewRequest("GET", "/audit-entries?page=1&pageSize=10", nil)
			}

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK && tc.name == "Success" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, float64(1), response["currentPage"])
				assert.Equal(t, float64(2), response["totalItems"])
				assert.Equal(t, float64(1), response["totalPages"])
				assert.Equal(t, false, response["hasNext"])
				assert.Equal(t, false, response["hasPrev"])

				items := response["items"].([]interface{})
				assert.Equal(t, 2, len(items))

				firstItem := items[0].(map[string]interface{})
				assert.Equal(t, "770e8400-e29b-41d4-a716-446655440000", firstItem["id"])
				assert.Equal(t, "admin", firstItem["authorityType"])
				assert.Equal(t, "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d", firstItem["authorityId"])
				assert.Equal(t, "participant_created", firstItem["type"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "880e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "participant_updated", secondItem["type"])
			}
		})
	}
}

// TestNewAuditEntryHandler tests the constructor
func TestNewAuditEntryHandler(t *testing.T) {
	querier := &mockAuditEntryQuerier{}
	commander := &mockAuditEntryCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewAuditEntryHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestAuditEntryHandlerRoutes tests that routes are properly registered
func TestAuditEntryHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockAuditEntryQuerier{}
	commander := &mockAuditEntryCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewAuditEntryHandler(querier, commander, authz)

	// Execute
	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	// Create a chi router and apply the routes
	r := chi.NewRouter()
	routeFunc(r)

	// Assert that endpoints are registered
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Check expected routes exist
		switch {
		case method == "GET" && route == "/":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestAuditEntryToResponse tests the auditEntryToResponse function
func TestAuditEntryToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	providerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	entityID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

	auditEntry := &domain.AuditEntry{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		AuthorityType: domain.AuthorityTypeAdmin,
		AuthorityID:   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
		EventType:     domain.EventTypeAgentCreated,
		Properties:    domain.JSON{"key": "value"},
		EntityID:      &entityID,
		ProviderID:    &providerID,
		AgentID:       &agentID,
		ConsumerID:    &consumerID,
	}

	response := auditEntryToResponse(auditEntry)

	assert.Equal(t, auditEntry.ID, response.ID)
	assert.Equal(t, auditEntry.AuthorityType, response.AuthorityType)
	assert.Equal(t, auditEntry.AuthorityID, response.AuthorityID)
	assert.Equal(t, auditEntry.EventType, response.Type)
	assert.Equal(t, auditEntry.Properties, response.Properties)
	assert.Equal(t, auditEntry.ProviderID, response.ProviderID)
	assert.Equal(t, auditEntry.AgentID, response.AgentID)
	assert.Equal(t, auditEntry.ConsumerID, response.ConsumerID)
	assert.Equal(t, JSONUTCTime(auditEntry.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(auditEntry.UpdatedAt), response.UpdatedAt)
}
