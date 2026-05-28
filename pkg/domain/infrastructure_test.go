// Infrastructure domain model unit tests.
package domain

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestInfrastructure_TableName(t *testing.T) {
	infra := &Infrastructure{}
	if infra.TableName() != "infrastructures" {
		t.Errorf("Expected table name 'infrastructures', got %q", infra.TableName())
	}
}

func TestNewInfrastructure(t *testing.T) {
	params := CreateInfrastructureParams{
		Name:                 "csi-node-01",
		ProviderID:           properties.UUID(uuid.New()),
		InfrastructureTypeID: properties.UUID(uuid.New()),
		Tags:                 []string{"region:eu"},
	}

	infra := NewInfrastructure(params)
	if infra.Name != params.Name {
		t.Errorf("Name mismatch: %q", infra.Name)
	}
	if infra.ProviderID != params.ProviderID {
		t.Errorf("ProviderID mismatch")
	}
	if infra.InfrastructureTypeID != params.InfrastructureTypeID {
		t.Errorf("InfrastructureTypeID mismatch")
	}
	if len(infra.Tags) != 1 || infra.Tags[0] != "region:eu" {
		t.Errorf("Tags mismatch: %v", infra.Tags)
	}
}

func TestInfrastructure_Validate(t *testing.T) {
	validProvider := properties.UUID(uuid.New())
	validType := properties.UUID(uuid.New())

	tests := []struct {
		name        string
		infra       *Infrastructure
		wantErr     bool
		errContains string
	}{
		{
			name: "happy path",
			infra: &Infrastructure{
				Name:                 "ok",
				InfrastructureTypeID: validType,
				ProviderID:           validProvider,
			},
		},
		{
			name: "empty name",
			infra: &Infrastructure{
				InfrastructureTypeID: validType,
				ProviderID:           validProvider,
			},
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name: "nil infrastructure type id",
			infra: &Infrastructure{
				Name:       "ok",
				ProviderID: validProvider,
			},
			wantErr:     true,
			errContains: "infrastructure type ID",
		},
		{
			name: "nil provider id",
			infra: &Infrastructure{
				Name:                 "ok",
				InfrastructureTypeID: validType,
			},
			wantErr:     true,
			errContains: "provider ID",
		},
		{
			name: "empty tag at index",
			infra: &Infrastructure{
				Name:                 "ok",
				InfrastructureTypeID: validType,
				ProviderID:           validProvider,
				Tags:                 []string{""},
			},
			wantErr:     true,
			errContains: "tag at index 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.infra.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Validate() error should contain %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestInfrastructure_Update(t *testing.T) {
	makeInfra := func() *Infrastructure {
		return &Infrastructure{
			BaseEntity:           BaseEntity{ID: properties.UUID(uuid.New())},
			Name:                 "Initial",
			InfrastructureTypeID: properties.UUID(uuid.New()),
			ProviderID:           properties.UUID(uuid.New()),
			Tags:                 []string{"old"},
		}
	}

	t.Run("update name only", func(t *testing.T) {
		infra := makeInfra()
		newName := "Renamed"
		infra.Update(UpdateInfrastructureParams{Name: &newName})
		if infra.Name != "Renamed" {
			t.Errorf("Expected 'Renamed', got %q", infra.Name)
		}
	})

	t.Run("update tags only", func(t *testing.T) {
		infra := makeInfra()
		newTags := []string{"a", "b"}
		infra.Update(UpdateInfrastructureParams{Tags: &newTags})
		if len(infra.Tags) != 2 || infra.Tags[0] != "a" || infra.Tags[1] != "b" {
			t.Errorf("Expected new tags, got %v", infra.Tags)
		}
	})

	t.Run("update configuration only", func(t *testing.T) {
		infra := makeInfra()
		cfg := properties.JSON{"endpoint": "https://x"}
		infra.Update(UpdateInfrastructureParams{Configuration: &cfg})
		if infra.Configuration == nil || (*infra.Configuration)["endpoint"] != "https://x" {
			t.Errorf("Expected configuration to be set, got %v", infra.Configuration)
		}
	})

	t.Run("nil pointers leave fields untouched", func(t *testing.T) {
		infra := makeInfra()
		before := *infra
		infra.Update(UpdateInfrastructureParams{})
		if infra.Name != before.Name || len(infra.Tags) != len(before.Tags) {
			t.Error("Update with nil params mutated fields")
		}
	})
}

// setupInfraMockStore mirrors setupMockStore in agent_test.go.
func setupInfraMockStore(t *testing.T) *MockStore {
	ms := NewMockStore(t)
	ms.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(Store) error) error {
		return fn(ms)
	}).Maybe()
	return ms
}

