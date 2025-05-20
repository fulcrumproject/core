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

func TestAgentState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		state   AgentState
		wantErr bool
	}{
		{
			name:    "New state",
			state:   AgentNew,
			wantErr: false,
		},
		{
			name:    "Connected state",
			state:   AgentConnected,
			wantErr: false,
		},
		{
			name:    "Disconnected state",
			state:   AgentDisconnected,
			wantErr: false,
		},
		{
			name:    "Error state",
			state:   AgentError,
			wantErr: false,
		},
		{
			name:    "Disabled state",
			state:   AgentDisabled,
			wantErr: false,
		},
		{
			name:    "Invalid state",
			state:   "InvalidState",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgent_TableName(t *testing.T) {
	agent := Agent{}
	assert.Equal(t, "agents", agent.TableName())
}

func TestParseAgentState(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    AgentState
		wantErr bool
	}{
		{
			name:    "Parse New state",
			value:   "New",
			want:    AgentNew,
			wantErr: false,
		},
		{
			name:    "Parse Connected state",
			value:   "Connected",
			want:    AgentConnected,
			wantErr: false,
		},
		{
			name:    "Parse Disconnected state",
			value:   "Disconnected",
			want:    AgentDisconnected,
			wantErr: false,
		},
		{
			name:    "Parse Error state",
			value:   "Error",
			want:    AgentError,
			wantErr: false,
		},
		{
			name:    "Parse Disabled state",
			value:   "Disabled",
			want:    AgentDisabled,
			wantErr: false,
		},
		{
			name:    "Parse invalid state",
			value:   "InvalidState",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAgentState(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
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
				Name:            "test-agent",
				State:           AgentConnected,
				LastStateUpdate: validTime,
				AgentTypeID:     validID,
				ProviderID:      validID,
				CountryCode:     "US",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			agent: &Agent{
				Name:            "",
				State:           AgentConnected,
				LastStateUpdate: validTime,
				AgentTypeID:     validID,
				ProviderID:      validID,
				CountryCode:     "US",
			},
			wantErr: true,
		},
		{
			name: "Invalid state",
			agent: &Agent{
				Name:            "test-agent",
				State:           "InvalidState",
				LastStateUpdate: validTime,
				AgentTypeID:     validID,
				ProviderID:      validID,
				CountryCode:     "US",
			},
			wantErr: true,
		},
		{
			name: "Zero time",
			agent: &Agent{
				Name:            "test-agent",
				State:           AgentConnected,
				LastStateUpdate: time.Time{},
				AgentTypeID:     validID,
				ProviderID:      validID,
				CountryCode:     "US",
			},
			wantErr: true,
		},
		{
			name: "Empty agent type ID",
			agent: &Agent{
				Name:            "test-agent",
				State:           AgentConnected,
				LastStateUpdate: validTime,
				AgentTypeID:     uuid.Nil,
				ProviderID:      validID,
				CountryCode:     "US",
			},
			wantErr: true,
		},
		{
			name: "Empty participant ID",
			agent: &Agent{
				Name:            "test-agent",
				State:           AgentConnected,
				LastStateUpdate: validTime,
				AgentTypeID:     validID,
				ProviderID:      uuid.Nil,
				CountryCode:     "US",
			},
			wantErr: true,
		},
		{
			name: "Invalid country code",
			agent: &Agent{
				Name:            "test-agent",
				State:           AgentConnected,
				LastStateUpdate: validTime,
				AgentTypeID:     validID,
				ProviderID:      validID,
				CountryCode:     "INVALID",
			},
			wantErr: true,
		},
		{
			name: "Valid with attributes",
			agent: &Agent{
				Name:            "test-agent",
				State:           AgentConnected,
				LastStateUpdate: validTime,
				AgentTypeID:     validID,
				ProviderID:      validID,
				CountryCode:     "US",
				Attributes:      Attributes{"key": []string{"value"}},
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
	validCountryCode := CountryCode("US")
	validAttributes := Attributes{"key": []string{"value"}}

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
					assert.Equal(t, validCountryCode, agent.CountryCode)
					assert.Equal(t, validAttributes, agent.Attributes)
					assert.Equal(t, validID, agent.ProviderID)
					assert.Equal(t, validID, agent.AgentTypeID)
					assert.Equal(t, AgentDisconnected, agent.State)
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
					return InvalidInputError{Err: errors.New("invalid country code")}
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
			agent, err := commander.Create(ctx, validName, validCountryCode, validAttributes, validID, validID)

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
				assert.Equal(t, validCountryCode, agent.CountryCode)
				assert.Equal(t, validAttributes, agent.Attributes)
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
	existingCountryCode := CountryCode("US")
	newCountryCode := CountryCode("CA")
	existingState := AgentDisconnected
	newState := AgentConnected

	existingAgent := &Agent{
		BaseEntity: BaseEntity{
			ID: agentID,
		},
		Name:            existingName,
		CountryCode:     existingCountryCode,
		State:           existingState,
		LastStateUpdate: time.Now(),
		ProviderID:      providerID,
		AgentTypeID:     agentTypeID,
	}

	tests := []struct {
		testName string
		// Create a deep copy of existingAgent for each test
		setupAgentCopy func() *Agent
		setupMocks     func(store *MockStore, audit *MockAuditEntryCommander)
		nameParam      *string
		countryCode    *CountryCode
		attributes     *Attributes
		stateParam     *AgentState
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
			countryCode: nil,
			attributes:  nil,
			stateParam:  nil,
			wantErr:     false,
		},
		{
			testName: "Update country code",
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
			countryCode: &newCountryCode,
			attributes:  nil,
			stateParam:  nil,
			wantErr:     false,
		},
		{
			testName: "Update state",
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
			countryCode: nil,
			attributes:  nil,
			stateParam:  &newState,
			wantErr:     false,
		},
		{
			testName: "Agent not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return nil, NewNotFoundErrorf("agent not found")
				}
				store.WithAgentRepo(agentRepo)
			},
			nameParam:   &newName,
			countryCode: nil,
			attributes:  nil,
			stateParam:  nil,
			wantErr:     true,
		},
		{
			testName: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return existingAgent, nil
				}

				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					// Simulate validation failure
					return InvalidInputError{Err: errors.New("invalid state")}
				}

				store.WithAgentRepo(agentRepo)
			},
			nameParam:   nil,
			countryCode: nil,
			attributes:  nil,
			stateParam:  &existingState,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewAgentCommander(store, audit)
			agent, err := commander.Update(ctx, agentID, tt.nameParam, tt.countryCode, tt.attributes, tt.stateParam)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)

				// Verify updates
				if tt.nameParam != nil {
					assert.Equal(t, *tt.nameParam, agent.Name)
				} else {
					assert.Equal(t, existingName, agent.Name)
				}

				if tt.countryCode != nil {
					assert.Equal(t, *tt.countryCode, agent.CountryCode)
				} else {
					assert.Equal(t, existingCountryCode, agent.CountryCode)
				}

				if tt.stateParam != nil {
					assert.Equal(t, *tt.stateParam, agent.State)
				} else {
					assert.Equal(t, existingState, agent.State)
				}
			}
		})
	}
}

