// Agent domain model unit tests
package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestAgent_TableName(t *testing.T) {
	agent := &Agent{}
	if agent.TableName() != "agents" {
		t.Errorf("Expected table name 'agents', got '%s'", agent.TableName())
	}
}

// setupMockStore creates a MockStore that delegates Atomic to fn(store) by default
func setupMockStore(t *testing.T) *MockStore {
	ms := NewMockStore(t)
	ms.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(Store) error) error {
		return fn(ms)
	}).Maybe()
	return ms
}

func TestAgentCommander_CreateWithConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		configuration properties.JSON
		wantErr       bool
		errContains   string
	}{
		{
			name: "valid configuration",
			configuration: properties.JSON{
				"apiEndpoint": "https://api.example.com",
				"maxRetries":  5,
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			configuration: properties.JSON{
				"maxRetries": 5,
			},
			wantErr:     true,
			errContains: "required property is missing",
		},
		{
			name: "invalid type",
			configuration: properties.JSON{
				"apiEndpoint": "https://api.example.com",
				"maxRetries":  "five", // Should be integer
			},
			wantErr:     true,
			errContains: "maxRetries",
		},
		{
			name: "default applied",
			configuration: properties.JSON{
				"apiEndpoint": "https://api.example.com",
				// maxRetries not provided, should get default
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			participantRepo := NewMockParticipantRepository(t)
			participantRepo.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Maybe()
			ms.On("ParticipantRepo").Return(participantRepo).Maybe()

			agentTypeRepo := NewMockAgentTypeRepository(t)
			agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(&AgentType{
				BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())},
				Name:       "Test Agent Type",
				ConfigurationSchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"apiEndpoint": {
							Type:     "string",
							Label:    "API Endpoint",
							Required: true,
						},
						"maxRetries": {
							Type:    "integer",
							Label:   "Max Retries",
							Default: 3,
						},
					},
				},
			}, nil).Maybe()
			ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

			agentRepo := NewMockAgentRepository(t)
			agentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("AgentRepo").Return(agentRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("EventRepo").Return(eventRepo).Maybe()

			engine := NewAgentConfigSchemaEngine(nil)
			installCmd := NewMockAgentInstallCommandCommander(t)
			installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
			commander := NewAgentCommander(ms, engine, installCmd)

			identity := &auth.Identity{
				Role: auth.RoleAdmin,
				ID:   properties.UUID(uuid.New()),
				Name: "Test Admin",
			}
			ctx := auth.WithIdentity(context.Background(), identity)

			params := CreateAgentParams{
				Name:          "Test Agent",
				ProviderID:    properties.UUID(uuid.New()),
				AgentTypeID:   properties.UUID(uuid.New()),
				Configuration: &tt.configuration,
			}

			agent, err := commander.Create(ctx, params)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Create() error should contain '%s', got: %v", tt.errContains, err)
				}
			}

			if !tt.wantErr {
				if agent == nil {
					t.Error("Expected agent to be created")
					return
				}
				if agent.Configuration == nil {
					t.Error("Expected configuration to be set")
					return
				}

				// For "default applied" test, verify default was applied
				if tt.name == "default applied" {
					configMap := map[string]any(*agent.Configuration)
					if maxRetries, ok := configMap["maxRetries"]; !ok || maxRetries != 3 {
						t.Errorf("Expected maxRetries default of 3, got %v", maxRetries)
					}
				}
			}
		})
	}
}