func TestInfrastructureCommander_Create(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	infraTypeID := properties.UUID(uuid.New())

	infraType := &InfrastructureType{
		BaseEntity: BaseEntity{ID: infraTypeID},
		Name:       "fulcrum-csp-node",
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint":   {Type: "string", Label: "Endpoint", Required: true},
					"maxRetries": {Type: "integer", Label: "Max Retries", Default: 3},
				},
			},
			ConfigContentType: "text/plain",
		},
	}

	tests := []struct {
		name             string
		providerExists   bool
		providerErr      error
		infraTypeGetErr  error
		configuration    *properties.JSON
		wantErr          bool
		errContains      string
		assertConfigured func(t *testing.T, infra *Infrastructure)
	}{
		{
			name:           "happy path with valid configuration",
			providerExists: true,
			configuration: &properties.JSON{
				"endpoint":   "https://example",
				"maxRetries": 5,
			},
		},
		{
			name:           "applies default when value omitted",
			providerExists: true,
			configuration: &properties.JSON{
				"endpoint": "https://example",
			},
			assertConfigured: func(t *testing.T, infra *Infrastructure) {
				cfg := map[string]any(*infra.Configuration)
				if cfg["maxRetries"] != 3 {
					t.Errorf("expected maxRetries default 3, got %v", cfg["maxRetries"])
				}
			},
		},
		{
			name:           "missing required property",
			providerExists: true,
			configuration: &properties.JSON{
				"maxRetries": 5,
			},
			wantErr:     true,
			errContains: "required property is missing",
		},
		{
			name:           "provider does not exist",
			providerExists: false,
			wantErr:        true,
			errContains:    "provider with ID",
		},
		{
			name:           "provider exists check errors",
			providerErr:    errors.New("provider lookup boom"),
			providerExists: false,
			wantErr:        true,
			errContains:    "provider lookup boom",
		},
		{
			name:            "infrastructure type does not exist",
			providerExists:  true,
			infraTypeGetErr: NotFoundError{Err: errors.New("not found")},
			wantErr:         true,
			errContains:     "infrastructure type with ID",
		},
		{
			name:            "infrastructure type lookup errors",
			providerExists:  true,
			infraTypeGetErr: errors.New("infra type lookup boom"),
			wantErr:         true,
			errContains:     "infra type lookup boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupInfraMockStore(t)

			participantRepo := NewMockParticipantRepository(t)
			participantRepo.On("Exists", mock.Anything, providerID).Return(tt.providerExists, tt.providerErr).Maybe()
			ms.On("ParticipantRepo").Return(participantRepo).Maybe()

			infraTypeRepo := NewMockInfrastructureTypeRepository(t)
			if tt.infraTypeGetErr != nil {
				infraTypeRepo.On("Get", mock.Anything, infraTypeID).Return(nil, tt.infraTypeGetErr).Maybe()
			} else {
				infraTypeRepo.On("Get", mock.Anything, infraTypeID).Return(infraType, nil).Maybe()
			}
			ms.On("InfrastructureTypeRepo").Return(infraTypeRepo).Maybe()

			infraRepo := NewMockInfrastructureRepository(t)
			infraRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			eventRepo.On("Create", mock.Anything, mock.MatchedBy(func(e *Event) bool {
				return e.Type == EventTypeInfrastructureCreated
			})).Return(nil).Maybe()
			ms.On("EventRepo").Return(eventRepo).Maybe()

			engine := NewInfrastructureConfigSchemaEngine(nil)
			commander := NewInfrastructureCommander(ms, engine)

			ctx := auth.WithIdentity(context.Background(), &auth.Identity{
				Role: auth.RoleAdmin,
				ID:   properties.UUID(uuid.New()),
				Name: "admin",
			})

			infra, err := commander.Create(ctx, CreateInfrastructureParams{
				Name:                 "infra-1",
				ProviderID:           providerID,
				InfrastructureTypeID: infraTypeID,
				Configuration:        tt.configuration,
			})

			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Create() error should contain %q, got %v", tt.errContains, err)
				}
				return
			}
			if infra == nil {
				t.Fatal("expected infrastructure to be returned")
			}
			if infra.ID == (properties.UUID{}) {
				t.Error("expected infra.ID to be pre-assigned")
			}
			if tt.assertConfigured != nil {
				tt.assertConfigured(t, infra)
			}
		})
	}
}

