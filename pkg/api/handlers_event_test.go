package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEventyHandler tests the constructor
func TestNewEventyHandler(t *testing.T) {
	querier := &mockEventQuerier{}
	eventSubscriptionCmd := &mockEventSubscriptionCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewEventHandler(querier, eventSubscriptionCmd, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, eventSubscriptionCmd, handler.eventSubscriptionCommander)
	assert.Equal(t, authz, handler.authz)
}

// TestEventyHandlerRoutes tests that routes are properly registered
func TestEventyHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockEventQuerier{}
	eventSubscriptionCmd := &mockEventSubscriptionCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewEventHandler(querier, eventSubscriptionCmd, authz)

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
		case method == "POST" && route == "/lease":
		case method == "POST" && route == "/ack":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestEventyToResponse tests the eventEntryToResponse function
func TestEventyToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	providerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	entityID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

	eventEntry := &domain.Event{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		InitiatorType: domain.InitiatorTypeUser,
		InitiatorID:   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
		Type:          domain.EventTypeAgentCreated,
		Payload:       properties.JSON{"key": "value"},
		EntityID:      &entityID,
		ProviderID:    &providerID,
		AgentID:       &agentID,
		ConsumerID:    &consumerID,
	}

	response := EventToRes(eventEntry)

	assert.Equal(t, eventEntry.ID, response.ID)
	assert.Equal(t, eventEntry.InitiatorType, response.InitiatorType)
	assert.Equal(t, eventEntry.InitiatorID, response.InitiatorID)
	assert.Equal(t, eventEntry.Type, response.Type)
	assert.Equal(t, eventEntry.Payload, response.Properties)
	assert.Equal(t, eventEntry.ProviderID, response.ProviderID)
	assert.Equal(t, eventEntry.AgentID, response.AgentID)
	assert.Equal(t, eventEntry.ConsumerID, response.ConsumerID)
	assert.Equal(t, JSONUTCTime(eventEntry.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(eventEntry.UpdatedAt), response.UpdatedAt)
}