func TestAgentCommander_UpdateWithConfiguration(t *testing.T) {
	existingAgent := &Agent{
		BaseEntity:  BaseEntity{ID: properties.UUID(uuid.New())},
		Name:        "Existing Agent",
		AgentTypeID: properties.UUID(uuid.New()),
		ProviderID:  properties.UUID(uuid.New()),
		Configuration: &properties.JSON{
			"apiEndpoint": "https://old.example.com",
			"maxRetries":  3,
		},
		Status:           AgentNew,
		LastStatusUpdate: time.Now(),
	}

	t.Run("update with valid configuration", func(t *testing.T) {
		ms := setupMockStore(t)

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("Get", mock.Anything, mock.Anything).Return(existingAgent, nil).Maybe()
		agentRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		agentTypeRepo := NewMockAgentTypeRepository(t)
		agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(&AgentType{
			BaseEntity: BaseEntity{ID: existingAgent.AgentTypeID},
			Name:       "Test Agent Type",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"apiEndpoint": {
						Type:     "string",
						Label:    "API Endpoint",
						Required: true,
					},
					"maxRetries": {
						Type:    "integer",
						Label:   "Max Retries",
						Default: 3,
					},
				},
			},
		}, nil).Maybe()
		ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

		eventRepo := NewMockEventRepository(t)
		eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("EventRepo").Return(eventRepo).Maybe()

		engine := NewAgentConfigSchemaEngine(nil)
		installCmd := NewMockAgentInstallCommandCommander(t)
			installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
			commander := NewAgentCommander(ms, engine, installCmd)

		identity := &auth.Identity{
			Role: auth.RoleAdmin,
			ID:   properties.UUID(uuid.New()),
			Name: "Test Admin",
		}
		ctx := auth.WithIdentity(context.Background(), identity)

		newConfig := properties.JSON{
			"apiEndpoint": "https://new.example.com",
			"maxRetries":  5,
		}

		params := UpdateAgentParams{
			ID:            existingAgent.ID,
			Configuration: &newConfig,
		}

		agent, err := commander.Update(ctx, params)
		if err != nil {
			t.Errorf("Update() error = %v, expected no error", err)
			return
		}

		if agent == nil || agent.Configuration == nil {
			t.Error("Expected agent with configuration")
			return
		}

		configMap := map[string]any(*agent.Configuration)
		if configMap["apiEndpoint"] != "https://new.example.com" {
			t.Errorf("Expected apiEndpoint to be updated, got %v", configMap["apiEndpoint"])
		}
	})

	t.Run("update with invalid type", func(t *testing.T) {
		ms := setupMockStore(t)

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("Get", mock.Anything, mock.Anything).Return(existingAgent, nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		agentTypeRepo := NewMockAgentTypeRepository(t)
		agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(&AgentType{
			BaseEntity: BaseEntity{ID: existingAgent.AgentTypeID},
			Name:       "Test Agent Type",
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"apiEndpoint": {
						Type:     "string",
						Label:    "API Endpoint",
						Required: true,
					},
					"maxRetries": {
						Type:    "integer",
						Label:   "Max Retries",
						Default: 3,
					},
				},
			},
		}, nil).Maybe()
		ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

		engine := NewAgentConfigSchemaEngine(nil)
		installCmd := NewMockAgentInstallCommandCommander(t)
			installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
			commander := NewAgentCommander(ms, engine, installCmd)

		identity := &auth.Identity{
			Role: auth.RoleAdmin,
			ID:   properties.UUID(uuid.New()),
			Name: "Test Admin",
		}
		ctx := auth.WithIdentity(context.Background(), identity)

		newConfig := properties.JSON{
			"apiEndpoint": "https://new.example.com",
			"maxRetries":  "invalid", // Should be integer
		}

		params := UpdateAgentParams{
			ID:            existingAgent.ID,
			Configuration: &newConfig,
		}

		_, err := commander.Update(ctx, params)
		if err == nil {
			t.Error("Update() expected error for invalid type")
		}
	})
}

