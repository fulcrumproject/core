package domain

import (
	"context"
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestInfrastructureType_ValidateWithEngine_Schema(t *testing.T) {
	engine := NewInfrastructureConfigSchemaEngine(nil)

	tests := []struct {
		name    string
		schema  schema.Schema
		wantErr bool
	}{
		{
			name: "valid simple schema",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {
						Type:     "string",
						Label:    "Endpoint",
						Required: true,
					},
				},
			},
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
					"field": {Type: "invalid_type"},
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
							{Type: "enum", Config: map[string]any{"values": []any{"a", "b"}}},
						},
					},
				},
			},
		},
		{
			name: "schema with schema-level validator",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"password": {Type: "string"},
					"sshKey":   {Type: "string"},
				},
				Validators: []schema.SchemaValidatorConfig{
					{Type: "exactlyOne", Config: map[string]any{"properties": []any{"password", "sshKey"}}},
				},
			},
		},
		{
			name: "schema referencing unknown validator type",
			schema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"x": {
						Type: "string",
						Validators: []schema.ValidatorConfig{
							{Type: "doesNotExist", Config: map[string]any{}},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &InfrastructureType{
				Name: "Test Infra",
				TemplateValidation: TemplateValidation{
					ConfigurationSchema: tt.schema,
				},
			}
			err := it.ValidateWithEngine(engine)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInfrastructureType_ValidateWithEngine_EmptyName(t *testing.T) {
	engine := NewInfrastructureConfigSchemaEngine(nil)

	it := &InfrastructureType{
		Name: "",
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string"},
				},
			},
		},
	}

	err := it.ValidateWithEngine(engine)
	if err == nil {
		t.Fatal("Expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("Expected error about empty name, got: %v", err)
	}
}

func TestInfrastructureType_Validate_EmptyName(t *testing.T) {
	it := &InfrastructureType{
		Name: "",
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string"},
				},
			},
		},
	}

	err := it.Validate()
	if err == nil || !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("Expected empty-name error from Validate(), got: %v", err)
	}
}

func TestNewInfrastructureType(t *testing.T) {
	t.Run("defaults ConfigContentType to text/plain when omitted", func(t *testing.T) {
		params := CreateInfrastructureTypeParams{
			Name: "infra",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string", Required: true},
				},
			},
		}
		it := NewInfrastructureType(params)
		if it.ConfigContentType != "text/plain" {
			t.Errorf("Expected default 'text/plain', got %q", it.ConfigContentType)
		}
	})

	t.Run("preserves explicit ConfigContentType", func(t *testing.T) {
		params := CreateInfrastructureTypeParams{
			Name:              "infra",
			ConfigContentType: "text/yaml",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string"},
				},
			},
		}
		it := NewInfrastructureType(params)
		if it.ConfigContentType != "text/yaml" {
			t.Errorf("Expected 'text/yaml', got %q", it.ConfigContentType)
		}
	})

	t.Run("populates all fields", func(t *testing.T) {
		params := CreateInfrastructureTypeParams{
			Name:              "infra",
			ConfigTemplate:    "endpoint={{.endpoint}}\n",
			CmdTemplate:       "curl {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'",
			ConfigContentType: "text/plain",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string", Required: true},
				},
			},
		}
		it := NewInfrastructureType(params)
		if it.Name != "infra" {
			t.Errorf("Name mismatch: %q", it.Name)
		}
		if it.ConfigTemplate != params.ConfigTemplate {
			t.Errorf("ConfigTemplate mismatch")
		}
		if it.CmdTemplate != params.CmdTemplate {
			t.Errorf("CmdTemplate mismatch")
		}
		if len(it.ConfigurationSchema.Properties) != 1 {
			t.Errorf("Expected 1 schema property, got %d", len(it.ConfigurationSchema.Properties))
		}
	})
}

