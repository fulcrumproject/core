package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
				Status:        "New",
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
				Status:        "New",
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
			name: "Empty status",
			service: &Service{
				Name:          "Web Server",
				Status:        "",
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service status cannot be empty",
		},
		{
			name: "Nil group ID",
			service: &Service{
				Name:          "Web Server",
				Status:        "New",
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
				Status:        "New",
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
				Status:        "New",
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
				Status:        "New",
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
				Name:            "Web Server",
				Status:          "New",
				GroupID:         validID,
				AgentID:         validID,
				ServiceTypeID:   validID,
				ProviderID:      validID,
				ConsumerID:      validID,
				AgentInstanceID: helpers.StringPtr("ext-123"),
			},
			wantErr: false,
		},
		{
			name: "With error message",
			service: &Service{
				Name:          "Web Server",
				Status:        "New",
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
				Status:        "New",
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

// Property merging tests removed - merging is now handled by the schema engine
// The engine's ApplyUpdate method handles merging old and new properties

// TestApplyAgentPropertyUpdates has been removed
// Agent property validation is now handled by the schema engine
// TODO: Reimplement tests when agent property validation is integrated with schema engine
