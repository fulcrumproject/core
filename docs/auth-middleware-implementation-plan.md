# Auth Middleware Implementation Plan

## Overview

This document provides a concrete implementation plan for refactoring all remaining API handlers to use the established middleware pattern. The Agent handler ([`internal/api/handlers_agent.go`](internal/api/handlers_agent.go)) serves as the reference implementation.

## Reference Pattern (Agent Handler)

### Key Components Established

1. **Generic Body Decoder**: [`DecodeBody[T any]()`](internal/api/middlewares.go:73)
2. **Authorization Middlewares**:
   - [`AuthzSimple()`](internal/api/middlewares.go:145) - for simple operations
   - [`AuthzFromID()`](internal/api/middlewares.go:111) - for resource-based operations
   - [`AuthzFromBody[T AuthTargetScopeProvider]()`](internal/api/middlewares.go:179) - for body-based operations
3. **Request Types**: Implement [`AuthTargetScopeProvider`](internal/api/middlewares.go:171) interface
4. **Pure Handlers**: Use [`MustGetBody[T]()`](internal/api/middlewares.go:95) for type-safe body access

### Reference Route Pattern
```go
func (h *AgentHandler) Routes() func(r chi.Router) {
    return func(r chi.Router) {
        // List - simple authorization
        r.With(
            AuthzSimple(domain.SubjectAgent, domain.ActionRead, h.authz),
        ).Get("/", h.handleList)
        
        // Create - decode body + authorize from body
        r.With(
            DecodeBody[CreateAgentRequest](),
            AuthzFromBody[CreateAgentRequest](domain.SubjectAgent, domain.ActionCreate, h.authz),
        ).Post("/", h.handleCreate)
        
        // Resource-specific routes
        r.Group(func(r chi.Router) {
            r.Use(ID)
            
            // Get - authorize from resource ID
            r.With(
                AuthzFromID(domain.SubjectAgent, domain.ActionRead, h.authz, h.querier),
            ).Get("/{id}", h.handleGet)
            
            // Update - decode body + authorize from resource ID
            r.With(
                DecodeBody[UpdateAgentRequest](),
                AuthzFromID(domain.SubjectAgent, domain.ActionUpdate, h.authz, h.querier),
            ).Patch("/{id}", h.handleUpdate)
        })
    }
}
```

## Handler Test Migration Strategy

### Overview
All handler tests must be updated to align with the new middleware architecture. The Agent handler tests ([`internal/api/handlers_agent_test.go`](internal/api/handlers_agent_test.go)) serve as the reference implementation.

### Key Changes Required

#### 1. Remove Authorization Testing from Handler Tests
**Before**: Tests included authorization scenarios (success/failure)
```go
// ❌ Old pattern - testing authorization in handler
mockSetup: func(querier *mockQuerier, commander *mockCommander, authz *MockAuthorizer) {
    authz.ShouldSucceed = false // Testing authorization failure
}
expectedStatus: http.StatusForbidden
```

**After**: Focus only on business logic testing
```go
// ✅ New pattern - pure business logic testing
mockSetup: func(commander *mockCommander) {
    commander.createFunc = func(...) (*domain.Entity, error) {
        return mockEntity, nil // Test business logic only
    }
}
```

#### 2. Simulate Middleware Context Setup
**For handlers that expect decoded bodies**:
```go
// Simulate DecodeBody middleware
req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, requestStruct))
```

**For handlers that expect resource IDs**:
```go
// Simulate ID middleware
parsedUUID, _ := domain.ParseUUID(tc.id)
req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
```

**For all handlers**:
```go
// Add auth identity (required by all handlers)
authIdentity := NewMockAuthAgent() // or appropriate identity
req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))
```

#### 3. Test Structure Changes

**Handler Tests** (focus on business logic):
- Test successful operations
- Test domain/business errors
- Test data validation
- **Do NOT test authorization** (handled by middleware)

**Integration Tests** (test full middleware chain):
- Test complete request flow with middleware
- Test authorization scenarios
- Test middleware interactions

#### 4. Common Test Patterns

**Create Handler Test**:
```go
func TestHandleCreate(t *testing.T) {
    // Test cases focus on business logic
    testCases := []struct {
        name           string
        requestBody    CreateEntityRequest  // Use struct directly
        mockSetup      func(commander *mockCommander)
        expectedStatus int
    }{
        {
            name: "Success",
            requestBody: CreateEntityRequest{...},
            mockSetup: func(commander *mockCommander) {
                commander.createFunc = func(...) (*domain.Entity, error) {
                    return mockEntity, nil
                }
            },
            expectedStatus: http.StatusCreated,
        },
        {
            name: "BusinessLogicError",
            requestBody: CreateEntityRequest{...},
            mockSetup: func(commander *mockCommander) {
                commander.createFunc = func(...) (*domain.Entity, error) {
                    return nil, domain.NewInvalidInputErrorf("validation failed")
                }
            },
            expectedStatus: http.StatusBadRequest,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup mocks (no authorization mock needed)
            commander := &mockCommander{}
            tc.mockSetup(commander)
            
            handler := NewHandler(querier, commander, authz)
            
            // Create request with simulated middleware context
            req := httptest.NewRequest("POST", "/entities", nil)
            req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, tc.requestBody))
            req = req.WithContext(domain.WithAuthIdentity(req.Context(), NewMockAuthAgent()))
            
            // Execute and assert
            w := httptest.NewRecorder()
            handler.handleCreate(w, req)
            assert.Equal(t, tc.expectedStatus, w.Code)
        })
    }
}
```