func TestInfrastructureCommander_Update(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	infraTypeID := properties.UUID(uuid.New())
	infraID := properties.UUID(uuid.New())

	existing := &Infrastructure{
		BaseEntity:           BaseEntity{ID: infraID},
		Name:                 "Existing",
		InfrastructureTypeID: infraTypeID,
		ProviderID:           providerID,
		Configuration: &properties.JSON{
			"endpoint":   "https://old",
			"maxRetries": 3,
		},
	}

	infraType := &InfrastructureType{
		BaseEntity: BaseEntity{ID: infraTypeID},
		Name:       "fulcrum-csp-node",
		TemplateValidation: TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint":   {Type: "string", Required: true},
					"maxRetries": {Type: "integer", Default: 3},
				},
			},
		},
	}

	t.Run("happy path", func(t *testing.T) {
		ms := setupInfraMockStore(t)

		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.On("Get", mock.Anything, infraID).Return(existing, nil).Maybe()
		infraRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

		infraTypeRepo := NewMockInfrastructureTypeRepository(t)
		infraTypeRepo.On("Get", mock.Anything, infraTypeID).Return(infraType, nil).Maybe()
		ms.On("InfrastructureTypeRepo").Return(infraTypeRepo).Maybe()

		eventRepo := NewMockEventRepository(t)
		eventRepo.On("Create", mock.Anything, mock.MatchedBy(func(e *Event) bool {
			return e.Type == EventTypeInfrastructureUpdated
		})).Return(nil).Maybe()
		ms.On("EventRepo").Return(eventRepo).Maybe()

		commander := NewInfrastructureCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		newCfg := properties.JSON{"endpoint": "https://new", "maxRetries": 5}
		infra, err := commander.Update(ctx, UpdateInfrastructureParams{
			ID:            infraID,
			Configuration: &newCfg,
		})
		if err != nil {
			t.Fatalf("Update() error = %v, want nil", err)
		}
		if infra == nil || infra.Configuration == nil {
			t.Fatal("expected infra with configuration")
		}
		if (*infra.Configuration)["endpoint"] != "https://new" {
			t.Errorf("expected endpoint=https://new, got %v", (*infra.Configuration)["endpoint"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		ms := setupInfraMockStore(t)

		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.On("Get", mock.Anything, infraID).Return(nil, NewNotFoundErrorf("infrastructure with ID %s not found", infraID)).Maybe()
		ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

		commander := NewInfrastructureCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		_, err := commander.Update(ctx, UpdateInfrastructureParams{ID: infraID})
		if err == nil {
			t.Fatal("expected not-found error")
		}
		var nfErr NotFoundError
		if !errors.As(err, &nfErr) {
			t.Errorf("expected NotFoundError, got %T: %v", err, err)
		}
	})

	t.Run("invalid configuration rejected", func(t *testing.T) {
		ms := setupInfraMockStore(t)

		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.On("Get", mock.Anything, infraID).Return(existing, nil).Maybe()
		ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

		infraTypeRepo := NewMockInfrastructureTypeRepository(t)
		infraTypeRepo.On("Get", mock.Anything, infraTypeID).Return(infraType, nil).Maybe()
		ms.On("InfrastructureTypeRepo").Return(infraTypeRepo).Maybe()

		commander := NewInfrastructureCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		bad := properties.JSON{"endpoint": "https://x", "maxRetries": "not-an-int"}
		_, err := commander.Update(ctx, UpdateInfrastructureParams{ID: infraID, Configuration: &bad})
		if err == nil {
			t.Fatal("expected error for invalid configuration type")
		}
	})
}

func TestInfrastructureCommander_Delete(t *testing.T) {
	infraID := properties.UUID(uuid.New())

	t.Run("happy path emits event then deletes", func(t *testing.T) {
		ms := setupInfraMockStore(t)

		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.On("Get", mock.Anything, infraID).Return(&Infrastructure{
			BaseEntity:           BaseEntity{ID: infraID},
			Name:                 "infra",
			InfrastructureTypeID: properties.UUID(uuid.New()),
			ProviderID:           properties.UUID(uuid.New()),
		}, nil).Maybe()
		infraRepo.On("Delete", mock.Anything, infraID).Return(nil).Maybe()
		ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("CountByInfrastructure", mock.Anything, infraID).Return(int64(0), nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		eventRepo := NewMockEventRepository(t)
		eventRepo.On("Create", mock.Anything, mock.MatchedBy(func(e *Event) bool {
			return e.Type == EventTypeInfrastructureDeleted
		})).Return(nil).Maybe()
		ms.On("EventRepo").Return(eventRepo).Maybe()

		commander := NewInfrastructureCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		if err := commander.Delete(ctx, infraID); err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
	})

	t.Run("blocks when dependent agents exist", func(t *testing.T) {
		ms := setupInfraMockStore(t)

		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.On("Get", mock.Anything, infraID).Return(&Infrastructure{
			BaseEntity:           BaseEntity{ID: infraID},
			Name:                 "infra",
			InfrastructureTypeID: properties.UUID(uuid.New()),
			ProviderID:           properties.UUID(uuid.New()),
		}, nil).Maybe()
		ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("CountByInfrastructure", mock.Anything, infraID).Return(int64(2), nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		commander := NewInfrastructureCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		err := commander.Delete(ctx, infraID)
		if err == nil {
			t.Fatal("expected error for dependent agents")
		}
		if !strings.Contains(err.Error(), "dependent agent") {
			t.Errorf("expected dependent-agent error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ms := setupInfraMockStore(t)

		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.On("Get", mock.Anything, infraID).Return(nil, NewNotFoundErrorf("infrastructure with ID %s not found", infraID)).Maybe()
		ms.On("InfrastructureRepo").Return(infraRepo).Maybe()

		commander := NewInfrastructureCommander(ms, NewInfrastructureConfigSchemaEngine(nil))
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		err := commander.Delete(ctx, infraID)
		if err == nil {
			t.Fatal("expected not-found error")
		}
		var nfErr NotFoundError
		if !errors.As(err, &nfErr) {
			t.Errorf("expected NotFoundError, got %T: %v", err, err)
		}
	})
}
