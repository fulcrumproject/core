package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test entity for standard handlers
type TestEntity struct {
	domain.BaseEntity
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Test response for standard handlers
type TestResponse struct {
	ID        properties.UUID `json:"id"`
	Name      string          `json:"name"`
	Status    string          `json:"status"`
	CreatedAt JSONUTCTime     `json:"createdAt"`
	UpdatedAt JSONUTCTime     `json:"updatedAt"`
}

// Response converter function
func testEntityToResponse(entity *TestEntity) *TestResponse {
	return &TestResponse{
		ID:        entity.ID,
		Name:      entity.Name,
		Status:    entity.Status,
		CreatedAt: JSONUTCTime(entity.CreatedAt),
		UpdatedAt: JSONUTCTime(entity.UpdatedAt),
	}
}

// Mock delete function
type mockDeleteFunc func(ctx context.Context, id properties.UUID) error

func TestList(t *testing.T) {
	testCases := []struct {
		name           string
		queryParams    string
		mockSetup      func(querier *BaseMockQuerier[TestEntity])
		expectedStatus int
		expectedItems  int
		expectError    bool
	}{
		{
			name:        "Success",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[TestEntity], error) {
					return &domain.PageResponse[TestEntity]{
						Items: []TestEntity{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:   "Test Entity 1",
								Status: "active",
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:   "Test Entity 2",
								Status: "inactive",
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
			expectedItems:  2,
			expectError:    false,
		},
		{
			name:        "InvalidPageParameter",
			queryParams: "?page=invalid",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				// Mock won't be called due to page parsing error
			},
			expectedStatus: http.StatusBadRequest,
			expectedItems:  0,
			expectError:    true,
		},
		{
			name:        "InvalidPageSizeParameter",
			queryParams: "?pageSize=invalid",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				// Mock won't be called due to pageSize parsing error
			},
			expectedStatus: http.StatusBadRequest,
			expectedItems:  0,
			expectError:    true,
		},
		{
			name:        "NegativePage",
			queryParams: "?page=-1",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				// Mock won't be called due to negative page validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedItems:  0,
			expectError:    true,
		},
		{
			name:        "ZeroPage",
			queryParams: "?page=0",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				// Mock won't be called due to zero page validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedItems:  0,
			expectError:    true,
		},
		{
			name:        "PageSizeTooLarge",
			queryParams: "?pageSize=1000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				// Mock won't be called due to pageSize exceeding max
			},
			expectedStatus: http.StatusBadRequest,
			expectedItems:  0,
			expectError:    true,
		},
		{
			name:        "ListError",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[TestEntity], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedItems:  0,
			expectError:    true,
		},
		{
			name:        "EmptyResult",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[TestEntity], error) {
					return &domain.PageResponse[TestEntity]{
						Items:       []TestEntity{},
						TotalItems:  0,
						CurrentPage: 1,
						TotalPages:  0,
						HasNext:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedItems:  0,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			querier := &BaseMockQuerier[TestEntity]{}
			tc.mockSetup(querier)

			// Create handler
			handler := List(querier, testEntityToResponse)

			// Create request
			req := httptest.NewRequest("GET", "/test"+tc.queryParams, nil)

			// Add auth identity to context
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if !tc.expectError {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				items, ok := response["items"].([]any)
				require.True(t, ok)
				assert.Equal(t, tc.expectedItems, len(items))

				if tc.expectedItems > 0 {
					firstItem := items[0].(map[string]any)
					assert.NotEmpty(t, firstItem["id"])
					assert.NotEmpty(t, firstItem["name"])
					assert.NotEmpty(t, firstItem["status"])
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *BaseMockQuerier[TestEntity])
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*TestEntity, error) {
					return &TestEntity{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:   "Test Entity",
						Status: "active",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*TestEntity, error) {
					return nil, domain.NewNotFoundErrorf("test entity not found")
				}
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "DatabaseError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*TestEntity, error) {
					return nil, fmt.Errorf("database connection error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "InvalidID",
			id:   "invalid-uuid",
			mockSetup: func(querier *BaseMockQuerier[TestEntity]) {
				// Mock won't be called due to ID parsing error in middleware
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			querier := &BaseMockQuerier[TestEntity]{}
			tc.mockSetup(querier)

			// Create handler
			handler := Get(querier, testEntityToResponse)

			// Create request
			req := httptest.NewRequest("GET", "/test/"+tc.id, nil)

			// Set up chi router context for URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Execute request with ID middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(handler)
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if !tc.expectError {
				var response TestResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.id, response.ID.String())
				assert.Equal(t, "Test Entity", response.Name)
				assert.Equal(t, "active", response.Status)
				assert.NotEmpty(t, response.CreatedAt)
				assert.NotEmpty(t, response.UpdatedAt)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *BaseMockQuerier[TestEntity], deleteFunc *mockDeleteFunc)
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity], deleteFunc *mockDeleteFunc) {
				querier.ExistsFunc = func(ctx context.Context, id properties.UUID) (bool, error) {
					return true, nil
				}
				*deleteFunc = func(ctx context.Context, id properties.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity], deleteFunc *mockDeleteFunc) {
				querier.ExistsFunc = func(ctx context.Context, id properties.UUID) (bool, error) {
					return false, nil
				}
				// deleteFunc won't be called
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "ExistsError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity], deleteFunc *mockDeleteFunc) {
				querier.ExistsFunc = func(ctx context.Context, id properties.UUID) (bool, error) {
					return false, fmt.Errorf("database connection error")
				}
				// deleteFunc won't be called
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *BaseMockQuerier[TestEntity], deleteFunc *mockDeleteFunc) {
				querier.ExistsFunc = func(ctx context.Context, id properties.UUID) (bool, error) {
					return true, nil
				}
				*deleteFunc = func(ctx context.Context, id properties.UUID) error {
					return fmt.Errorf("delete failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "InvalidID",
			id:   "invalid-uuid",
			mockSetup: func(querier *BaseMockQuerier[TestEntity], deleteFunc *mockDeleteFunc) {
				// Mocks won't be called due to ID parsing error in middleware
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &BaseMockQuerier[TestEntity]{}
			var deleteFunc mockDeleteFunc
			tc.mockSetup(querier, &deleteFunc)

			// Create handler
			handler := Delete(querier, deleteFunc)

			// Create request
			req := httptest.NewRequest("DELETE", "/test/"+tc.id, nil)

			// Set up chi router context for URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Execute request with ID middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(handler)
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if !tc.expectError {
				// For successful delete, body should be empty
				assert.Empty(t, w.Body.String())
			}
		})
	}
}

// Integration tests to verify all handlers work together
func TestStandardHandlersIntegration(t *testing.T) {
	// Create test data
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	testEntity := &TestEntity{
		BaseEntity: domain.BaseEntity{
			ID:        testID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:   "Integration Test Entity",
		Status: "active",
	}

	// Setup shared querier
	querier := &BaseMockQuerier[TestEntity]{
		ListFunc: func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[TestEntity], error) {
			return &domain.PageResponse[TestEntity]{
				Items:       []TestEntity{*testEntity},
				TotalItems:  1,
				CurrentPage: 1,
				TotalPages:  1,
				HasNext:     false,
			}, nil
		},
		GetFunc: func(ctx context.Context, id properties.UUID) (*TestEntity, error) {
			if id == testID {
				return testEntity, nil
			}
			return nil, domain.NewNotFoundErrorf("not found")
		},
		ExistsFunc: func(ctx context.Context, id properties.UUID) (bool, error) {
			return id == testID, nil
		},
	}

	deleteFunc := mockDeleteFunc(func(ctx context.Context, id properties.UUID) error {
		if id == testID {
			return nil
		}
		return fmt.Errorf("not found")
	})

	// Create handlers
	listHandler := List(querier, testEntityToResponse)
	getHandler := Get(querier, testEntityToResponse)
	deleteHandler := Delete(querier, deleteFunc)

	// Test List
	t.Run("List", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?page=1&pageSize=10", nil)
		authIdentity := NewMockAuthAdmin()
		req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

		w := httptest.NewRecorder()
		listHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		items := response["items"].([]any)
		assert.Equal(t, 1, len(items))
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/"+testID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		middlewareHandler := middlewares.ID(getHandler)
		middlewareHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response TestResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, testID, response.ID)
		assert.Equal(t, "Integration Test Entity", response.Name)
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/test/"+testID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", testID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		middlewareHandler := middlewares.ID(deleteHandler)
		middlewareHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})
}