func TestAgentCommander_ServicePoolSetValidation(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	otherProviderID := properties.UUID(uuid.New())
	agentTypeID := properties.UUID(uuid.New())
	validPoolSetID := properties.UUID(uuid.New())
	invalidPoolSetID := properties.UUID(uuid.New())

	existingAgent := &Agent{
		BaseEntity:       BaseEntity{ID: properties.UUID(uuid.New())},
		Name:             "Test Agent",
		AgentTypeID:      agentTypeID,
		ProviderID:       providerID,
		Status:           AgentNew,
		LastStatusUpdate: time.Now(),
	}

	t.Run("create with valid pool set", func(t *testing.T) {
		ms := setupMockStore(t)

		participantRepo := NewMockParticipantRepository(t)
		participantRepo.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		ms.On("ParticipantRepo").Return(participantRepo).Maybe()

		agentTypeRepo := NewMockAgentTypeRepository(t)
		agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(&AgentType{
			BaseEntity:          BaseEntity{ID: agentTypeID},
			ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{}},
		}, nil).Maybe()
		ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		eventRepo := NewMockEventRepository(t)
		eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("EventRepo").Return(eventRepo).Maybe()

		servicePoolSetRepo := NewMockServicePoolSetRepository(t)
		servicePoolSetRepo.On("Get", mock.Anything, validPoolSetID).Return(&ServicePoolSet{
			BaseEntity: BaseEntity{ID: validPoolSetID},
			ProviderID: providerID,
		}, nil).Maybe()
		ms.On("ServicePoolSetRepo").Return(servicePoolSetRepo).Maybe()

		installCmd := NewMockAgentInstallCommandCommander(t)
		installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
		commander := NewAgentCommander(ms, NewAgentConfigSchemaEngine(nil), installCmd)
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		_, err := commander.Create(ctx, CreateAgentParams{
			Name:             "Agent",
			ProviderID:       providerID,
			AgentTypeID:      agentTypeID,
			ServicePoolSetID: &validPoolSetID,
		})
		if err != nil {
			t.Errorf("Create() error = %v, expected success", err)
		}
	})

	t.Run("create with invalid pool set should fail", func(t *testing.T) {
		ms := setupMockStore(t)

		participantRepo := NewMockParticipantRepository(t)
		participantRepo.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		ms.On("ParticipantRepo").Return(participantRepo).Maybe()

		agentTypeRepo := NewMockAgentTypeRepository(t)
		agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(&AgentType{
			BaseEntity:          BaseEntity{ID: agentTypeID},
			ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{}},
		}, nil).Maybe()
		ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		eventRepo := NewMockEventRepository(t)
		eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
		ms.On("EventRepo").Return(eventRepo).Maybe()

		servicePoolSetRepo := NewMockServicePoolSetRepository(t)
		servicePoolSetRepo.On("Get", mock.Anything, invalidPoolSetID).Return(&ServicePoolSet{
			BaseEntity: BaseEntity{ID: invalidPoolSetID},
			ProviderID: otherProviderID,
		}, nil).Maybe()
		ms.On("ServicePoolSetRepo").Return(servicePoolSetRepo).Maybe()

		installCmd := NewMockAgentInstallCommandCommander(t)
		installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
		commander := NewAgentCommander(ms, NewAgentConfigSchemaEngine(nil), installCmd)
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		_, err := commander.Create(ctx, CreateAgentParams{
			Name:             "Agent",
			ProviderID:       providerID,
			AgentTypeID:      agentTypeID,
			ServicePoolSetID: &invalidPoolSetID,
		})
		if err == nil {
			t.Error("Create() expected error for pool set from different provider")
		}
		if err != nil && !strings.Contains(err.Error(), "does not belong to provider") {
			t.Errorf("Create() error = %v, expected 'does not belong to provider'", err)
		}
	})

	t.Run("update with invalid pool set should fail", func(t *testing.T) {
		ms := setupMockStore(t)

		agentRepo := NewMockAgentRepository(t)
		agentRepo.On("Get", mock.Anything, mock.Anything).Return(existingAgent, nil).Maybe()
		ms.On("AgentRepo").Return(agentRepo).Maybe()

		agentTypeRepo := NewMockAgentTypeRepository(t)
		agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(&AgentType{
			BaseEntity:          BaseEntity{ID: agentTypeID},
			ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{}},
		}, nil).Maybe()
		ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

		servicePoolSetRepo := NewMockServicePoolSetRepository(t)
		servicePoolSetRepo.On("Get", mock.Anything, invalidPoolSetID).Return(&ServicePoolSet{
			BaseEntity: BaseEntity{ID: invalidPoolSetID},
			ProviderID: otherProviderID,
		}, nil).Maybe()
		ms.On("ServicePoolSetRepo").Return(servicePoolSetRepo).Maybe()

		installCmd := NewMockAgentInstallCommandCommander(t)
		installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
		commander := NewAgentCommander(ms, NewAgentConfigSchemaEngine(nil), installCmd)
		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

		_, err := commander.Update(ctx, UpdateAgentParams{
			ID:               existingAgent.ID,
			ServicePoolSetID: &invalidPoolSetID,
		})
		if err == nil {
			t.Error("Update() expected error for pool set from different provider")
		}
		if err != nil && !strings.Contains(err.Error(), "does not belong to provider") {
			t.Errorf("Update() error = %v, expected 'does not belong to provider'", err)
		}
	})
}

