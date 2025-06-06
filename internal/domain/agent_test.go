package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentStatus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		status  AgentStatus
		wantErr bool
	}{
		{
			name:    "New status",
			status:  AgentNew,
			wantErr: false,
		},
		{
			name:    "Connected status",
			status:  AgentConnected,
			wantErr: false,
		},
		{
			name:    "Disconnected status",
			status:  AgentDisconnected,
			wantErr: false,
		},
		{
			name:    "Error status",
			status:  AgentError,
			wantErr: false,
		},
		{
			name:    "Disabled status",
			status:  AgentDisabled,
			wantErr: false,
		},
		{
			name:    "Invalid status",
			status:  "InvalidStatus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseAgentStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    AgentStatus
		wantErr bool
	}{
		{
			name:    "Parse New status",
			value:   "New",
			want:    AgentNew,
			wantErr: false,
		},
		{
			name:    "Parse Connected status",
			value:   "Connected",
			want:    AgentConnected,
			wantErr: false,
		},
		{
			name:    "Parse Disconnected status",
			value:   "Disconnected",
			want:    AgentDisconnected,
			wantErr: false,
		},
		{
			name:    "Parse Error status",
			value:   "Error",
			want:    AgentError,
			wantErr: false,
		},
		{
			name:    "Parse Disabled status",
			value:   "Disabled",
			want:    AgentDisabled,
			wantErr: false,
		},
		{
			name:    "Parse invalid status",
			value:   "InvalidStatus",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAgentStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAgent_TableName(t *testing.T) {
	agent := Agent{}
	assert.Equal(t, "agents", agent.TableName())
}

func TestAgent_Validate(t *testing.T) {
	validID := uuid.New()
	validTime := time.Now()

	tests := []struct {
		name    string
		agent   *Agent
		wantErr bool
	}{
		{
			name: "Valid agent",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			agent: &Agent{
				Name:             "",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Invalid status",
			agent: &Agent{
				Name:             "test-agent",
				Status:           "InvalidStatus",
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Zero time",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: time.Time{},
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Empty agent type ID",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      uuid.Nil,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Empty participant ID",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       uuid.Nil,
			},
			wantErr: true,
		},
		{
			name: "Valid agent",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// MockAuditEntryCommander is a mock implementation of AuditEntryCommander
// Using the implementation from mock_commander_test.go

func TestAgentCommander_Create(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New()
	validName := "test-agent"

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				agentTypeRepo := &MockAgentTypeRepository{}
				agentTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				agentRepo := &MockAgentRepository{}
				agentRepo.createFunc = func(ctx context.Context, agent *Agent) error {
					assert.Equal(t, validName, agent.Name)
					assert.Equal(t, validID, agent.ProviderID)
					assert.Equal(t, validID, agent.AgentTypeID)
					assert.Equal(t, AgentDisconnected, agent.Status)
					return nil
				}

				store.WithParticipantRepo(participantRepo)
				store.WithAgentTypeRepo(agentTypeRepo)
				store.WithAgentRepo(agentRepo)
			},
			wantErr: false,
		},
		{
			name: "Participant not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}

				store.WithParticipantRepo(participantRepo)
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				var invalidInputErr InvalidInputError
				require.True(t, errors.As(err, &invalidInputErr))
				assert.Contains(t, err.Error(), "provider with ID")
			},
		},
		{
			name: "Agent type not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				agentTypeRepo := &MockAgentTypeRepository{}
				agentTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}

				store.WithParticipantRepo(participantRepo)
				store.WithAgentTypeRepo(agentTypeRepo)
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				var invalidInputErr InvalidInputError
				require.True(t, errors.As(err, &invalidInputErr))
				assert.Contains(t, err.Error(), "agent type with ID")
			},
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				agentTypeRepo := &MockAgentTypeRepository{}
				agentTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				// Force an agent validation error with a bad country code
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return InvalidInputError{Err: errors.New("validation error")}
				}

				store.WithParticipantRepo(participantRepo)
				store.WithAgentTypeRepo(agentTypeRepo)
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				var invalidInputErr InvalidInputError
				require.True(t, errors.As(err, &invalidInputErr))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewAgentCommander(store, audit)
			agent, err := commander.Create(ctx, validName, validID, validID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, validName, agent.Name)
			}
		})
	}
}

