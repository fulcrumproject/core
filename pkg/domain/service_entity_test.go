package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceStatus_Validate(t *testing.T) {
	tests := []struct {
		name       string
		status     ServiceStatus
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid Creating status",
			status:  ServiceCreating,
			wantErr: false,
		},
		{
			name:    "Valid Created status",
			status:  ServiceCreated,
			wantErr: false,
		},
		{
			name:    "Valid Starting status",
			status:  ServiceStarting,
			wantErr: false,
		},
		{
			name:    "Valid Started status",
			status:  ServiceStarted,
			wantErr: false,
		},
		{
			name:    "Valid Stopping status",
			status:  ServiceStopping,
			wantErr: false,
		},
		{
			name:    "Valid Stopped status",
			status:  ServiceStopped,
			wantErr: false,
		},
		{
			name:    "Valid HotUpdating status",
			status:  ServiceHotUpdating,
			wantErr: false,
		},
		{
			name:    "Valid ColdUpdating status",
			status:  ServiceColdUpdating,
			wantErr: false,
		},
		{
			name:    "Valid Deleting status",
			status:  ServiceDeleting,
			wantErr: false,
		},
		{
			name:    "Valid Deleted status",
			status:  ServiceDeleted,
			wantErr: false,
		},
		{
			name:       "Invalid status",
			status:     "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid service status",
		},
		{
			name:       "Empty status",
			status:     "",
			wantErr:    true,
			errMessage: "invalid service status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
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

func TestParseServiceStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       ServiceStatus
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid Creating status",
			input:   "Creating",
			want:    ServiceCreating,
			wantErr: false,
		},
		{
			name:    "Valid Created status",
			input:   "Created",
			want:    ServiceCreated,
			wantErr: false,
		},
		{
			name:    "Valid Started status",
			input:   "Started",
			want:    ServiceStarted,
			wantErr: false,
		},
		{
			name:       "Invalid status",
			input:      "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid service status",
		},
		{
			name:       "Empty status",
			input:      "",
			wantErr:    true,
			errMessage: "invalid service status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParseServiceStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Equal(t, ServiceStatus(""), status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, status)
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
	createdStatus := ServiceCreated

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
				CurrentStatus: ServiceCreated,
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
				CurrentStatus: ServiceCreated,
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
			name: "Invalid current status",
			service: &Service{
				Name:          "Web Server",
				CurrentStatus: "InvalidStatus",
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "invalid service status",
		},
		{
			name: "Invalid target status",
			service: &Service{
				Name:          "Web Server",
				CurrentStatus: ServiceCreated,
				TargetStatus:  (*ServiceStatus)(stringPtr("InvalidStatus")),
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "invalid service status",
		},
		{
			name: "Valid target status",
			service: &Service{
				Name:          "Web Server",
				CurrentStatus: ServiceCreated,
				TargetStatus:  &createdStatus,
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
				CurrentStatus: ServiceCreated,
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
				CurrentStatus: ServiceCreated,
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
				CurrentStatus: ServiceCreated,
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
			name: "With properties",
			service: &Service{
				Name:              "Web Server",
				CurrentStatus:     ServiceCreated,
				GroupID:           validID,
				AgentID:           validID,
				ServiceTypeID:     validID,
				ProviderID:        validID,
				ConsumerID:        validID,
				CurrentProperties: &properties.JSON{"port": 8080},
				TargetProperties:  &properties.JSON{"port": 8888},
			},
			wantErr: false,
		},
		{
			name: "With external ID",
			service: &Service{
				Name:          "Web Server",
				CurrentStatus: ServiceCreated,
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
				CurrentStatus: ServiceCreated,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				Resources:     &properties.JSON{"containerId": "c123"},
			},
			wantErr: false,
		},
		{
			name: "With error message",
			service: &Service{
				Name:          "Web Server",
				CurrentStatus: ServiceCreated,
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
				CurrentStatus: ServiceCreated,
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

func svcActionPtr(svcAction ServiceAction) *ServiceAction {
	return &svcAction
}

func TestServiceNextStatusAndAction(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus ServiceStatus
		targetStatus  ServiceStatus
		wantStatus    ServiceStatus
		wantAction    ServiceAction
		wantErr       bool
		errMessage    string
	}{
		{
			name:          "Created to Started",
			currentStatus: ServiceCreated,
			targetStatus:  ServiceStarted,
			wantStatus:    ServiceStarting,
			wantAction:    ServiceActionStart,
			wantErr:       false,
		},
		{
			name:          "Started to Stopped",
			currentStatus: ServiceStarted,
			targetStatus:  ServiceStopped,
			wantStatus:    ServiceStopping,
			wantAction:    ServiceActionStop,
			wantErr:       false,
		},
		{
			name:          "Stopped to Started",
			currentStatus: ServiceStopped,
			targetStatus:  ServiceStarted,
			wantStatus:    ServiceStarting,
			wantAction:    ServiceActionStart,
			wantErr:       false,
		},
		{
			name:          "Stopped to Deleted",
			currentStatus: ServiceStopped,
			targetStatus:  ServiceDeleted,
			wantStatus:    ServiceDeleting,
			wantAction:    ServiceActionDelete,
			wantErr:       false,
		},
		{
			name:          "Invalid transition",
			currentStatus: ServiceStarting,
			targetStatus:  ServiceStopped,
			wantErr:       true,
			errMessage:    "invalid transition",
		},
		{
			name:          "Invalid - Started to Deleted",
			currentStatus: ServiceStarted,
			targetStatus:  ServiceDeleted,
			wantErr:       true,
			errMessage:    "invalid transition",
		},
		{
			name:          "Invalid - Creating to Stopped",
			currentStatus: ServiceCreating,
			targetStatus:  ServiceStopped,
			wantErr:       true,
			errMessage:    "invalid transition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, action, err := serviceNextStatusAndAction(tt.currentStatus, tt.targetStatus)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, status)
				assert.Equal(t, tt.wantAction, action)
			}
		})
	}
}

func TestServiceUpdateNextStatusAndAction(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus ServiceStatus
		wantStatus    ServiceStatus
		wantAction    ServiceAction
		wantErr       bool
		errMessage    string
	}{
		{
			name:          "Stopped to ColdUpdating",
			currentStatus: ServiceStopped,
			wantStatus:    ServiceColdUpdating,
			wantAction:    ServiceActionColdUpdate,
			wantErr:       false,
		},
		{
			name:          "Started to HotUpdating",
			currentStatus: ServiceStarted,
			wantStatus:    ServiceHotUpdating,
			wantAction:    ServiceActionHotUpdate,
			wantErr:       false,
		},
		{
			name:          "Invalid - Creating",
			currentStatus: ServiceCreating,
			wantErr:       true,
			errMessage:    "cannot update properties",
		},
		{
			name:          "Invalid - Deleting",
			currentStatus: ServiceDeleting,
			wantErr:       true,
			errMessage:    "cannot update properties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, action, err := serviceUpdateNextStatusAndAction(tt.currentStatus)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, status)
				assert.Equal(t, tt.wantAction, action)
			}
		})
	}
}