func TestAgentCommander_CreateWithPoolGenerator(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	poolValueID := properties.UUID(uuid.New())

	agentType := &AgentType{
		BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())},
		Name:       "Pool-Using Agent Type",
		ConfigurationSchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"publicIP": {
					Type:  "string",
					Label: "Public IP",
					Generator: &schema.GeneratorConfig{
						Type:   "pool",
						Config: map[string]any{"poolType": "public_ip"},
					},
				},
			},
		},
	}
	pool := &AgentPool{
		BaseEntity:    BaseEntity{ID: poolID},
		Type:          "public_ip",
		PropertyType:  "string",
		GeneratorType: PoolGeneratorList,
	}

	tests := []struct {
		name        string
		setupPool   func(*MockAgentPoolRepository)
		setupValue  func(*MockAgentPoolValueRepository)
		wantErr     bool
		errContains string
		wantIP      string
	}{
		{
			name: "happy path allocates from pool",
			setupPool: func(r *MockAgentPoolRepository) {
				r.On("FindByType", mock.Anything, "public_ip").Return(pool, nil)
			},
			setupValue: func(r *MockAgentPoolValueRepository) {
				r.On("FindAvailable", mock.Anything, poolID).Return([]*AgentPoolValue{
					{BaseEntity: BaseEntity{ID: poolValueID}, AgentPoolID: poolID, Value: "203.0.113.10"},
				}, nil)
				r.On("Update", mock.Anything, mock.MatchedBy(func(v *AgentPoolValue) bool {
					return v.AgentID != nil && v.PropertyName != nil && *v.PropertyName == "publicIP" && v.AllocatedAt != nil
				})).Return(nil).Once()
			},
			wantIP: "203.0.113.10",
		},
		{
			name: "FindByType errors",
			setupPool: func(r *MockAgentPoolRepository) {
				r.On("FindByType", mock.Anything, "public_ip").Return(nil, errors.New("pool lookup boom"))
			},
			setupValue:  func(r *MockAgentPoolValueRepository) {},
			wantErr:     true,
			errContains: "pool lookup boom",
		},
		{
			name: "no available values",
			setupPool: func(r *MockAgentPoolRepository) {
				r.On("FindByType", mock.Anything, "public_ip").Return(pool, nil)
			},
			setupValue: func(r *MockAgentPoolValueRepository) {
				r.On("FindAvailable", mock.Anything, poolID).Return([]*AgentPoolValue{}, nil)
			},
			wantErr:     true,
			errContains: "no available values",
		},
		{
			name: "Update errors",
			setupPool: func(r *MockAgentPoolRepository) {
				r.On("FindByType", mock.Anything, "public_ip").Return(pool, nil)
			},
			setupValue: func(r *MockAgentPoolValueRepository) {
				r.On("FindAvailable", mock.Anything, poolID).Return([]*AgentPoolValue{
					{BaseEntity: BaseEntity{ID: poolValueID}, AgentPoolID: poolID, Value: "203.0.113.10"},
				}, nil)
				r.On("Update", mock.Anything, mock.AnythingOfType("*domain.AgentPoolValue")).Return(errors.New("update boom"))
			},
			wantErr:     true,
			errContains: "update boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			participantRepo := NewMockParticipantRepository(t)
			participantRepo.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Maybe()
			ms.On("ParticipantRepo").Return(participantRepo).Maybe()

			agentTypeRepo := NewMockAgentTypeRepository(t)
			agentTypeRepo.On("Get", mock.Anything, mock.Anything).Return(agentType, nil).Maybe()
			ms.On("AgentTypeRepo").Return(agentTypeRepo).Maybe()

			agentPoolRepo := NewMockAgentPoolRepository(t)
			tt.setupPool(agentPoolRepo)
			ms.On("AgentPoolRepo").Return(agentPoolRepo).Maybe()

			agentPoolValueRepo := NewMockAgentPoolValueRepository(t)
			tt.setupValue(agentPoolValueRepo)
			ms.On("AgentPoolValueRepo").Return(agentPoolValueRepo).Maybe()

			agentRepo := NewMockAgentRepository(t)
			agentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("AgentRepo").Return(agentRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("EventRepo").Return(eventRepo).Maybe()

			installCmd := NewMockAgentInstallCommandCommander(t)
		installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
		commander := NewAgentCommander(ms, NewAgentConfigSchemaEngine(nil), installCmd)
			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

			initialConfig := properties.JSON{}
			agent, err := commander.Create(ctx, CreateAgentParams{
				Name:          "Pool Agent",
				ProviderID:    properties.UUID(uuid.New()),
				AgentTypeID:   properties.UUID(uuid.New()),
				Configuration: &initialConfig,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() error = %v, want nil", err)
			}
			if agent == nil || agent.Configuration == nil {
				t.Fatal("expected agent with populated configuration")
			}
			configMap := map[string]any(*agent.Configuration)
			if configMap["publicIP"] != tt.wantIP {
				t.Errorf("expected publicIP=%v, got %v", tt.wantIP, configMap["publicIP"])
			}
			if agent.ID == (properties.UUID{}) {
				t.Error("expected agent.ID to be pre-assigned")
			}
		})
	}
}