func TestAgentCommander_Update(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	providerID := uuid.New()
	agentTypeID := uuid.New()
	existingName := "existing-agent"
	newName := "updated-agent"
	existingStatus := AgentDisconnected
	newStatus := AgentConnected

	existingAgent := &Agent{
		BaseEntity: BaseEntity{
			ID: agentID,
		},
		Name:             existingName,
		Status:           existingStatus,
		LastStatusUpdate: time.Now(),
		ProviderID:       providerID,
		AgentTypeID:      agentTypeID,
	}

	tests := []struct {
		testName string
		// Create a deep copy of existingAgent for each test
		setupAgentCopy func() *Agent
		setupMocks     func(store *MockStore, audit *MockAuditEntryCommander)
		nameParam      *string
		statusParam    *AgentStatus
		wantErr        bool
	}{
		{
			testName: "Update name",
			setupAgentCopy: func() *Agent {
				copy := *existingAgent
				return &copy
			},
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					copy := *existingAgent
					return &copy, nil
				}
				agentRepo.updateFunc = func(ctx context.Context, agent *Agent) error {
					return nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}

				store.WithAgentRepo(agentRepo)
			},
			nameParam:   &newName,
			statusParam: nil,
			wantErr:     false,
		},
		{
			testName: "Update status",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					copy := *existingAgent
					return &copy, nil
				}
				agentRepo.updateFunc = func(ctx context.Context, agent *Agent) error {
					return nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}

				store.WithAgentRepo(agentRepo)
			},
			nameParam:   nil,
			statusParam: &newStatus,
			wantErr:     false,
		},
		{
			testName: "Agent not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return nil, NewNotFoundErrorf("agent with ID %s not found", id.String())
				}

				store.WithAgentRepo(agentRepo)
			},
			nameParam:   &newName,
			statusParam: nil,
			wantErr:     true,
		},
		{
			testName: "Update validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					copy := *existingAgent
					return &copy, nil
				}

				// Make atomic return an error during validation
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return InvalidInputError{Err: errors.New("validation error")}
				}

				store.WithAgentRepo(agentRepo)
			},
			nameParam:   func() *string { s := ""; return &s }(), // Empty name to cause validation error
			statusParam: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewAgentCommander(store, audit)
			agent, err := commander.Update(ctx, agentID, tt.nameParam, tt.statusParam)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)

				if tt.nameParam != nil {
					assert.Equal(t, *tt.nameParam, agent.Name)
				} else {
					assert.Equal(t, existingName, agent.Name)
				}

				if tt.statusParam != nil {
					assert.Equal(t, *tt.statusParam, agent.Status)
				} else {
					assert.Equal(t, existingStatus, agent.Status)
				}
			}
		})
	}
}

func TestAgent_UpdateStatus(t *testing.T) {
	now := time.Now()
	agent := &Agent{
		Status:           AgentDisconnected,
		LastStatusUpdate: now.Add(-time.Hour), // Set to 1 hour ago
	}

	agent.UpdateStatus(AgentConnected)

	assert.Equal(t, AgentConnected, agent.Status)
	assert.True(t, agent.LastStatusUpdate.After(now), "LastStatusUpdate should be updated to a newer time")
}

func TestAgent_UpdateHeartbeat(t *testing.T) {
	now := time.Now()
	agent := &Agent{
		Status:           AgentConnected,
		LastStatusUpdate: now.Add(-time.Hour), // Set to 1 hour ago
	}

	agent.UpdateHeartbeat()

	assert.Equal(t, AgentConnected, agent.Status, "Status should not change")
	assert.True(t, agent.LastStatusUpdate.After(now), "LastStatusUpdate should be updated to a newer time")
}

func TestAgent_RegisterMetadata(t *testing.T) {
	agent := &Agent{
		Name: "original-name",
	}

	// Test updating the name
	newName := "new-name"
	agent.RegisterMetadata(&newName)

	assert.Equal(t, newName, agent.Name)

	// Test updating to another name
	newerName := "newer-name"
	agent.RegisterMetadata(&newerName)

	assert.Equal(t, newerName, agent.Name)
}