// TestEventHandleLease tests the handleLease method
func TestEventHandleLease(t *testing.T) {
	testCases := []struct {
		name                     string
		requestBody              string
		mockEventSetup           func(querier *mockEventQuerier)
		mockSubscriptionSetup    func(cmd *mockEventSubscriptionCommander)
		expectedStatus           int
		expectedResponseContains map[string]any
	}{
		{
			name: "Success - lease acquired and events fetched",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1",
				"leaseDurationSeconds": 300,
				"limit": 10
			}`,
			mockEventSetup: func(querier *mockEventQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFromSequenceFunc = func(ctx context.Context, fromSequenceNumber int64, limit int) ([]*domain.Event, error) {
					return []*domain.Event{
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: createdAt,
								UpdatedAt: updatedAt,
							},
							SequenceNumber: 101,
							InitiatorType:  domain.InitiatorTypeUser,
							InitiatorID:    "user-123",
							Type:           domain.EventTypeParticipantCreated,
							Payload:        properties.JSON{"key": "value"},
						},
					}, nil
				}
			},
			mockSubscriptionSetup: func(cmd *mockEventSubscriptionCommander) {
				leaseExpiresAt := time.Now().Add(5 * time.Minute)
				cmd.acquireLeaseFunc = func(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error) {
					return &domain.EventSubscription{
						BaseEntity: domain.BaseEntity{
							ID: uuid.New(),
						},
						SubscriberID:               params.SubscriberID,
						LastEventSequenceProcessed: 100,
						LeaseOwnerInstanceID:       &params.InstanceID,
						LeaseExpiresAt:             &leaseExpiresAt,
						IsActive:                   true,
					}, nil
				}
			},
			expectedStatus: 200,
			expectedResponseContains: map[string]any{
				"lastEventSequenceProcessed": float64(100),
			},
		},
		{
			name: "Conflict - lease held by another instance",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1"
			}`,
			mockEventSetup: func(querier *mockEventQuerier) {
				// No setup needed for this test
			},
			mockSubscriptionSetup: func(cmd *mockEventSubscriptionCommander) {
				cmd.acquireLeaseFunc = func(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error) {
					return nil, domain.NewInvalidInputErrorf("lease is already held by instance instance-2")
				}
			},
			expectedStatus: 409,
		},
		{
			name: "Invalid request - missing subscriberId",
			requestBody: `{
				"instanceId": "instance-1"
			}`,
			mockEventSetup: func(querier *mockEventQuerier) {
				// No setup needed for this test
			},
			mockSubscriptionSetup: func(cmd *mockEventSubscriptionCommander) {
				// No setup needed for this test
			},
			expectedStatus: 400,
		},
		{
			name: "Invalid request - missing instanceId",
			requestBody: `{
				"subscriberId": "test-subscriber"
			}`,
			mockEventSetup: func(querier *mockEventQuerier) {
				// No setup needed for this test
			},
			mockSubscriptionSetup: func(cmd *mockEventSubscriptionCommander) {
				// No setup needed for this test
			},
			expectedStatus: 400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockEventQuerier{}
			eventSubscriptionCmd := &mockEventSubscriptionCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockEventSetup(querier)
			tc.mockSubscriptionSetup(eventSubscriptionCmd)

			// Create the handler
			handler := NewEventHandler(querier, eventSubscriptionCmd, authz)

			// Create request
			req := httptest.NewRequest("POST", "/events/lease", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context (required by all handlers)
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.Lease(rr, req)

			// Assert status code
			assert.Equal(t, tc.expectedStatus, rr.Code)

			if tc.expectedStatus == 200 {
				// Parse response for success cases
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				// Check expected response content
				for key, expectedValue := range tc.expectedResponseContains {
					assert.Equal(t, expectedValue, response[key])
				}

				// Verify response structure
				assert.Contains(t, response, "events")
				assert.Contains(t, response, "leaseExpiresAt")
				assert.Contains(t, response, "lastEventSequenceProcessed")
			}
		})
	}
}

