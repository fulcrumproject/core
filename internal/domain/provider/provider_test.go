package provider

import (
	"testing"

	"fulcrumproject.org/core/internal/domain/common"
	"github.com/google/uuid"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name        string
		provName    string
		countryCode string
		attributes  common.Attributes
		wantErr     bool
	}{
		{
			name:        "Valid provider",
			provName:    "Test Provider",
			countryCode: "US",
			attributes: common.Attributes{
				"region": {"east", "west"},
				"tier":   {"premium"},
			},
			wantErr: false,
		},
		{
			name:        "Empty name",
			provName:    "",
			countryCode: "US",
			attributes:  common.Attributes{},
			wantErr:     true,
		},
		{
			name:        "Invalid country code",
			provName:    "Test Provider",
			countryCode: "USA",
			attributes:  common.Attributes{},
			wantErr:     true,
		},
		{
			name:        "Invalid attributes key",
			provName:    "Test Provider",
			countryCode: "US",
			attributes: common.Attributes{
				"": {"value"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.provName, tt.countryCode, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if provider.Name != tt.provName {
				t.Errorf("NewProvider() name = %v, want %v", provider.Name, tt.provName)
			}
			if provider.CountryCode != tt.countryCode {
				t.Errorf("NewProvider() countryCode = %v, want %v", provider.CountryCode, tt.countryCode)
			}
			if provider.State != ProviderEnabled {
				t.Errorf("NewProvider() state = %v, want %v", provider.State, ProviderEnabled)
			}

			// Test attributes conversion
			attrs, err := provider.GetAttributes()
			if err != nil {
				t.Errorf("GetAttributes() error = %v", err)
				return
			}

			for key, values := range tt.attributes {
				gotValues, ok := attrs[key]
				if !ok {
					t.Errorf("Attribute key %s not found", key)
					continue
				}
				if len(values) != len(gotValues) {
					t.Errorf("Attribute values length mismatch for key %s", key)
					continue
				}
				for i, value := range values {
					if value != gotValues[i] {
						t.Errorf("Attribute value mismatch at index %d for key %s", i, key)
					}
				}
			}
		})
	}
}

func TestProviderValidate(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		wantErr  bool
	}{
		{
			name: "Valid provider",
			provider: &Provider{
				Name:        "Test Provider",
				State:       ProviderEnabled,
				CountryCode: "US",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			provider: &Provider{
				Name:        "",
				State:       ProviderEnabled,
				CountryCode: "US",
			},
			wantErr: true,
		},
		{
			name: "Invalid state",
			provider: &Provider{
				Name:        "Test Provider",
				State:       "InvalidState",
				CountryCode: "US",
			},
			wantErr: true,
		},
		{
			name: "Invalid country code",
			provider: &Provider{
				Name:        "Test Provider",
				State:       ProviderEnabled,
				CountryCode: "USA",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.provider.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Provider.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderAgentManagement(t *testing.T) {
	provider := &Provider{
		BaseEntity: common.BaseEntity{
			ID: uuid.New(),
		},
		Name:        "Test Provider",
		State:       ProviderEnabled,
		CountryCode: "US",
	}

	agent1 := &Agent{
		BaseEntity: common.BaseEntity{
			ID: uuid.New(),
		},
		Name:        "Agent 1",
		State:       AgentNew,
		CountryCode: "US",
		ProviderID:  provider.ID,
	}

	agent2 := &Agent{
		BaseEntity: common.BaseEntity{
			ID: uuid.New(),
		},
		Name:        "Agent 2",
		State:       AgentNew,
		CountryCode: "US",
		ProviderID:  provider.ID,
	}

	t.Run("Add agent", func(t *testing.T) {
		if err := provider.AddAgent(agent1); err != nil {
			t.Errorf("AddAgent() error = %v", err)
		}
		if len(provider.Agents) != 1 {
			t.Errorf("AddAgent() agents length = %v, want 1", len(provider.Agents))
		}
	})

	t.Run("Add second agent", func(t *testing.T) {
		if err := provider.AddAgent(agent2); err != nil {
			t.Errorf("AddAgent() error = %v", err)
		}
		if len(provider.Agents) != 2 {
			t.Errorf("AddAgent() agents length = %v, want 2", len(provider.Agents))
		}
	})

	t.Run("Get agent", func(t *testing.T) {
		agent, err := provider.GetAgent(agent1.ID)
		if err != nil {
			t.Errorf("GetAgent() error = %v", err)
		}
		if agent.ID != agent1.ID {
			t.Errorf("GetAgent() agent ID = %v, want %v", agent.ID, agent1.ID)
		}
	})

	t.Run("Remove agent", func(t *testing.T) {
		if err := provider.RemoveAgent(agent1.ID); err != nil {
			t.Errorf("RemoveAgent() error = %v", err)
		}
		if len(provider.Agents) != 1 {
			t.Errorf("RemoveAgent() agents length = %v, want 1", len(provider.Agents))
		}
	})

	t.Run("Get non-existent agent", func(t *testing.T) {
		_, err := provider.GetAgent(agent1.ID)
		if err == nil {
			t.Error("GetAgent() expected error for non-existent agent")
		}
	})
}

func TestProviderStateTransitions(t *testing.T) {
	provider := &Provider{
		Name:        "Test Provider",
		State:       ProviderEnabled,
		CountryCode: "US",
	}

	t.Run("Disable enabled provider", func(t *testing.T) {
		if err := provider.Disable(); err != nil {
			t.Errorf("Disable() error = %v", err)
		}
		if provider.State != ProviderDisabled {
			t.Errorf("Disable() state = %v, want %v", provider.State, ProviderDisabled)
		}
	})

	t.Run("Disable already disabled provider", func(t *testing.T) {
		if err := provider.Disable(); err != nil {
			t.Errorf("Disable() error = %v", err)
		}
		if provider.State != ProviderDisabled {
			t.Errorf("Disable() state = %v, want %v", provider.State, ProviderDisabled)
		}
	})

	t.Run("Enable disabled provider", func(t *testing.T) {
		if err := provider.Enable(); err != nil {
			t.Errorf("Enable() error = %v", err)
		}
		if provider.State != ProviderEnabled {
			t.Errorf("Enable() state = %v, want %v", provider.State, ProviderEnabled)
		}
	})

	t.Run("Enable already enabled provider", func(t *testing.T) {
		if err := provider.Enable(); err != nil {
			t.Errorf("Enable() error = %v", err)
		}
		if provider.State != ProviderEnabled {
			t.Errorf("Enable() state = %v, want %v", provider.State, ProviderEnabled)
		}
	})
}
