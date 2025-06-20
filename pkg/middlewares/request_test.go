package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	tests := []struct {
		name           string
		urlParam       string
		expectedStatus int
		expectUUID     bool
		expectPanic    bool
	}{
		{
			name:           "Valid UUID",
			urlParam:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			expectUUID:     true,
			expectPanic:    false,
		},
		{
			name:           "Invalid UUID format",
			urlParam:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectUUID:     false,
			expectPanic:    false,
		},
		{
			name:           "Empty UUID",
			urlParam:       "",
			expectedStatus: http.StatusOK,
			expectUUID:     false,
			expectPanic:    false,
		},
		{
			name:           "Malformed UUID",
			urlParam:       "550e8400-e29b-41d4-a716",
			expectedStatus: http.StatusBadRequest,
			expectUUID:     false,
			expectPanic:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that checks if UUID is in context
			var capturedUUID properties.UUID
			var uuidFound bool
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectUUID {
					capturedUUID = MustGetID(r.Context())
					uuidFound = true
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := ID(testHandler)

			// Create request with chi context
			req := httptest.NewRequest("GET", "/test", nil)
			rctx := chi.NewRouteContext()
			if tt.urlParam != "" {
				rctx.URLParams.Add("id", tt.urlParam)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")

			if tt.expectUUID {
				assert.True(t, uuidFound, "UUID should be found in context")
				assert.NotEqual(t, properties.UUID{}, capturedUUID, "UUID should not be empty")
			}
		})
	}
}

func TestMustGetID(t *testing.T) {
	testUUID := properties.NewUUID()

	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectPanic bool
		expectedID  properties.UUID
	}{
		{
			name: "Valid UUID in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), uuidContextKey, testUUID)
			},
			expectPanic: false,
			expectedID:  testUUID,
		},
		{
			name: "No UUID in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectPanic: true,
		},
		{
			name: "Wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), uuidContextKey, "not-a-uuid")
			},
			expectPanic: true,
		},
		{
			name: "Nil value in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), uuidContextKey, nil)
			},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			if tt.expectPanic {
				assert.Panics(t, func() {
					MustGetID(ctx)
				}, "MustGetID should panic")
			} else {
				result := MustGetID(ctx)
				assert.Equal(t, tt.expectedID, result, "UUID should match expected")
			}
		})
	}
}

func TestDecodeBody(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name           string
		body           interface{}
		contentType    string
		expectedStatus int
		expectBody     bool
	}{
		{
			name: "Valid JSON body",
			body: TestStruct{
				Name:  "test",
				Value: 42,
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectBody:     true,
		},
		{
			name:           "Invalid JSON body",
			body:           `{"name": "test", "value": }`, // malformed JSON
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBody:     false,
		},
		{
			name:           "Empty body",
			body:           "",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBody:     false,
		},
		{
			name: "Valid body with extra fields",
			body: map[string]interface{}{
				"name":  "test",
				"value": 42,
				"extra": "ignored",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectBody:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			var bodyReader *bytes.Reader
			if str, ok := tt.body.(string); ok {
				bodyReader = bytes.NewReader([]byte(str))
			} else {
				bodyBytes, err := json.Marshal(tt.body)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			// Create test handler that checks if body is in context
			var capturedBody TestStruct
			var bodyFound bool
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectBody {
					capturedBody = MustGetBody[TestStruct](r.Context())
					bodyFound = true
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := DecodeBody[TestStruct]()(testHandler)

			// Create request
			req := httptest.NewRequest("POST", "/test", bodyReader)
			req.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")

			if tt.expectBody {
				assert.True(t, bodyFound, "Body should be found in context")
				if expectedStruct, ok := tt.body.(TestStruct); ok {
					assert.Equal(t, expectedStruct.Name, capturedBody.Name, "Name should match")
					assert.Equal(t, expectedStruct.Value, capturedBody.Value, "Value should match")
				}
			}
		})
	}
}

func TestMustGetBody(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	testStruct := TestStruct{Name: "test", Value: 42}
	testStructPtr := &testStruct

	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectPanic bool
		expected    TestStruct
	}{
		{
			name: "Valid struct pointer in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), decodedBodyContextKey, testStructPtr)
			},
			expectPanic: false,
			expected:    testStruct,
		},
		{
			name: "Valid struct value in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), decodedBodyContextKey, testStruct)
			},
			expectPanic: false,
			expected:    testStruct,
		},
		{
			name: "No body in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectPanic: true,
		},
		{
			name: "Nil body in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), decodedBodyContextKey, nil)
			},
			expectPanic: true,
		},
		{
			name: "Wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), decodedBodyContextKey, "not-a-struct")
			},
			expectPanic: true,
		},
		{
			name: "Wrong struct type in context",
			setupCtx: func() context.Context {
				type DifferentStruct struct {
					Other string
				}
				return context.WithValue(context.Background(), decodedBodyContextKey, DifferentStruct{Other: "test"})
			},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			if tt.expectPanic {
				assert.Panics(t, func() {
					MustGetBody[TestStruct](ctx)
				}, "MustGetBody should panic")
			} else {
				result := MustGetBody[TestStruct](ctx)
				assert.Equal(t, tt.expected, result, "Body should match expected")
			}
		})
	}
}

func TestIntegration_IDAndDecodeBody(t *testing.T) {
	type RequestBody struct {
		Name string `json:"name"`
	}

	// Test that both middlewares work together
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should be able to get both UUID and body
		id := MustGetID(r.Context())
		body := MustGetBody[RequestBody](r.Context())

		assert.NotEqual(t, properties.UUID{}, id, "UUID should be present")
		assert.Equal(t, "test", body.Name, "Body should be decoded correctly")

		w.WriteHeader(http.StatusOK)
	})

	// Chain middlewares
	handler := ID(DecodeBody[RequestBody]()(testHandler))

	// Create request
	bodyBytes, err := json.Marshal(RequestBody{Name: "test"})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add chi context with UUID
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "550e8400-e29b-41d4-a716-446655440000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should succeed with both middlewares")
}
