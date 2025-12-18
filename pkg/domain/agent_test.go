// Agent domain model unit tests
package domain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

func TestAgent_TableName(t *testing.T) {
	agent := &Agent{}
	if agent.TableName() != "agents" {
		t.Errorf("Expected table name 'agents', got '%s'", agent.TableName())
	}
}

func TestAgentCommander_CreateWithConfiguration(t *testing.T) {
	// Mock store and repositories
	var ms *mockStore
	ms = &mockStore{
		participantRepo: &mockParticipantRepository{
			existsFunc: func(ctx context.Context, id properties.UUID) (bool, error) {
				return true, nil
			},
		},
		agentTypeRepo: &mockAgentTypeRepository{
			getFunc: func(ctx context.Context, id properties.UUID) (*AgentType, error) {
				return &AgentType{
					BaseEntity: BaseEntity{ID: id},
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
				}, nil
			},
		},
		agentRepo: &mockAgentRepository{
			createFunc: func(ctx context.Context, agent *Agent) error {
				return nil
			},
		},
		eventRepo: &mockEventRepository{
			createFunc: func(ctx context.Context, event *Event) error {
				return nil
			},
		},
		atomicFunc: func(ctx context.Context, fn func(Store) error) error {
			return fn(ms)
		},
	}

	engine := NewAgentConfigSchemaEngine(nil)
	commander := NewAgentCommander(ms, engine)

	// Create context with mock identity for event creation
	identity := &auth.Identity{
		Role: auth.RoleAdmin,
		ID:   properties.UUID(uuid.New()),
		Name: "Test Admin",
	}
	ctx := auth.WithIdentity(context.Background(), identity)

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

	var ms *mockStore
	ms = &mockStore{
		agentRepo: &mockAgentRepository{
			getFunc: func(ctx context.Context, id properties.UUID) (*Agent, error) {
				return existingAgent, nil
			},
			saveFunc: func(ctx context.Context, agent *Agent) error {
				return nil
			},
		},
		agentTypeRepo: &mockAgentTypeRepository{
			getFunc: func(ctx context.Context, id properties.UUID) (*AgentType, error) {
				return &AgentType{
					BaseEntity: BaseEntity{ID: id},
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
				}, nil
			},
		},
		eventRepo: &mockEventRepository{
			createFunc: func(ctx context.Context, event *Event) error {
				return nil
			},
		},
		atomicFunc: func(ctx context.Context, fn func(Store) error) error {
			return fn(ms)
		},
	}

	engine := NewAgentConfigSchemaEngine(nil)
	commander := NewAgentCommander(ms, engine)

	// Create context with mock identity for event creation
	identity := &auth.Identity{
		Role: auth.RoleAdmin,
		ID:   properties.UUID(uuid.New()),
		Name: "Test Admin",
	}
	ctx := auth.WithIdentity(context.Background(), identity)

	t.Run("update with valid configuration", func(t *testing.T) {
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

func TestAgent_Update_ServicePoolSetID(t *testing.T) {
	t.Run("updating ServicePoolSetID clears ServicePoolSet association", func(t *testing.T) {
		oldPoolSetID := properties.UUID(uuid.New())
		newPoolSetID := properties.UUID(uuid.New())
		
		// Create agent with old ServicePoolSetID and simulate a preloaded association
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
		
		// Verify initial state
		if agent.ServicePoolSetID == nil || *agent.ServicePoolSetID != oldPoolSetID {
			t.Error("Expected agent to have old ServicePoolSetID")
		}
		if agent.ServicePoolSet == nil || agent.ServicePoolSet.ID != oldPoolSetID {
			t.Error("Expected agent to have old ServicePoolSet association")
		}
		
		// Update ServicePoolSetID
		updated := agent.Update(nil, nil, nil, &newPoolSetID)
		
		// Verify update was applied
		if !updated {
			t.Error("Expected Update() to return true")
		}
		if agent.ServicePoolSetID == nil || *agent.ServicePoolSetID != newPoolSetID {
			t.Errorf("Expected ServicePoolSetID to be updated to %v, got %v", newPoolSetID, agent.ServicePoolSetID)
		}
		
		// Verify ServicePoolSet association was cleared (this is the fix)
		if agent.ServicePoolSet != nil {
			t.Errorf("Expected ServicePoolSet to be nil after update, but got %v", agent.ServicePoolSet)
		}
	})
	
	t.Run("updating ServicePoolSetID when association is nil", func(t *testing.T) {
		newPoolSetID := properties.UUID(uuid.New())
		
		// Create agent without ServicePoolSet association (like in Create)
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
		
		// Update ServicePoolSetID
		updated := agent.Update(nil, nil, nil, &newPoolSetID)
		
		// Verify update was applied
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

// Mock implementations
type mockStore struct {
	participantRepo ParticipantRepository
	agentTypeRepo   AgentTypeRepository
	agentRepo       AgentRepository
	eventRepo       EventRepository
	atomicFunc      func(context.Context, func(Store) error) error
}

func (m *mockStore) ParticipantRepo() ParticipantRepository             { return m.participantRepo }
func (m *mockStore) AgentTypeRepo() AgentTypeRepository                 { return m.agentTypeRepo }
func (m *mockStore) AgentRepo() AgentRepository                         { return m.agentRepo }
func (m *mockStore) ServiceTypeRepo() ServiceTypeRepository             { return nil }
func (m *mockStore) ServiceRepo() ServiceRepository                     { return nil }
func (m *mockStore) ServiceGroupRepo() ServiceGroupRepository           { return nil }
func (m *mockStore) ServiceOptionTypeRepo() ServiceOptionTypeRepository { return nil }
func (m *mockStore) ServiceOptionRepo() ServiceOptionRepository         { return nil }
func (m *mockStore) ServicePoolSetRepo() ServicePoolSetRepository       { return nil }
func (m *mockStore) ServicePoolRepo() ServicePoolRepository             { return nil }
func (m *mockStore) ServicePoolValueRepo() ServicePoolValueRepository   { return nil }
func (m *mockStore) JobRepo() JobRepository                             { return nil }
func (m *mockStore) MetricTypeRepo() MetricTypeRepository               { return nil }
func (m *mockStore) MetricEntryRepo() MetricEntryRepository             { return nil }
func (m *mockStore) EventRepo() EventRepository                         { return m.eventRepo }
func (m *mockStore) EventSubscriptionRepo() EventSubscriptionRepository { return nil }
func (m *mockStore) TokenRepo() TokenRepository                         { return nil }
func (m *mockStore) Atomic(ctx context.Context, fn func(Store) error) error {
	if m.atomicFunc != nil {
		return m.atomicFunc(ctx, fn)
	}
	return fn(m)
}

type mockParticipantRepository struct {
	existsFunc func(context.Context, properties.UUID) (bool, error)
}

func (m *mockParticipantRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return false, nil
}
func (m *mockParticipantRepository) Create(context.Context, *Participant) error { return nil }
func (m *mockParticipantRepository) Get(context.Context, properties.UUID) (*Participant, error) {
	return nil, nil
}
func (m *mockParticipantRepository) Save(context.Context, *Participant) error      { return nil }
func (m *mockParticipantRepository) Delete(context.Context, properties.UUID) error { return nil }
func (m *mockParticipantRepository) AuthScope(context.Context, properties.UUID) (auth.ObjectScope, error) {
	return nil, nil
}
func (m *mockParticipantRepository) Count(context.Context) (int64, error) { return 0, nil }
func (m *mockParticipantRepository) List(context.Context, *auth.IdentityScope, *PageReq) (*PageRes[Participant], error) {
	return nil, nil
}

type mockAgentTypeRepository struct {
	getFunc func(context.Context, properties.UUID) (*AgentType, error)
}

func (m *mockAgentTypeRepository) Get(ctx context.Context, id properties.UUID) (*AgentType, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockAgentTypeRepository) Exists(context.Context, properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockAgentTypeRepository) Create(context.Context, *AgentType) error      { return nil }
func (m *mockAgentTypeRepository) Save(context.Context, *AgentType) error        { return nil }
func (m *mockAgentTypeRepository) Delete(context.Context, properties.UUID) error { return nil }
func (m *mockAgentTypeRepository) AuthScope(context.Context, properties.UUID) (auth.ObjectScope, error) {
	return nil, nil
}
func (m *mockAgentTypeRepository) Count(context.Context) (int64, error) { return 0, nil }
func (m *mockAgentTypeRepository) List(context.Context, *auth.IdentityScope, *PageReq) (*PageRes[AgentType], error) {
	return nil, nil
}

type mockAgentRepository struct {
	createFunc func(context.Context, *Agent) error
	getFunc    func(context.Context, properties.UUID) (*Agent, error)
	saveFunc   func(context.Context, *Agent) error
}

func (m *mockAgentRepository) Create(ctx context.Context, agent *Agent) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, agent)
	}
	return nil
}
func (m *mockAgentRepository) Get(ctx context.Context, id properties.UUID) (*Agent, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockAgentRepository) Save(ctx context.Context, agent *Agent) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, agent)
	}
	return nil
}
func (m *mockAgentRepository) Exists(context.Context, properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockAgentRepository) Delete(context.Context, properties.UUID) error { return nil }
func (m *mockAgentRepository) AuthScope(context.Context, properties.UUID) (auth.ObjectScope, error) {
	return nil, nil
}
func (m *mockAgentRepository) Count(context.Context) (int64, error) { return 0, nil }
func (m *mockAgentRepository) CountByAgentType(context.Context, properties.UUID) (int64, error) {
	return 0, nil
}
func (m *mockAgentRepository) CountByProvider(context.Context, properties.UUID) (int64, error) {
	return 0, nil
}
func (m *mockAgentRepository) List(context.Context, *auth.IdentityScope, *PageReq) (*PageRes[Agent], error) {
	return nil, nil
}
func (m *mockAgentRepository) FindByServiceTypeAndTags(context.Context, properties.UUID, []string) ([]*Agent, error) {
	return nil, nil
}
func (m *mockAgentRepository) MarkInactiveAgentsAsDisconnected(context.Context, time.Duration) (int64, error) {
	return 0, nil
}

type mockEventRepository struct {
	createFunc func(context.Context, *Event) error
}

func (m *mockEventRepository) Create(ctx context.Context, event *Event) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, event)
	}
	return nil
}
func (m *mockEventRepository) Get(context.Context, properties.UUID) (*Event, error) { return nil, nil }
func (m *mockEventRepository) AuthScope(context.Context, properties.UUID) (auth.ObjectScope, error) {
	return nil, nil
}
func (m *mockEventRepository) Count(context.Context) (int64, error)          { return 0, nil }
func (m *mockEventRepository) Delete(context.Context, properties.UUID) error { return nil }
func (m *mockEventRepository) Exists(context.Context, properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockEventRepository) List(context.Context, *auth.IdentityScope, *PageReq) (*PageRes[Event], error) {
	return nil, nil
}
func (m *mockEventRepository) ListFromSequence(context.Context, int64, int) ([]*Event, error) {
	return nil, nil
}
func (m *mockEventRepository) Save(context.Context, *Event) error { return nil }
func (m *mockEventRepository) ServiceUptime(context.Context, properties.UUID, time.Time, time.Time) (uint64, uint64, error) {
	return 0, 0, nil
}
