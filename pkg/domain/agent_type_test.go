package domain

import (
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

func TestAgentType_WithConfigurationSchema(t *testing.T) {
	engine := NewAgentConfigSchemaEngine(nil)

	tests := []struct {
		name    string
		schema  schema.Schema
		wantErr bool
	}{
		{
			name: "valid simple schema",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"apiKey": {
						Type:     "string",
						Label:    "API Key",
						Required: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty schema is invalid",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{},
			},
			wantErr: true,
		},
		{
			name: "invalid type in schema",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"field": {
						Type: "invalid_type",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "schema with validators",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"region": {
						Type: "string",
						Validators: []schema.ValidatorConfig{
							{
								Type:   "enum",
								Config: map[string]any{"values": []any{"us-east-1", "us-west-2"}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "schema with secret",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"password": {
						Type:   "string",
						Secret: &schema.SecretConfig{Type: "persistent"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "schema with nested object",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"credentials": {
						Type: "object",
						Properties: map[string]schema.PropertyDefinition{
							"username": {
								Type:     "string",
								Required: true,
							},
							"password": {
								Type:     "string",
								Required: true,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "schema with schema validator",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"password": {
						Type: "string",
					},
					"sshKey": {
						Type: "string",
					},
				},
				Validators: []schema.SchemaValidatorConfig{
					{
						Type:   "exactlyOne",
						Config: map[string]any{"properties": []any{"password", "sshKey"}},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentType := &AgentType{
				Name: "Test Agent",
				TemplateValidation: TemplateValidation{
					ConfigurationSchema: tt.schema,
				},
			}

			err := agentType.ValidateWithEngine(engine)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentType_ValidateTemplates(t *testing.T) {
	baseProps := map[string]schema.PropertyDefinition{
		"host": {Type: "string"},
		"port": {Type: "integer"},
	}
	arrayProps := map[string]schema.PropertyDefinition{
		"servers": {Type: "array"},
	}
	nestedProps := map[string]schema.PropertyDefinition{
		"proxmoxAPI": {
			Type: "object",
			Properties: map[string]schema.PropertyDefinition{
				"apiToken": {Type: "string"},
				"apiUrls": {
					Type:  "array",
					Items: &schema.PropertyDefinition{Type: "string"},
				},
			},
		},
	}
	nestedMissingProps := map[string]schema.PropertyDefinition{
		"proxmoxAPI": {
			Type:       "object",
			Properties: map[string]schema.PropertyDefinition{},
		},
	}

	tests := []struct {
		name              string
		props             map[string]schema.PropertyDefinition
		configTemplate    string
		cmdTemplate       string
		configContentType string
		wantErr           bool
		wantMsgContains   string
	}{
		{
			name:              "accept valid refs",
			props:             baseProps,
			configTemplate:    "host={{.host}}\nport={{.port}}",
			cmdTemplate:       "run --p {{.port}} --tok {{.authToken}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           false,
		},
		{
			name:              "accept empty templates",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "",
			configContentType: "text/plain",
			wantErr:           false,
		},
		{
			name:              "reject unknown ref in configTemplate",
			props:             baseProps,
			configTemplate:    "{{.missing}}",
			cmdTemplate:       "",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "missing",
		},
		{
			name:              "reject unknown ref in cmdTemplate",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "{{.unknown}}",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "unknown",
		},
		{
			name:              "reject configUrl in configTemplate",
			props:             baseProps,
			configTemplate:    "url={{.configUrl}}",
			cmdTemplate:       "",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "configUrl",
		},
		{
			name:              "reject configUrl in cmdTemplate only (coupling)",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "{{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "both be set or both be empty",
		},
		{
			name:              "reject unparseable template",
			props:             baseProps,
			configTemplate:    "{{.host",
			cmdTemplate:       "",
			configContentType: "text/plain",
			wantErr:           true,
		},
		{
			name:              "accept range over declared array prop",
			props:             arrayProps,
			configTemplate:    "{{range .servers}}x{{end}}",
			cmdTemplate:       "install --tok {{.authToken}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           false,
		},
		{
			name:              "accept nested object ref",
			props:             nestedProps,
			configTemplate:    "token={{.proxmoxAPI.apiToken}}",
			cmdTemplate:       "install --tok {{.authToken}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           false,
		},
		{
			name:              "reject nested ref to missing nested prop",
			props:             nestedMissingProps,
			configTemplate:    "token={{.proxmoxAPI.apiToken}}",
			cmdTemplate:       "install --tok {{.authToken}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "apiToken",
		},
		{
			name:              "accept range over nested array-of-string",
			props:             nestedProps,
			configTemplate:    "{{range .proxmoxAPI.apiUrls}}{{.}}\n{{end}}",
			cmdTemplate:       "install --tok {{.authToken}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           false,
		},
		{
			name:              "reject configTemplate set but cmdTemplate empty",
			props:             baseProps,
			configTemplate:    "host={{.host}}",
			cmdTemplate:       "",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "both be set or both be empty",
		},
		{
			name:              "reject cmdTemplate set but configTemplate empty",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "install --tok {{.authToken}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "both be set or both be empty",
		},
		{
			name:              "reject both set but cmd missing configUrl",
			props:             baseProps,
			configTemplate:    "host={{.host}}",
			cmdTemplate:       "run --h {{.host}} --tok {{.authToken}}",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "must reference {{.configUrl}}",
		},
		{
			name:              "reject both set but cmd missing authToken",
			props:             baseProps,
			configTemplate:    "host={{.host}}",
			cmdTemplate:       "run --h {{.host}} --url {{.configUrl}}",
			configContentType: "text/plain",
			wantErr:           true,
			wantMsgContains:   "must reference {{.authToken}}",
		},
		{
			name:              "accept configUrl with whitespace and trim markers",
			props:             baseProps,
			configTemplate:    "host={{.host}}",
			cmdTemplate:       "run --url {{- .configUrl -}} --tok {{ .authToken }} --host {{ .host }}",
			configContentType: "text/plain",
			wantErr:           false,
		},
		{
			name:              "reject malformed mime type",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "",
			configContentType: "not a mime",
			wantErr:           true,
			wantMsgContains:   "configContentType",
		},
		{
			name:              "accept common mime type",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "",
			configContentType: "application/yaml",
			wantErr:           false,
		},
		{
			name:              "accept mime type with params",
			props:             baseProps,
			configTemplate:    "",
			cmdTemplate:       "",
			configContentType: "text/plain; charset=utf-8",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := &AgentType{
				Name: "Test Agent",
				TemplateValidation: TemplateValidation{
					ConfigurationSchema: schema.Schema{Properties: tt.props},
					ConfigTemplate:      tt.configTemplate,
					CmdTemplate:         tt.cmdTemplate,
					ConfigContentType:   tt.configContentType,
				},
			}
			err := at.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantMsgContains != "" && !strings.Contains(err.Error(), tt.wantMsgContains) {
				t.Errorf("expected error to contain %q, got: %v", tt.wantMsgContains, err)
			}
		})
	}
}

func TestAgentType_ValidateWithEngine_EmptyName(t *testing.T) {
	engine := NewAgentConfigSchemaEngine(nil)

	agentType := &AgentType{
		Name: "",
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{},
			},
		},
	}

	err := agentType.ValidateWithEngine(engine)
	if err == nil {
		t.Error("Expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("Expected error about empty name, got: %v", err)
	}
}

func TestNewAgentType(t *testing.T) {
	t.Run("create with schema", func(t *testing.T) {
		params := CreateAgentTypeParams{
			Name: "AWS Agent",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"region": {
						Type:     "string",
						Required: true,
					},
				},
			},
		}

		agentType := NewAgentType(params)

		if agentType.Name != "AWS Agent" {
			t.Errorf("Expected name 'AWS Agent', got '%s'", agentType.Name)
		}

		if len(agentType.ConfigurationSchema.Properties) != 1 {
			t.Errorf("Expected 1 property in schema, got %d", len(agentType.ConfigurationSchema.Properties))
		}
	})

	t.Run("create with minimal schema", func(t *testing.T) {
		params := CreateAgentTypeParams{
			Name: "Simple Agent",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"enabled": {
						Type:    "boolean",
						Default: true,
					},
				},
			},
		}

		agentType := NewAgentType(params)

		if agentType == nil {
			t.Fatal("Expected agent type to be created")
		}

		if len(agentType.ConfigurationSchema.Properties) != 1 {
			t.Errorf("Expected 1 property in schema, got %d", len(agentType.ConfigurationSchema.Properties))
		}
	})

	t.Run("create with complex schema", func(t *testing.T) {
		params := CreateAgentTypeParams{
			Name: "Complex Agent",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"apiKey": {
						Type:     "string",
						Required: true,
						Validators: []schema.ValidatorConfig{
							{
								Type:   "minLength",
								Config: map[string]any{"value": 10},
							},
						},
					},
					"timeout": {
						Type:    "integer",
						Default: 30,
						Validators: []schema.ValidatorConfig{
							{
								Type:   "min",
								Config: map[string]any{"value": 1},
							},
							{
								Type:   "max",
								Config: map[string]any{"value": 300},
							},
						},
					},
					"credentials": {
						Type: "object",
						Properties: map[string]schema.PropertyDefinition{
							"username": {
								Type:     "string",
								Required: true,
							},
							"password": {
								Type:   "string",
								Secret: &schema.SecretConfig{Type: "persistent"},
							},
						},
					},
				},
			},
		}

		agentType := NewAgentType(params)

		if len(agentType.ConfigurationSchema.Properties) != 3 {
			t.Errorf("Expected 3 properties in schema, got %d", len(agentType.ConfigurationSchema.Properties))
		}
	})
}

func TestAgentType_InfrastructureTypes(t *testing.T) {
	id1 := properties.UUID(uuid.New())
	id2 := properties.UUID(uuid.New())

	t.Run("Validate rejects more than one infrastructure type", func(t *testing.T) {
		at := &AgentType{
			Name: "vpn-agent",
			InfrastructureTypes: []InfrastructureType{
				{BaseEntity: BaseEntity{ID: id1}},
				{BaseEntity: BaseEntity{ID: id2}},
			},
		}
		err := at.Validate()
		if err == nil {
			t.Fatal("expected error for len > 1")
		}
		if !strings.Contains(err.Error(), "at most one infrastructure type") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Validate accepts zero or one", func(t *testing.T) {
		cases := []struct {
			name string
			its  []InfrastructureType
		}{
			{"empty", nil},
			{"one", []InfrastructureType{{BaseEntity: BaseEntity{ID: id1}}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				at := &AgentType{Name: "vpn-agent", InfrastructureTypes: tc.its}
				if err := at.Validate(); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("RequiredInfrastructureType returns nil when empty", func(t *testing.T) {
		at := &AgentType{}
		if got := at.RequiredInfrastructureType(); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("RequiredInfrastructureType returns first when present", func(t *testing.T) {
		at := &AgentType{InfrastructureTypes: []InfrastructureType{{BaseEntity: BaseEntity{ID: id1}}}}
		got := at.RequiredInfrastructureType()
		if got == nil || got.ID != id1 {
			t.Errorf("expected ID %v, got %v", id1, got)
		}
	})

	t.Run("NewAgentType materialises InfrastructureTypeIds", func(t *testing.T) {
		at := NewAgentType(CreateAgentTypeParams{
			Name:                  "vpn-agent",
			InfrastructureTypeIds: []properties.UUID{id1},
		})
		if len(at.InfrastructureTypes) != 1 || at.InfrastructureTypes[0].ID != id1 {
			t.Errorf("expected single InfrastructureType with ID %v, got %v", id1, at.InfrastructureTypes)
		}
	})

	t.Run("Update replaces InfrastructureTypes when non-nil", func(t *testing.T) {
		at := &AgentType{InfrastructureTypes: []InfrastructureType{{BaseEntity: BaseEntity{ID: id1}}}}
		newIDs := []properties.UUID{id2}
		at.Update(UpdateAgentTypeParams{InfrastructureTypeIds: &newIDs})
		if len(at.InfrastructureTypes) != 1 || at.InfrastructureTypes[0].ID != id2 {
			t.Errorf("expected replacement to ID %v, got %v", id2, at.InfrastructureTypes)
		}

		emptyIDs := []properties.UUID{}
		at.Update(UpdateAgentTypeParams{InfrastructureTypeIds: &emptyIDs})
		if len(at.InfrastructureTypes) != 0 {
			t.Errorf("expected empty after replace, got %v", at.InfrastructureTypes)
		}
	})

	t.Run("Update leaves InfrastructureTypes alone when nil", func(t *testing.T) {
		at := &AgentType{InfrastructureTypes: []InfrastructureType{{BaseEntity: BaseEntity{ID: id1}}}}
		at.Update(UpdateAgentTypeParams{InfrastructureTypeIds: nil})
		if len(at.InfrastructureTypes) != 1 || at.InfrastructureTypes[0].ID != id1 {
			t.Errorf("expected unchanged, got %v", at.InfrastructureTypes)
		}
	})
}

func TestAgentType_Update(t *testing.T) {
	engine := NewAgentConfigSchemaEngine(nil)

	agentType := &AgentType{
		Name: "Initial Agent",
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"apiKey": {
						Type: "string",
					},
				},
			},
		},
	}

	t.Run("update schema", func(t *testing.T) {
		newSchema := schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"apiKey": {
					Type:     "string",
					Required: true,
				},
				"region": {
					Type: "string",
				},
			},
		}

		updateParams := UpdateAgentTypeParams{
			ConfigurationSchema: &newSchema,
		}

		agentType.Update(updateParams)

		if len(agentType.ConfigurationSchema.Properties) != 2 {
			t.Errorf("Expected 2 properties after update, got %d", len(agentType.ConfigurationSchema.Properties))
		}

		// Verify the update is valid
		if err := agentType.ValidateWithEngine(engine); err != nil {
			t.Errorf("Updated schema should be valid: %v", err)
		}
	})

	t.Run("update name only", func(t *testing.T) {
		newName := "Updated Agent Name"
		updateParams := UpdateAgentTypeParams{
			Name: &newName,
		}

		agentType.Update(updateParams)

		if agentType.Name != "Updated Agent Name" {
			t.Errorf("Expected name 'Updated Agent Name', got '%s'", agentType.Name)
		}
	})
}