func TestAgentCommander_DeleteReleasesPoolValues(t *testing.T) {
	agentID := properties.UUID(uuid.New())
	poolA := properties.UUID(uuid.New())
	poolB := properties.UUID(uuid.New())
	now := time.Now()

	tests := []struct {
		name        string
		setupValues func(*MockAgentPoolValueRepository)
		setupPools  func(*MockAgentPoolRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "single pool, two allocations",
			setupValues: func(r *MockAgentPoolValueRepository) {
				r.On("FindByAgent", mock.Anything, agentID).Return([]*AgentPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolA, AgentID: &agentID, AllocatedAt: &now, Value: "ip1"},
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolA, AgentID: &agentID, AllocatedAt: &now, Value: "ip2"},
				}, nil).Once()
				r.On("Update", mock.Anything, mock.MatchedBy(func(v *AgentPoolValue) bool {
					return v.AgentID == nil && v.AllocatedAt == nil && v.PropertyName == nil
				})).Return(nil).Twice()
			},
			setupPools: func(r *MockAgentPoolRepository) {
				r.On("Get", mock.Anything, poolA).Return(&AgentPool{
					BaseEntity: BaseEntity{ID: poolA}, Type: "public_ip", PropertyType: "string", GeneratorType: PoolGeneratorList,
				}, nil).Once()
			},
		},
		{
			name: "multiple pools → dedup via seen map, one Get per pool",
			setupValues: func(r *MockAgentPoolValueRepository) {
				r.On("FindByAgent", mock.Anything, agentID).Return([]*AgentPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolA, AgentID: &agentID, AllocatedAt: &now, Value: "ip1"},
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolB, AgentID: &agentID, AllocatedAt: &now, Value: "hn1"},
				}, nil).Once()
				r.On("Update", mock.Anything, mock.MatchedBy(func(v *AgentPoolValue) bool {
					return v.AgentID == nil && v.AllocatedAt == nil && v.PropertyName == nil
				})).Return(nil).Twice()
			},
			setupPools: func(r *MockAgentPoolRepository) {
				r.On("Get", mock.Anything, poolA).Return(&AgentPool{
					BaseEntity: BaseEntity{ID: poolA}, Type: "public_ip", PropertyType: "string", GeneratorType: PoolGeneratorList,
				}, nil).Once()
				r.On("Get", mock.Anything, poolB).Return(&AgentPool{
					BaseEntity: BaseEntity{ID: poolB}, Type: "hostname", PropertyType: "string", GeneratorType: PoolGeneratorList,
				}, nil).Once()
			},
		},
		{
			name: "no allocations → no pool Get, no Update",
			setupValues: func(r *MockAgentPoolValueRepository) {
				r.On("FindByAgent", mock.Anything, agentID).Return([]*AgentPoolValue{}, nil).Once()
			},
			setupPools: func(r *MockAgentPoolRepository) {},
		},
		{
			name: "FindByAgent errors",
			setupValues: func(r *MockAgentPoolValueRepository) {
				r.On("FindByAgent", mock.Anything, agentID).Return(nil, errors.New("db boom")).Once()
			},
			setupPools:  func(r *MockAgentPoolRepository) {},
			wantErr:     true,
			errContains: "db boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			agentRepo := NewMockAgentRepository(t)
			agentRepo.On("Get", mock.Anything, agentID).Return(&Agent{
				BaseEntity:       BaseEntity{ID: agentID},
				Name:             "Agent to delete",
				AgentTypeID:      properties.UUID(uuid.New()),
				ProviderID:       properties.UUID(uuid.New()),
				Status:           AgentDisconnected,
				LastStatusUpdate: now,
			}, nil).Maybe()
			agentRepo.On("Delete", mock.Anything, agentID).Return(nil).Maybe()
			ms.On("AgentRepo").Return(agentRepo).Maybe()

			serviceRepo := NewMockServiceRepository(t)
			serviceRepo.On("CountByAgent", mock.Anything, agentID).Return(int64(0), nil).Maybe()
			ms.On("ServiceRepo").Return(serviceRepo).Maybe()

			tokenRepo := NewMockTokenRepository(t)
			tokenRepo.On("DeleteByAgentID", mock.Anything, agentID).Return(nil).Maybe()
			ms.On("TokenRepo").Return(tokenRepo).Maybe()

			agentPoolValueRepo := NewMockAgentPoolValueRepository(t)
			tt.setupValues(agentPoolValueRepo)
			ms.On("AgentPoolValueRepo").Return(agentPoolValueRepo).Maybe()

			agentPoolRepo := NewMockAgentPoolRepository(t)
			tt.setupPools(agentPoolRepo)
			ms.On("AgentPoolRepo").Return(agentPoolRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("EventRepo").Return(eventRepo).Maybe()

			installCmd := NewMockAgentInstallCommandCommander(t)
		installCmd.On("DeleteByAgentID", mock.Anything, mock.Anything).Return(nil).Maybe()
		commander := NewAgentCommander(ms, NewAgentConfigSchemaEngine(nil), installCmd)
			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

			err := commander.Delete(ctx, agentID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Delete() error = %v, want nil", err)
			}
		})
	}
}

