package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
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

func TestApplyAgentPropertyUpdates(t *testing.T) {
	ctx := context.Background()

	// Create a simple schema for testing
	testSchema := &schema.Schema{
		Properties: map[string]schema.PropertyDefinition{
			"ipAddress": {
				Type: "string",
				Validators: []schema.ValidatorConfig{
					{
						Type:   "source",
						Config: map[string]any{"source": "agent"},
					},
				},
			},
			"port": {
				Type: "integer",
				Validators: []schema.ValidatorConfig{
					{
						Type:   "source",
						Config: map[string]any{"source": "agent"},
					},
				},
			},
			"hostname": {
				Type: "string",
				// No source validator means default = user input
			},
		},
	}

	serviceType := &ServiceType{
		BaseEntity:     BaseEntity{ID: uuid.New()},
		Name:           "test-service",
		PropertySchema: testSchema,
	}

	tests := []struct {
		name          string
		service       *Service
		updates       map[string]any
		expectError   bool
		expectedProps map[string]any
		errorContains string
	}{
		{
			name: "Agent can update agent-source properties",
			service: &Service{
				BaseEntity: BaseEntity{ID: uuid.New()},
				Status:     "Running",
				Properties: &properties.JSON{
					"hostname": "test-host",
				},
			},
			updates: map[string]any{
				"ipAddress": "192.168.1.100",
				"port":      8080,
			},
			expectError: false,
			expectedProps: map[string]any{
				"hostname":  "test-host",
				"ipAddress": "192.168.1.100",
				"port":      8080, // Integers are preserved as int
			},
		},
		{
			name: "Empty updates do nothing",
			service: &Service{
				BaseEntity: BaseEntity{ID: uuid.New()},
				Status:     "Running",
				Properties: &properties.JSON{
					"hostname": "test-host",
				},
			},
			updates:     map[string]any{},
			expectError: false,
			expectedProps: map[string]any{
				"hostname": "test-host",
			},
		},
		{
			name: "Service with nil properties",
			service: &Service{
				BaseEntity: BaseEntity{ID: uuid.New()},
				Status:     "New",
				Properties: nil,
			},
			updates: map[string]any{
				"ipAddress": "192.168.1.100",
			},
			expectError: false,
			expectedProps: map[string]any{
				"ipAddress": "192.168.1.100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create engine with validators
			mockStore := NewMockStore(t)
			engine := NewServicePropertyEngine(mockStore, nil)

			// Apply updates
			err := ApplyAgentPropertyUpdates(ctx, engine, tt.service, serviceType, tt.updates)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.expectedProps != nil {
					assert.NotNil(t, tt.service.Properties)
					for k, v := range tt.expectedProps {
						assert.Equal(t, v, (*tt.service.Properties)[k], "Property %s mismatch", k)
					}
				}
			}
		})
	}
}