// TestEventLeaseRequest_Bind tests the Bind method
func TestEventLeaseRequest_Bind(t *testing.T) {
	testCases := []struct {
		name        string
		request     EventLeaseReq
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			request: EventLeaseReq{
				SubscriberID: "test-subscriber",
				InstanceID:   "instance-1",
			},
			expectError: false,
		},
		{
			name: "Missing subscriberId",
			request: EventLeaseReq{
				InstanceID: "instance-1",
			},
			expectError: true,
			errorMsg:    "subscriberId is required",
		},
		{
			name: "Missing instanceId",
			request: EventLeaseReq{
				SubscriberID: "test-subscriber",
			},
			expectError: true,
			errorMsg:    "instanceId is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			err := tc.request.Bind(req)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEventHandleAcknowledge tests the handleAcknowledge method
func TestEventHandleAcknowledge(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    string
		setupMock      func(*mockEventSubscriptionCommander)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - events acknowledged",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1",
				"lastEventSequenceProcessed": 100
			}`,
			setupMock: func(cmd *mockEventSubscriptionCommander) {
				subscription := &domain.EventSubscription{
					SubscriberID:               "test-subscriber",
					LastEventSequenceProcessed: 100,
				}
				cmd.acknowledgeEventsFunc = func(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error) {
					return subscription, nil
				}
			},
			expectedStatus: 200,
			expectedBody:   `{"lastEventSequenceProcessed":100}`,
		},
		{
			name: "Conflict - no active lease",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1",
				"lastEventSequenceProcessed": 100
			}`,
			setupMock: func(cmd *mockEventSubscriptionCommander) {
				cmd.acknowledgeEventsFunc = func(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error) {
					return nil, domain.NewInvalidInputErrorf("no active lease found for subscriber %s", params.SubscriberID)
				}
			},
			expectedStatus: 409,
			expectedBody:   `"invalid input: no active lease found for subscriber test-subscriber"`,
		},
		{
			name: "Conflict - lease not owned by instance",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1",
				"lastEventSequenceProcessed": 100
			}`,
			setupMock: func(cmd *mockEventSubscriptionCommander) {
				cmd.acknowledgeEventsFunc = func(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error) {
					return nil, domain.NewInvalidInputErrorf("lease is not owned by instance %s", params.InstanceID)
				}
			},
			expectedStatus: 409,
			expectedBody:   `"invalid input: lease is not owned by instance instance-1"`,
		},
		{
			name: "Conflict - sequence regression",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1",
				"lastEventSequenceProcessed": 50
			}`,
			setupMock: func(cmd *mockEventSubscriptionCommander) {
				cmd.acknowledgeEventsFunc = func(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error) {
					return nil, domain.NewInvalidInputErrorf("cannot acknowledge sequence %d: must be greater than current sequence %d", params.LastEventSequenceProcessed, 100)
				}
			},
			expectedStatus: 409,
			expectedBody:   `"invalid input: cannot acknowledge sequence 50: must be greater than current sequence 100"`,
		},
		{
			name: "Invalid request - missing subscriberId",
			requestBody: `{
				"instanceId": "instance-1",
				"lastEventSequenceProcessed": 100
			}`,
			setupMock:      func(cmd *mockEventSubscriptionCommander) {},
			expectedStatus: 400,
			expectedBody:   `"subscriberId is required"`,
		},
		{
			name: "Invalid request - missing instanceId",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"lastEventSequenceProcessed": 100
			}`,
			setupMock:      func(cmd *mockEventSubscriptionCommander) {},
			expectedStatus: 400,
			expectedBody:   `"instanceId is required"`,
		},
		{
			name: "Invalid request - invalid lastEventSequenceProcessed",
			requestBody: `{
				"subscriberId": "test-subscriber",
				"instanceId": "instance-1",
				"lastEventSequenceProcessed": 0
			}`,
			setupMock:      func(cmd *mockEventSubscriptionCommander) {},
			expectedStatus: 400,
			expectedBody:   `"lastEventSequenceProcessed must be greater than 0"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockEventQuerier{}
			eventSubscriptionCmd := &mockEventSubscriptionCommander{}
			tc.setupMock(eventSubscriptionCmd)
			authz := &MockAuthorizer{ShouldSucceed: true}

			handler := NewEventHandler(querier, eventSubscriptionCmd, authz)

			// Create request
			req := httptest.NewRequest("POST", "/ack", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context (required by all handlers)
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.Acknowledge(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}

// TestEventAckRequest_Bind tests the Bind method
func TestEventAckRequest_Bind(t *testing.T) {
	testCases := []struct {
		name        string
		request     EventAckReq
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			request: EventAckReq{
				SubscriberID:               "test-subscriber",
				InstanceID:                 "instance-1",
				LastEventSequenceProcessed: 100,
			},
			expectError: false,
		},
		{
			name: "Missing subscriberId",
			request: EventAckReq{
				InstanceID:                 "instance-1",
				LastEventSequenceProcessed: 100,
			},
			expectError: true,
			errorMsg:    "subscriberId is required",
		},
		{
			name: "Missing instanceId",
			request: EventAckReq{
				SubscriberID:               "test-subscriber",
				LastEventSequenceProcessed: 100,
			},
			expectError: true,
			errorMsg:    "instanceId is required",
		},
		{
			name: "Invalid lastEventSequenceProcessed",
			request: EventAckReq{
				SubscriberID:               "test-subscriber",
				InstanceID:                 "instance-1",
				LastEventSequenceProcessed: 0,
			},
			expectError: true,
			errorMsg:    "lastEventSequenceProcessed must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			err := tc.request.Bind(req)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
