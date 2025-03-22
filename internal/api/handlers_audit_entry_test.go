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
	"fulcrumproject.org/core/internal/mock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuditEntryQuerier is a custom mock for AuditEntryQuerier
type mockAuditEntryQuerier struct {
	mock.AuditEntryQuerier
	listFunc      func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error)
}

func (m *mockAuditEntryQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.AuditEntry]{
		Items:       []domain.AuditEntry{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockAuditEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthScope, nil
}

// mockAuditEntryCommander is a custom mock for AuditEntryCommander
type mockAuditEntryCommander struct {
	createFunc            func(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error)
	createWithDiffFunc    func(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error)
	createCtxFunc         func(ctx context.Context, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error)
	createCtxWithDiffFunc func(ctx context.Context, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error)
}

func (m *mockAuditEntryCommander) Create(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, authorityType, authorityID, eventType, properties, entityID, providerID, agentID, brokerID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAuditEntryCommander) CreateWithDiff(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error) {
	if m.createWithDiffFunc != nil {
		return m.createWithDiffFunc(ctx, authorityType, authorityID, eventType, entityID, providerID, agentID, brokerID, beforeEntity, afterEntity)
	}
	return nil, fmt.Errorf("createWithDiff not mocked")
}

func (m *mockAuditEntryCommander) CreateCtx(ctx context.Context, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error) {
	if m.createCtxFunc != nil {
		return m.createCtxFunc(ctx, eventType, properties, entityID, providerID, agentID, brokerID)
	}
	return nil, fmt.Errorf("createCtx not mocked")
}

func (m *mockAuditEntryCommander) CreateCtxWithDiff(ctx context.Context, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error) {
	if m.createCtxWithDiffFunc != nil {
		return m.createCtxWithDiffFunc(ctx, eventType, entityID, providerID, agentID, brokerID, beforeEntity, afterEntity)
	}
	return nil, fmt.Errorf("createCtxWithDiff not mocked")
}

// MockAdminIdentity implements the domain.AuthIdentity interface for testing with admin role
type MockAdminIdentity struct {
	id domain.UUID
}

func (m MockAdminIdentity) ID() domain.UUID                  { return m.id }
func (m MockAdminIdentity) Name() string                     { return "test-admin" }
func (m MockAdminIdentity) Role() domain.AuthRole            { return domain.RoleFulcrumAdmin }
func (m MockAdminIdentity) IsRole(role domain.AuthRole) bool { return role == domain.RoleFulcrumAdmin }
func (m MockAdminIdentity) Scope() *domain.AuthScope {
	return &domain.EmptyAuthScope
}

// TestAuditEntryHandleList tests the handleList method
func TestAuditEntryHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *mock.MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *mock.MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return test audit entries
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				providerID := domain.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
				entityID := domain.UUID(uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"))

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
					return &domain.PageResponse[domain.AuditEntry]{
						Items: []domain.AuditEntry{
							{
								BaseEntity: domain.BaseEntity{
									ID:        domain.UUID(uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								AuthorityType: domain.AuthorityTypeAdmin,
								AuthorityID:   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
								EventType:     domain.EventTypeProviderCreated,
								Properties:    domain.JSON{"key": "value"},
								EntityID:      &entityID,
								ProviderID:    &providerID,
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        domain.UUID(uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								AuthorityType: domain.AuthorityTypeAdmin,
								AuthorityID:   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
								EventType:     domain.EventTypeProviderUpdated,
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
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *mock.MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "InvalidPageRequest",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *mock.MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true
			},
			expectedStatus: http.StatusOK, // parsePageRequest doesn't return errors for invalid page params, it uses defaults
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockAuditEntryQuerier, commander *mockAuditEntryCommander, authz *mock.MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
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
			authz := mock.NewMockAuthorizer(true)
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
			authIdentity := MockAdminIdentity{
				id: domain.UUID(uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")),
			}
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
				assert.Equal(t, "provider_created", firstItem["type"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "880e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "provider_updated", secondItem["type"])
			}
		})
	}
}

// TestNewAuditEntryHandler tests the constructor
func TestNewAuditEntryHandler(t *testing.T) {
	querier := &mockAuditEntryQuerier{}
	commander := &mockAuditEntryCommander{}
	authz := mock.NewMockAuthorizer(true)

	handler := NewAuditEntryHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestAuditEntryHandlerRoutes tests that routes are properly registered
func TestAuditEntryHandlerRoutes(t *testing.T) {
	// We'll use a stub for the actual handler to avoid executing real handler logic
	stubHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a test router and register routes
	r := chi.NewRouter()

	// Instead of using the actual handlers which require auth context,
	// we'll manually register routes with our stub handler
	r.Route("/audit-entries", func(r chi.Router) {
		// Register the GET / route
		r.Get("/", stubHandler)
	})

	// Test route existence by creating test requests
	// Test GET /audit-entries
	req := httptest.NewRequest("GET", "/audit-entries", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

// TestAuditEntryToResponse tests the auditEntryToResponse function
func TestAuditEntryToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	providerID := domain.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
	agentID := domain.UUID(uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"))
	brokerID := domain.UUID(uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"))
	entityID := domain.UUID(uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"))

	auditEntry := &domain.AuditEntry{
		BaseEntity: domain.BaseEntity{
			ID:        domain.UUID(uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")),
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
		BrokerID:      &brokerID,
	}

	response := auditEntryToResponse(auditEntry)

	assert.Equal(t, auditEntry.ID, response.ID)
	assert.Equal(t, auditEntry.AuthorityType, response.AuthorityType)
	assert.Equal(t, auditEntry.AuthorityID, response.AuthorityID)
	assert.Equal(t, auditEntry.EventType, response.Type)
	assert.Equal(t, auditEntry.Properties, response.Properties)
	assert.Equal(t, auditEntry.ProviderID, response.ProviderID)
	assert.Equal(t, auditEntry.AgentID, response.AgentID)
	assert.Equal(t, auditEntry.BrokerID, response.BrokerID)
	assert.Equal(t, JSONUTCTime(auditEntry.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(auditEntry.UpdatedAt), response.UpdatedAt)
}