### Migration Checklist for Each Handler

- [ ] Remove authorization test cases from handler tests
- [ ] Add middleware context simulation to all tests
- [ ] Focus test cases on business logic scenarios
- [ ] Add integration tests for middleware chain
- [ ] Update mock setup to exclude authorization mocks in handler tests
- [ ] Ensure all tests use appropriate mock identity types
- [ ] Verify `MustGetBody` usage with correct types

### Fixed Issues (Agent Handler Reference)

1. **Type Handling**: Fixed `MustGetBody` to handle pointer dereferencing from `DecodeBody` middleware
2. **Mock Functions**: Added `NewMockAuthAgentWithID` for agent-specific testing
3. **Test Structure**: Separated business logic tests from authorization tests
4. **Integration Testing**: Added full middleware chain testing

## Handlers to Refactor

### 1. Job Handler ([`internal/api/handlers_job.go`](internal/api/handlers_job.go))

**Current Issues**:
- Authorization logic embedded in handlers
- Missing middleware chain pattern
- No request types for POST operations

**Required Changes**:

#### Add Request Types
```go
type ClaimJobRequest struct {
    AgentID domain.UUID `json:"agentId"`
}

func (r ClaimJobRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    return &domain.AuthTargetScope{AgentID: &r.AgentID}, nil
}

type CompleteJobRequest struct {
    Result *string `json:"result,omitempty"`
}

type FailJobRequest struct {
    Error string `json:"error"`
}
```

#### Refactor Routes
```go
func (h *JobHandler) Routes() func(r chi.Router) {
    return func(r chi.Router) {
        // List jobs
        r.With(
            AuthzSimple(domain.SubjectJob, domain.ActionRead, h.authz),
        ).Get("/", h.handleList)
        
        // Agent job polling
        r.With(
            RequireAgentIdentity(),
            AuthzSimple(domain.SubjectJob, domain.ActionListPending, h.authz),
        ).Get("/pending", h.handleGetPendingJobs)
        
        // Resource-specific routes
        r.Group(func(r chi.Router) {
            r.Use(ID)
            
            // Get job
            r.With(
                AuthzFromID(domain.SubjectJob, domain.ActionRead, h.authz, h.querier),
            ).Get("/{id}", h.handleGet)
            
            // Agent actions
            r.With(
                RequireAgentIdentity(),
                DecodeBody[ClaimJobRequest](),
                AuthzFromBody[ClaimJobRequest](domain.SubjectJob, domain.ActionClaim, h.authz),
            ).Post("/{id}/claim", h.handleClaimJob)
            
            r.With(
                RequireAgentIdentity(),
                DecodeBody[CompleteJobRequest](),
                AuthzFromID(domain.SubjectJob, domain.ActionComplete, h.authz, h.querier),
            ).Post("/{id}/complete", h.handleCompleteJob)
            
            r.With(
                RequireAgentIdentity(),
                DecodeBody[FailJobRequest](),
                AuthzFromID(domain.SubjectJob, domain.ActionFail, h.authz, h.querier),
            ).Post("/{id}/fail", h.handleFailJob)
        })
    }
}
```

### 2. Service Handler ([`internal/api/handlers_service.go`](internal/api/handlers_service.go))

**Expected Request Types**:
```go
type CreateServiceRequest struct {
    Name         string            `json:"name"`
    ServiceTypeID domain.UUID      `json:"serviceTypeId"`
    ProviderID   domain.UUID       `json:"providerId"`
    ConsumerID   *domain.UUID      `json:"consumerId,omitempty"`
    Properties   domain.Properties `json:"properties,omitempty"`
    Attributes   domain.Attributes `json:"attributes,omitempty"`
}

func (r CreateServiceRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    return &domain.AuthTargetScope{
        ProviderID: &r.ProviderID,
        ConsumerID: r.ConsumerID,
    }, nil
}

type UpdateServiceRequest struct {
    Name       *string            `json:"name,omitempty"`
    Properties *domain.Properties `json:"properties,omitempty"`
    Attributes *domain.Attributes `json:"attributes,omitempty"`
}

type ServiceActionRequest struct {
    Action string `json:"action"` // "start", "stop"
}
```

### 3. Participant Handler ([`internal/api/handlers_participant.go`](internal/api/handlers_participant.go))