func TestInfrastructureType_Update(t *testing.T) {
	engine := NewInfrastructureConfigSchemaEngine(nil)

	makeIT := func() *InfrastructureType {
		return &InfrastructureType{
			Name: "Initial",
			TemplateValidation: TemplateValidation{
				ConfigurationSchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"endpoint": {Type: "string"},
					},
				},
				ConfigContentType: "text/plain",
			},
		}
	}

	t.Run("update name only", func(t *testing.T) {
		it := makeIT()
		newName := "Renamed"
		it.Update(UpdateInfrastructureTypeParams{Name: &newName})
		if it.Name != "Renamed" {
			t.Errorf("Expected 'Renamed', got %q", it.Name)
		}
		if err := it.ValidateWithEngine(engine); err != nil {
			t.Errorf("post-update validation failed: %v", err)
		}
	})

	t.Run("update schema", func(t *testing.T) {
		it := makeIT()
		newSchema := schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"endpoint": {Type: "string", Required: true},
				"region":   {Type: "string"},
			},
		}
		it.Update(UpdateInfrastructureTypeParams{ConfigurationSchema: &newSchema})
		if len(it.ConfigurationSchema.Properties) != 2 {
			t.Errorf("Expected 2 properties, got %d", len(it.ConfigurationSchema.Properties))
		}
	})

	t.Run("explicit empty ConfigContentType resets to text/plain", func(t *testing.T) {
		it := makeIT()
		it.ConfigContentType = "text/yaml"
		empty := ""
		it.Update(UpdateInfrastructureTypeParams{ConfigContentType: &empty})
		if it.ConfigContentType != "text/plain" {
			t.Errorf("Expected reset to 'text/plain', got %q", it.ConfigContentType)
		}
	})

	t.Run("nil pointers leave fields untouched", func(t *testing.T) {
		it := makeIT()
		before := *it
		it.Update(UpdateInfrastructureTypeParams{})
		if it.Name != before.Name || it.ConfigContentType != before.ConfigContentType {
			t.Errorf("Update with nil params mutated fields")
		}
	})
}

func TestInfrastructureTypeCommander_Delete(t *testing.T) {
	itID := properties.UUID(uuid.New())

	type stubs struct {
		infraCount int64
		atCount    int64
	}

	tests := []struct {
		name        string
		stubs       stubs
		wantErr     bool
		errContains string
	}{
		{name: "happy path", stubs: stubs{infraCount: 0, atCount: 0}, wantErr: false},
		{name: "blocked by dependent infrastructures", stubs: stubs{infraCount: 1, atCount: 0}, wantErr: true, errContains: "dependent infrastructure"},
		{name: "blocked by dependent agent types", stubs: stubs{infraCount: 0, atCount: 2}, wantErr: true, errContains: "dependent agent type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMockStore(t)
			ms.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(Store) error) error {
				return fn(ms)
			}).Maybe()

			itRepo := NewMockInfrastructureTypeRepository(t)
			itRepo.On("Get", mock.Anything, itID).Return(&InfrastructureType{BaseEntity: BaseEntity{ID: itID}, Name: "it"}, nil).Maybe()
			itRepo.On("Delete", mock.Anything, itID).Return(nil).Maybe()
			ms.On("InfrastructureTypeRepo").Return(itRepo).Maybe()

			infraRepo := NewMockInfrastructureRepository(t)
			infraRepo.On("CountByInfrastructureType", mock.Anything, itID).Return(tt.stubs.infraCount, nil).Maybe()
			ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

			atRepo := NewMockAgentTypeRepository(t)
			atRepo.On("CountByInfrastructureType", mock.Anything, itID).Return(tt.stubs.atCount, nil).Maybe()
			ms.On("AgentTypeRepo").Return(atRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("EventRepo").Return(eventRepo).Maybe()

			commander := NewInfrastructureTypeCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

			err := commander.Delete(ctx, itID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Delete() error = %v", err)
			}
		})
	}
}
