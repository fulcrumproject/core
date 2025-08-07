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
			name:    "Valid New status",
			status:  ServiceNew,
			wantErr: false,
		},
		{
			name:    "Valid Started status",
			status:  ServiceStarted,
			wantErr: false,
		},

		{
			name:    "Valid Stopped status",
			status:  ServiceStopped,
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
			name:    "Valid New status",
			input:   "New",
			want:    ServiceNew,
			wantErr: false,
		},
		{
			name:    "Valid Started status",
			input:   "Started",
			want:    ServiceStarted,
			wantErr: false,
		},
		{
			name:    "Valid Stopped status",
			input:   "Stopped",
			want:    ServiceStopped,
			wantErr: false,
		},
		{
			name:    "Valid Deleted status",
			input:   "Deleted",
			want:    ServiceDeleted,
			wantErr: false,
		},
		{
			name:       "Invalid status",
			input:      "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid service status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_TableName(t *testing.T) {
	svc := &Service{}
	assert.Equal(t, "services", svc.TableName())
}

func TestService_Validate(t *testing.T) {
	validID := uuid.New()

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
				Status:        ServiceNew,
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
				Status:        ServiceNew,
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
			name: "Invalid status",
			service: &Service{
				Name:          "Web Server",
				Status:        "InvalidStatus",
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
			name: "Nil group ID",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
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
				Status:        ServiceNew,
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
				Status:        ServiceNew,
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
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				Properties:    &properties.JSON{"port": 8080},
			},
			wantErr: false,
		},
		{
			name: "With external ID",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				ExternalID:    stringPtr("ext-123"),
			},
			wantErr: false,
		},
		{
			name: "With error message",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr: false,
		},
		{
			name: "With failed action",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
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
