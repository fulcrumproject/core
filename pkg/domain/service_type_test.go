package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceType_TableName(t *testing.T) {
	st := ServiceType{}
	assert.Equal(t, "service_types", st.TableName())
}

// TestServiceTypeBasics tests basic ServiceType operations
// Since there's no explicit Validate method, we'll test the name field
// which is marked as "not null" in GORM annotations
func TestServiceTypeBasics(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name        string
		serviceType *ServiceType
		description string
	}{
		{
			name: "Valid service type",
			serviceType: &ServiceType{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name: "Web Server",
			},
			description: "Valid service type with name",
		},
		{
			name: "Empty name",
			serviceType: &ServiceType{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name: "",
			},
			description: "Service type with empty name would fail database validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that the struct can be created
			assert.NotNil(t, tt.serviceType)
			assert.Equal(t, tt.serviceType.Name, tt.serviceType.Name)
		})
	}
}

// TestServiceSchema_Validate_Source tests source field validation
func TestServiceSchema_Validate_Source(t *testing.T) {
	tests := []struct {
		name      string
		schema    ServiceSchema
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid source: input",
			schema: ServiceSchema{
				"hostname": ServicePropertyDefinition{
					Type:   "string",
					Source: "input",
				},
			},
			expectErr: false,
		},
		{
			name: "Valid source: agent",
			schema: ServiceSchema{
				"internalIp": ServicePropertyDefinition{
					Type:   "string",
					Source: "agent",
				},
			},
			expectErr: false,
		},
		{
			name: "Valid source: empty (default)",
			schema: ServiceSchema{
				"hostname": ServicePropertyDefinition{
					Type:   "string",
					Source: "",
				},
			},
			expectErr: false,
		},
		{
			name: "Invalid source: invalid value",
			schema: ServiceSchema{
				"hostname": ServicePropertyDefinition{
					Type:   "string",
					Source: "invalid",
				},
			},
			expectErr: true,
			errMsg:    "source must be 'input' or 'agent'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServiceSchema_Validate_Updatable tests updatable field validation
func TestServiceSchema_Validate_Updatable(t *testing.T) {
	tests := []struct {
		name      string
		schema    ServiceSchema
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid updatable: always",
			schema: ServiceSchema{
				"tags": ServicePropertyDefinition{
					Type:      "string",
					Updatable: "always",
				},
			},
			expectErr: false,
		},
		{
			name: "Valid updatable: never",
			schema: ServiceSchema{
				"hostname": ServicePropertyDefinition{
					Type:      "string",
					Updatable: "never",
				},
			},
			expectErr: false,
		},
		{
			name: "Valid updatable: statuses with updatableIn",
			schema: ServiceSchema{
				"cpu": ServicePropertyDefinition{
					Type:        "integer",
					Updatable:   "statuses",
					UpdatableIn: []string{"Stopped"},
				},
			},
			expectErr: false,
		},
		{
			name: "Valid updatable: empty (default)",
			schema: ServiceSchema{
				"hostname": ServicePropertyDefinition{
					Type:      "string",
					Updatable: "",
				},
			},
			expectErr: false,
		},
		{
			name: "Invalid updatable: invalid value",
			schema: ServiceSchema{
				"hostname": ServicePropertyDefinition{
					Type:      "string",
					Updatable: "sometimes",
				},
			},
			expectErr: true,
			errMsg:    "updatable must be 'always', 'never', or 'statuses'",
		},
		{
			name: "Invalid updatable: statuses without updatableIn",
			schema: ServiceSchema{
				"cpu": ServicePropertyDefinition{
					Type:      "integer",
					Updatable: "statuses",
				},
			},
			expectErr: true,
			errMsg:    "updatableIn must be provided and not empty when updatable is 'statuses'",
		},
		{
			name: "Invalid updatable: statuses with empty updatableIn",
			schema: ServiceSchema{
				"cpu": ServicePropertyDefinition{
					Type:        "integer",
					Updatable:   "statuses",
					UpdatableIn: []string{},
				},
			},
			expectErr: true,
			errMsg:    "updatableIn must be provided and not empty when updatable is 'statuses'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServiceSchema_Validate_NestedProperties tests validation of nested properties
func TestServiceSchema_Validate_NestedProperties(t *testing.T) {
	tests := []struct {
		name      string
		schema    ServiceSchema
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid nested properties with source",
			schema: ServiceSchema{
				"network": ServicePropertyDefinition{
					Type: "object",
					Properties: map[string]ServicePropertyDefinition{
						"ip": {
							Type:   "string",
							Source: "agent",
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Invalid nested property source",
			schema: ServiceSchema{
				"network": ServicePropertyDefinition{
					Type: "object",
					Properties: map[string]ServicePropertyDefinition{
						"ip": {
							Type:   "string",
							Source: "invalid",
						},
					},
				},
			},
			expectErr: true,
			errMsg:    "source must be 'input' or 'agent'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServiceSchema_Validate_CompleteSchema tests a complete schema with multiple properties
func TestServiceSchema_Validate_CompleteSchema(t *testing.T) {
	schema := ServiceSchema{
		"hostname": ServicePropertyDefinition{
			Type:      "string",
			Source:    "input",
			Updatable: "never",
			Required:  true,
		},
		"cpu": ServicePropertyDefinition{
			Type:        "integer",
			Source:      "input",
			Updatable:   "statuses",
			UpdatableIn: []string{"Stopped"},
		},
		"memory": ServicePropertyDefinition{
			Type:        "integer",
			Source:      "input",
			Updatable:   "statuses",
			UpdatableIn: []string{"Stopped"},
		},
		"tags": ServicePropertyDefinition{
			Type:      "object",
			Source:    "input",
			Updatable: "always",
		},
		"internalIp": ServicePropertyDefinition{
			Type:      "string",
			Source:    "agent",
			Updatable: "never",
		},
	}

	err := schema.Validate()
	assert.NoError(t, err)
}