func TestAgent_Update_ServicePoolSetID(t *testing.T) {
	t.Run("updating ServicePoolSetID clears ServicePoolSet association", func(t *testing.T) {
		oldPoolSetID := properties.UUID(uuid.New())
		newPoolSetID := properties.UUID(uuid.New())

		agent := &Agent{
			BaseEntity:       BaseEntity{ID: properties.UUID(uuid.New())},
			Name:             "Test Agent",
			AgentTypeID:      properties.UUID(uuid.New()),
			ProviderID:       properties.UUID(uuid.New()),
			Status:           AgentNew,
			LastStatusUpdate: time.Now(),
			ServicePoolSetID: &oldPoolSetID,
			ServicePoolSet: &ServicePoolSet{
				BaseEntity: BaseEntity{ID: oldPoolSetID},
				Name:       "Old Pool Set",
			},
		}

		if agent.ServicePoolSetID == nil || *agent.ServicePoolSetID != oldPoolSetID {
			t.Error("Expected agent to have old ServicePoolSetID")
		}
		if agent.ServicePoolSet == nil || agent.ServicePoolSet.ID != oldPoolSetID {
			t.Error("Expected agent to have old ServicePoolSet association")
		}

		updated := agent.Update(nil, nil, nil, &newPoolSetID)

		if !updated {
			t.Error("Expected Update() to return true")
		}
		if agent.ServicePoolSetID == nil || *agent.ServicePoolSetID != newPoolSetID {
			t.Errorf("Expected ServicePoolSetID to be updated to %v, got %v", newPoolSetID, agent.ServicePoolSetID)
		}

		if agent.ServicePoolSet != nil {
			t.Errorf("Expected ServicePoolSet to be nil after update, but got %v", agent.ServicePoolSet)
		}
	})

	t.Run("updating ServicePoolSetID when association is nil", func(t *testing.T) {
		newPoolSetID := properties.UUID(uuid.New())

		agent := &Agent{
			BaseEntity:       BaseEntity{ID: properties.UUID(uuid.New())},
			Name:             "Test Agent",
			AgentTypeID:      properties.UUID(uuid.New()),
			ProviderID:       properties.UUID(uuid.New()),
			Status:           AgentNew,
			LastStatusUpdate: time.Now(),
			ServicePoolSetID: nil,
			ServicePoolSet:   nil,
		}

		updated := agent.Update(nil, nil, nil, &newPoolSetID)

		if !updated {
			t.Error("Expected Update() to return true")
		}
		if agent.ServicePoolSetID == nil || *agent.ServicePoolSetID != newPoolSetID {
			t.Errorf("Expected ServicePoolSetID to be set to %v, got %v", newPoolSetID, agent.ServicePoolSetID)
		}
		if agent.ServicePoolSet != nil {
			t.Error("Expected ServicePoolSet to remain nil")
		}
	})
}