func TestAgentCommander_Delete(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	providerID := uuid.New()

	existingAgent := &Agent{
		BaseEntity: BaseEntity{
			ID: agentID,
		},
		Name:            "test-agent",
		ProviderID:      providerID,
		State:           AgentDisconnected,
		LastStateUpdate: time.Now(),
	}

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					assert.Equal(t, agentID, id)
					return existingAgent, nil
				}

				serviceRepo := &MockServiceRepository{}
				serviceRepo.countByAgentFunc = func(ctx context.Context, agentID UUID) (int64, error) {
					return 0, nil // No services associated
				}

				tokenRepo := &MockTokenRepository{}
				tokenRepo.deleteByAgentIDFunc = func(ctx context.Context, agentID UUID) error {
					return nil
				}

				store.WithAgentRepo(agentRepo)
				store.WithServiceRepo(serviceRepo)
				store.WithTokenRepo(tokenRepo)
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
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Agent has services",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return existingAgent, nil
				}

				serviceRepo := &MockServiceRepository{}
				serviceRepo.countByAgentFunc = func(ctx context.Context, agentID UUID) (int64, error) {
					return 5, nil // Has 5 services associated
				}

				store.WithAgentRepo(agentRepo)
				store.WithServiceRepo(serviceRepo)
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "cannot delete agent with associated services")
			},
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
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgentCommander_UpdateState(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	providerID := uuid.New()
	agentTypeID := uuid.New()

	// Set up a more complete agent with required fields for UpdateState to succeed
	currentTime := time.Now().Add(-time.Hour)
	countryCode := CountryCode("US")
	agentName := "test-agent"

	existingState := AgentDisconnected
	newState := AgentConnected

	existingAgent := &Agent{
		BaseEntity: BaseEntity{
			ID:        agentID,
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
		},
		Name:        agentName,
		CountryCode: countryCode,
		ProviderID:  providerID,
		AgentTypeID: agentTypeID,
	}

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		state      AgentState
		wantErr    bool
	}{
		{
			name: "Update state success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}

				// Important: Make a complete copy of existingAgent that has all required fields
				agentCopy := *existingAgent
				agentCopy.State = existingState
				agentCopy.LastStateUpdate = time.Now().Add(-time.Hour)

				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return &agentCopy, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}

				agentRepo.updateFunc = func(ctx context.Context, agent *Agent) error {
					assert.Equal(t, newState, agent.State)
					// Verify last state update was updated
					assert.True(t, agent.LastStateUpdate.After(existingAgent.LastStateUpdate))
					return nil
				}

				store.WithAgentRepo(agentRepo)
			},
			state:   newState,
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
			state:   newState,
			wantErr: true,
		},
		{
			name: "Invalid state",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return existingAgent, nil
				}

				// The atomic function will simulate the validation failure
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return InvalidInputError{Err: errors.New("invalid state")}
				}

				store.WithAgentRepo(agentRepo)
			},
			state:   AgentState("InvalidState"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewAgentCommander(store, audit)
			agent, err := commander.UpdateState(ctx, agentID, tt.state)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tt.state, agent.State)
			}
		})
	}
}