func TestAgentCommander_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	providerID := uuid.New()
	currentTime := time.Now().Add(-time.Hour)
	agentName := "test-agent"

	existingAgent := &Agent{
		BaseEntity: BaseEntity{
			ID: agentID,
		},
		Name:        agentName,
		ProviderID:  providerID,
		Status:      AgentDisconnected,
		AgentTypeID: uuid.New(),
	}

	tests := []struct {
		name        string
		setupMocks  func(store *MockStore, audit *MockAuditEntryCommander)
		newStatus   AgentStatus
		wantErr     bool
		errorString string
	}{
		{
			name: "Update status success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					require.Equal(t, agentID, id)
					copy := *existingAgent
					copy.LastStatusUpdate = currentTime
					return &copy, nil
				}
				agentRepo.updateFunc = func(ctx context.Context, agent *Agent) error {
					assert.Equal(t, agentID, agent.ID)
					assert.Equal(t, AgentConnected, agent.Status)
					assert.True(t, agent.LastStatusUpdate.After(currentTime))
					return nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}

				store.WithAgentRepo(agentRepo)
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID, before interface{}, after interface{}) (*AuditEntry, error) {
					return &AuditEntry{}, nil
				}
			},
			newStatus: AgentConnected,
			wantErr:   false,
		},
		{
			name: "Agent not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return nil, NewNotFoundErrorf("agent not found")
				}
				store.WithAgentRepo(agentRepo)
			},
			newStatus:   AgentConnected,
			wantErr:     true,
			errorString: "agent not found",
		},
		{
			name: "Invalid status",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					copy := *existingAgent
					return &copy, nil
				}

				// Make atomic return an error during validation
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return InvalidInputError{Err: errors.New("invalid status")}
				}

				store.WithAgentRepo(agentRepo)
			},
			newStatus:   "InvalidStatus",
			wantErr:     true,
			errorString: "invalid agent status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewAgentCommander(store, audit)
			agent, err := commander.UpdateStatus(ctx, agentID, tt.newStatus)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tt.newStatus, agent.Status)
				assert.True(t, agent.LastStatusUpdate.After(currentTime))
			}
		})
	}
}

func TestAgentCommander_Delete(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	providerID := uuid.New()

	tests := []struct {
		name          string
		setupMocks    func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr       bool
		expectedError string
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					require.Equal(t, agentID, id)
					return &Agent{
						BaseEntity: BaseEntity{ID: agentID},
						ProviderID: providerID,
					}, nil
				}
				serviceRepo := &MockServiceRepository{}
				serviceRepo.countByAgentFunc = func(ctx context.Context, id UUID) (int64, error) {
					require.Equal(t, agentID, id)
					return 0, nil
				}
				tokenRepo := &MockTokenRepository{}
				tokenRepo.deleteByAgentIDFunc = func(ctx context.Context, id UUID) error {
					require.Equal(t, agentID, id)
					return nil
				}
				agentRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					require.Equal(t, agentID, id)
					return nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}

				store.WithAgentRepo(agentRepo)
				store.WithServiceRepo(serviceRepo)
				store.WithTokenRepo(tokenRepo)

				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID) (*AuditEntry, error) {
					assert.NotNil(t, entityID)
					assert.Equal(t, agentID, entityID)
					return &AuditEntry{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Agent not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return nil, NewNotFoundErrorf("agent not found")
				}
				store.WithAgentRepo(agentRepo)
			},
			wantErr:       true,
			expectedError: "agent not found",
		},
		{
			name: "Has dependent services",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return &Agent{BaseEntity: BaseEntity{ID: agentID}}, nil
				}
				serviceRepo := &MockServiceRepository{}
				serviceRepo.countByAgentFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 5, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}

				store.WithAgentRepo(agentRepo)
				store.WithServiceRepo(serviceRepo)
			},
			wantErr:       true,
			expectedError: "cannot delete agent with associated services",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewAgentCommander(store, audit)
			err := commander.Delete(ctx, agentID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
