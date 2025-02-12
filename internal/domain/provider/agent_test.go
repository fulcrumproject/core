package provider

import (
	"encoding/json"
	"reflect"
	"testing"

	"fulcrumproject.org/core/internal/domain/common"
	"github.com/google/uuid"
)

func TestNewAgent(t *testing.T) {
	providerID := uuid.New()
	agentTypeID := uuid.New()

	tests := []struct {
		name        string
		agentName   string
		countryCode string
		attributes  common.Attributes
		properties  common.JSON
		providerID  common.UUID
		agentTypeID common.UUID
		wantErr     bool
	}{
		{
			name:        "Valid agent",
			agentName:   "Test Agent",
			countryCode: "US",
			attributes: common.Attributes{
				"zone":     {"zone1", "zone2"},
				"capacity": {"high"},
			},
			properties: common.JSON{
				"maxConnections": float64(100), // Esplicitamente usando float64 per i numeri
				"timeout":        float64(30),  // Esplicitamente usando float64 per i numeri
			},
			providerID:  providerID,
			agentTypeID: agentTypeID,
			wantErr:     false,
		},
		{
			name:        "Empty name",
			agentName:   "",
			countryCode: "US",
			attributes:  common.Attributes{},
			properties:  common.JSON{},
			providerID:  providerID,
			agentTypeID: agentTypeID,
			wantErr:     true,
		},
		{
			name:        "Invalid country code",
			agentName:   "Test Agent",
			countryCode: "USA",
			attributes:  common.Attributes{},
			properties:  common.JSON{},
			providerID:  providerID,
			agentTypeID: agentTypeID,
			wantErr:     true,
		},
		{
			name:        "Invalid attributes key",
			agentName:   "Test Agent",
			countryCode: "US",
			attributes: common.Attributes{
				"": {"value"},
			},
			properties:  common.JSON{},
			providerID:  providerID,
			agentTypeID: agentTypeID,
			wantErr:     true,
		},
		{
			name:        "Invalid properties key",
			agentName:   "Test Agent",
			countryCode: "US",
			attributes:  common.Attributes{},
			properties: common.JSON{
				"": "value",
			},
			providerID:  providerID,
			agentTypeID: agentTypeID,
			wantErr:     true,
		},
		{
			name:        "Nil UUID provider",
			agentName:   "Test Agent",
			countryCode: "US",
			attributes:  common.Attributes{},
			properties:  common.JSON{},
			providerID:  uuid.Nil,
			agentTypeID: agentTypeID,
			wantErr:     true,
		},
		{
			name:        "Nil UUID agent type",
			agentName:   "Test Agent",
			countryCode: "US",
			attributes:  common.Attributes{},
			properties:  common.JSON{},
			providerID:  providerID,
			agentTypeID: uuid.Nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.agentName, tt.countryCode, tt.attributes, tt.properties, tt.providerID, tt.agentTypeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if agent.Name != tt.agentName {
				t.Errorf("NewAgent() name = %v, want %v", agent.Name, tt.agentName)
			}
			if agent.CountryCode != tt.countryCode {
				t.Errorf("NewAgent() countryCode = %v, want %v", agent.CountryCode, tt.countryCode)
			}
			if agent.State != AgentNew {
				t.Errorf("NewAgent() state = %v, want %v", agent.State, AgentNew)
			}
			if agent.ProviderID != tt.providerID {
				t.Errorf("NewAgent() providerID = %v, want %v", agent.ProviderID, tt.providerID)
			}
			if agent.AgentTypeID != tt.agentTypeID {
				t.Errorf("NewAgent() agentTypeID = %v, want %v", agent.AgentTypeID, tt.agentTypeID)
			}

			// Test attributes conversion
			attrs, err := agent.GetAttributes()
			if err != nil {
				t.Errorf("GetAttributes() error = %v", err)
				return
			}

			if !reflect.DeepEqual(attrs, tt.attributes) {
				attrsJSON, _ := json.Marshal(attrs)
				expectedJSON, _ := json.Marshal(tt.attributes)
				t.Errorf("Attributes don't match\nGot: %s\nWant: %s", attrsJSON, expectedJSON)
			}

			// Test properties conversion
			props, err := agent.GetProperties()
			if err != nil {
				t.Errorf("GetProperties() error = %v", err)
				return
			}

			if !reflect.DeepEqual(props, tt.properties) {
				propsJSON, _ := json.Marshal(props)
				expectedJSON, _ := json.Marshal(tt.properties)
				t.Errorf("Properties don't match\nGot: %s\nWant: %s", propsJSON, expectedJSON)
			}
		})
	}
}

func TestAgentValidate(t *testing.T) {
	providerID := uuid.New()
	agentTypeID := uuid.New()

	tests := []struct {
		name    string
		agent   *Agent
		wantErr bool
	}{
		{
			name: "Valid agent",
			agent: &Agent{
				Name:        "Test Agent",
				State:       AgentNew,
				CountryCode: "US",
				ProviderID:  providerID,
				AgentTypeID: agentTypeID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			agent: &Agent{
				Name:        "",
				State:       AgentNew,
				CountryCode: "US",
				ProviderID:  providerID,
				AgentTypeID: agentTypeID,
			},
			wantErr: true,
		},
		{
			name: "Invalid state",
			agent: &Agent{
				Name:        "Test Agent",
				State:       "InvalidState",
				CountryCode: "US",
				ProviderID:  providerID,
				AgentTypeID: agentTypeID,
			},
			wantErr: true,
		},
		{
			name: "Invalid country code",
			agent: &Agent{
				Name:        "Test Agent",
				State:       AgentNew,
				CountryCode: "USA",
				ProviderID:  providerID,
				AgentTypeID: agentTypeID,
			},
			wantErr: true,
		},
		{
			name: "Nil provider ID",
			agent: &Agent{
				Name:        "Test Agent",
				State:       AgentNew,
				CountryCode: "US",
				ProviderID:  uuid.Nil,
				AgentTypeID: agentTypeID,
			},
			wantErr: true,
		},
		{
			name: "Nil agent type ID",
			agent: &Agent{
				Name:        "Test Agent",
				State:       AgentNew,
				CountryCode: "US",
				ProviderID:  providerID,
				AgentTypeID: uuid.Nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.agent.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Agent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentStateTransitions(t *testing.T) {
	agent := &Agent{
		Name:        "Test Agent",
		State:       AgentNew,
		CountryCode: "US",
		ProviderID:  uuid.New(),
		AgentTypeID: uuid.New(),
	}

	validTransitions := []struct {
		from AgentState
		to   AgentState
	}{
		{AgentNew, AgentConnected},
		{AgentConnected, AgentDisconnected},
		{AgentDisconnected, AgentConnected},
		{AgentConnected, AgentError},
		{AgentError, AgentConnected},
		{AgentConnected, AgentDisabled},
		{AgentDisabled, AgentConnected},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			agent.State = tt.from
			if err := agent.UpdateState(tt.to); err != nil {
				t.Errorf("UpdateState() error = %v", err)
			}
			if agent.State != tt.to {
				t.Errorf("UpdateState() state = %v, want %v", agent.State, tt.to)
			}
		})
	}

	t.Run("Invalid state transition", func(t *testing.T) {
		if err := agent.UpdateState("InvalidState"); err == nil {
			t.Error("UpdateState() expected error for invalid state")
		}
	})
}