**Expected Request Types**:
```go
type CreateParticipantRequest struct {
    Name        string             `json:"name"`
    CountryCode domain.CountryCode `json:"countryCode,omitempty"`
    Attributes  domain.Attributes  `json:"attributes,omitempty"`
}

func (r CreateParticipantRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    // Only fulcrum_admin can create participants
    return &domain.EmptyAuthTargetScope, nil
}

type UpdateParticipantRequest struct {
    Name        *string             `json:"name,omitempty"`
    CountryCode *domain.CountryCode `json:"countryCode,omitempty"`
    Attributes  *domain.Attributes  `json:"attributes,omitempty"`
}
```

### 4. Token Handler ([`internal/api/handlers_token.go`](internal/api/handlers_token.go))

**Expected Request Types**:
```go
type CreateTokenRequest struct {
    Name         string        `json:"name"`
    Role         domain.AuthRole `json:"role"`
    ParticipantID *domain.UUID   `json:"participantId,omitempty"`
    AgentID      *domain.UUID   `json:"agentId,omitempty"`
    ExpiresAt    *time.Time    `json:"expireAt,omitempty"`
}

func (r CreateTokenRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    return &domain.AuthTargetScope{
        ParticipantID: r.ParticipantID,
        AgentID:      r.AgentID,
    }, nil
}
```

### 5. Service Group Handler ([`internal/api/handlers_service_group.go`](internal/api/handlers_service_group.go))

**Expected Request Types**:
```go
type CreateServiceGroupRequest struct {
    Name        string            `json:"name"`
    ProviderID  domain.UUID       `json:"providerId"`
    Attributes  domain.Attributes `json:"attributes,omitempty"`
}

func (r CreateServiceGroupRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    return &domain.AuthTargetScope{ProviderID: &r.ProviderID}, nil
}
```

### 6. Metric Entry Handler ([`internal/api/handlers_metric_entry.go`](internal/api/handlers_metric_entry.go))

**Expected Request Types**:
```go
type CreateMetricEntryRequest struct {
    MetricTypeID domain.UUID `json:"metricTypeId"`
    EntityType   string      `json:"entityType"`
    EntityID     domain.UUID `json:"entityId"`
    Value        float64     `json:"value"`
    Timestamp    time.Time   `json:"timestamp"`
}

func (r CreateMetricEntryRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    // Metrics are typically created by agents or the system
    return &domain.EmptyAuthTargetScope, nil
}
```

## Implementation Steps

### Phase 1: Job Handler (Priority 1)
1. Add request types with [`AuthTargetScopeProvider`](internal/api/middlewares.go:171)
2. Refactor [`Routes()`](internal/api/handlers_job.go:35) method
3. Remove authorization logic from handler methods
4. Update tests

### Phase 2: Service Handler (Priority 2)
1. Add request types
2. Refactor routes with complex authorization (provider/consumer scoping)
3. Handle service state transitions
4. Update tests

### Phase 3: Core Handlers (Priority 3)
1. Participant handler
2. Token handler
3. Service Group handler

### Phase 4: Supporting Handlers (Priority 4)
1. Metric Entry handler
2. Audit Entry handler (read-only)
3. Agent Type handler
4. Service Type handler
5. Metric Type handler

## Common Patterns

### Simple List Operations
```go
r.With(
    AuthzSimple(domain.SubjectX, domain.ActionRead, h.authz),
).Get("/", h.handleList)
```

### Create Operations
```go
r.With(
    DecodeBody[CreateXRequest](),
    AuthzFromBody[CreateXRequest](domain.SubjectX, domain.ActionCreate, h.authz),
).Post("/", h.handleCreate)
```

### Resource Operations
```go
r.Group(func(r chi.Router) {
    r.Use(ID)
    
    r.With(
        AuthzFromID(domain.SubjectX, domain.ActionRead, h.authz, h.querier),
    ).Get("/{id}", h.handleGet)
    
    r.With(
        DecodeBody[UpdateXRequest](),
        AuthzFromID(domain.SubjectX, domain.ActionUpdate, h.authz, h.querier),
    ).Patch("/{id}", h.handleUpdate)
})
```

### Agent-Only Operations
```go
r.With(
    RequireAgentIdentity(),
    DecodeBody[XRequest](),
    AuthzFromBody[XRequest](domain.SubjectX, domain.ActionY, h.authz),
).Post("/action", h.handleAction)
```

## Testing Strategy

For each refactored handler:

1. **Middleware Tests**: Test authorization middleware behavior
2. **Integration Tests**: Test complete request flow with middleware chain
3. **Handler Tests**: Test pure business logic (mocked authorization)

## Success Criteria

- [ ] All handlers use middleware chain pattern
- [ ] No authorization logic in handler methods
- [ ] All request types implement [`AuthTargetScopeProvider`](internal/api/middlewares.go:171)
- [ ] Comprehensive test coverage
- [ ] Consistent error handling
- [ ] Type-safe body handling with generics
