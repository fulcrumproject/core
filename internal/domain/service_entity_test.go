package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceState_Validate(t *testing.T) {
	tests := []struct {
		name       string
		state      ServiceState
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid Creating state",
			state:   ServiceCreating,
			wantErr: false,
		},
		{
			name:    "Valid Created state",
			state:   ServiceCreated,
			wantErr: false,
		},
		{
			name:    "Valid Starting state",
			state:   ServiceStarting,
			wantErr: false,
		},
		{
			name:    "Valid Started state",
			state:   ServiceStarted,
			wantErr: false,
		},
		{
			name:    "Valid Stopping state",
			state:   ServiceStopping,
			wantErr: false,
		},
		{
			name:    "Valid Stopped state",
			state:   ServiceStopped,
			wantErr: false,
		},
		{
			name:    "Valid HotUpdating state",
			state:   ServiceHotUpdating,
			wantErr: false,
		},
		{
			name:    "Valid ColdUpdating state",
			state:   ServiceColdUpdating,
			wantErr: false,
		},
		{
			name:    "Valid Deleting state",
			state:   ServiceDeleting,
			wantErr: false,
		},
		{
			name:    "Valid Deleted state",
			state:   ServiceDeleted,
			wantErr: false,
		},
		{
			name:       "Invalid state",
			state:      "InvalidState",
			wantErr:    true,
			errMessage: "invalid service state",
		},
		{
			name:       "Empty state",
			state:      "",
			wantErr:    true,
			errMessage: "invalid service state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseServiceState(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       ServiceState
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid Creating state",
			input:   "Creating",
			want:    ServiceCreating,
			wantErr: false,
		},
		{
			name:    "Valid Created state",
			input:   "Created",
			want:    ServiceCreated,
			wantErr: false,
		},
		{
			name:    "Valid Started state",
			input:   "Started",
			want:    ServiceStarted,
			wantErr: false,
		},
		{
			name:       "Invalid state",
			input:      "InvalidState",
			wantErr:    true,
			errMessage: "invalid service state",
		},
		{
			name:       "Empty state",
			input:      "",
			wantErr:    true,
			errMessage: "invalid service state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := ParseServiceState(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Equal(t, ServiceState(""), state)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, state)
			}
		})
	}
}

func TestService_TableName(t *testing.T) {
	svc := Service{}
	assert.Equal(t, "services", svc.TableName())
}

func TestService_Validate(t *testing.T) {
	validID := uuid.New()
	createdState := ServiceCreated

	tests := []struct {
		name       string
		service    *Service
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid service",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			service: &Service{
				Name:          "",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service name cannot be empty",
		},
		{
			name: "Invalid current state",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  "InvalidState",
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "invalid service state",
		},
		{
			name: "Invalid target state",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				TargetState:   (*ServiceState)(stringPtr("InvalidState")),
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "invalid service state",
		},
		{
			name: "Valid target state",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				TargetState:   &createdState,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr: false,
		},
		{
			name: "Nil group ID",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       uuid.Nil,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service group ID cannot be nil",
		},
		{
			name: "Nil agent ID",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       uuid.Nil,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service agent ID cannot be nil",
		},
		{
			name: "Nil service type ID",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: uuid.Nil,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service type ID cannot be nil",
		},
		{
			name: "With valid attributes",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				Attributes:    Attributes{"tier": {"premium"}},
			},
			wantErr: false,
		},
		{
			name: "With invalid attributes",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				Attributes:    Attributes{"tier": {""}}, // Empty value
			},
			wantErr:    true,
			errMessage: "has an empty value",
		},
		{
			name: "With properties",
			service: &Service{
				Name:              "Web Server",
				CurrentState:      ServiceCreated,
				GroupID:           validID,
				AgentID:           validID,
				ServiceTypeID:     validID,
				ProviderID:        validID,
				ConsumerID:        validID,
				CurrentProperties: &JSON{"port": 8080},
				TargetProperties:  &JSON{"port": 8888},
			},
			wantErr: false,
		},
		{
			name: "With external ID",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				ExternalID:    stringPtr("srv-123"),
			},
			wantErr: false,
		},
		{
			name: "With resources",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				Resources:     &JSON{"containerId": "c123"},
			},
			wantErr: false,
		},
		{
			name: "With error message",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				ErrorMessage:  stringPtr("Failed to start"),
			},
			wantErr: false,
		},
		{
			name: "With failed action",
			service: &Service{
				Name:          "Web Server",
				CurrentState:  ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				FailedAction:  svcActionPtr(ServiceActionStart),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.service.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceNextStateAndAction(t *testing.T) {
	tests := []struct {
		name         string
		currentState ServiceState
		targetState  ServiceState
		wantState    ServiceState
		wantAction   ServiceAction
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "Created to Started",
			currentState: ServiceCreated,
			targetState:  ServiceStarted,
			wantState:    ServiceStarting,
			wantAction:   ServiceActionStart,
			wantErr:      false,
		},
		{
			name:         "Started to Stopped",
			currentState: ServiceStarted,
			targetState:  ServiceStopped,
			wantState:    ServiceStopping,
			wantAction:   ServiceActionStop,
			wantErr:      false,
		},
		{
			name:         "Stopped to Started",
			currentState: ServiceStopped,
			targetState:  ServiceStarted,
			wantState:    ServiceStarting,
			wantAction:   ServiceActionStart,
			wantErr:      false,
		},
		{
			name:         "Stopped to Deleted",
			currentState: ServiceStopped,
			targetState:  ServiceDeleted,
			wantState:    ServiceDeleting,
			wantAction:   ServiceActionDelete,
			wantErr:      false,
		},
		{
			name:         "Invalid transition",
			currentState: ServiceStarting,
			targetState:  ServiceStopped,
			wantErr:      true,
			errMessage:   "invalid transition",
		},
		{
			name:         "Invalid - Started to Deleted",
			currentState: ServiceStarted,
			targetState:  ServiceDeleted,
			wantErr:      true,
			errMessage:   "invalid transition",
		},
		{
			name:         "Invalid - Creating to Stopped",
			currentState: ServiceCreating,
			targetState:  ServiceStopped,
			wantErr:      true,
			errMessage:   "invalid transition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, action, err := serviceNextStateAndAction(tt.currentState, tt.targetState)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantState, state)
				assert.Equal(t, tt.wantAction, action)
			}
		})
	}
}

func TestServiceUpdateNextStateAndAction(t *testing.T) {
	tests := []struct {
		name         string
		currentState ServiceState
		wantState    ServiceState
		wantAction   ServiceAction
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "Stopped to ColdUpdating",
			currentState: ServiceStopped,
			wantState:    ServiceColdUpdating,
			wantAction:   ServiceActionColdUpdate,
			wantErr:      false,
		},
		{
			name:         "Started to HotUpdating",
			currentState: ServiceStarted,
			wantState:    ServiceHotUpdating,
			wantAction:   ServiceActionHotUpdate,
			wantErr:      false,
		},
		{
			name:         "Invalid - Creating",
			currentState: ServiceCreating,
			wantErr:      true,
			errMessage:   "cannot update attributes",
		},
		{
			name:         "Invalid - Deleting",
			currentState: ServiceDeleting,
			wantErr:      true,
			errMessage:   "cannot update attributes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, action, err := serviceUpdateNextStateAndAction(tt.currentState)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantState, state)
				assert.Equal(t, tt.wantAction, action)
			}
		})
	}
}
