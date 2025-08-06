package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAgentStatus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		status  AgentStatus
		wantErr bool
	}{
		{
			name:    "New status",
			status:  AgentNew,
			wantErr: false,
		},
		{
			name:    "Connected status",
			status:  AgentConnected,
			wantErr: false,
		},
		{
			name:    "Disconnected status",
			status:  AgentDisconnected,
			wantErr: false,
		},
		{
			name:    "Error status",
			status:  AgentError,
			wantErr: false,
		},
		{
			name:    "Disabled status",
			status:  AgentDisabled,
			wantErr: false,
		},
		{
			name:    "Invalid status",
			status:  "InvalidStatus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseAgentStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    AgentStatus
		wantErr bool
	}{
		{
			name:    "Parse New status",
			value:   "New",
			want:    AgentNew,
			wantErr: false,
		},
		{
			name:    "Parse Connected status",
			value:   "Connected",
			want:    AgentConnected,
			wantErr: false,
		},
		{
			name:    "Parse Disconnected status",
			value:   "Disconnected",
			want:    AgentDisconnected,
			wantErr: false,
		},
		{
			name:    "Parse Error status",
			value:   "Error",
			want:    AgentError,
			wantErr: false,
		},
		{
			name:    "Parse Disabled status",
			value:   "Disabled",
			want:    AgentDisabled,
			wantErr: false,
		},
		{
			name:    "Parse invalid status",
			value:   "InvalidStatus",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAgentStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAgent_TableName(t *testing.T) {
	agent := Agent{}
	assert.Equal(t, "agents", agent.TableName())
}

func TestAgent_Validate(t *testing.T) {
	validID := uuid.New()
	validTime := time.Now()

	tests := []struct {
		name    string
		agent   *Agent
		wantErr bool
	}{
		{
			name: "Valid agent",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			agent: &Agent{
				Name:             "",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Invalid status",
			agent: &Agent{
				Name:             "test-agent",
				Status:           "InvalidStatus",
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Zero time",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: time.Time{},
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Empty agent type ID",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      uuid.Nil,
				ProviderID:       validID,
			},
			wantErr: true,
		},
		{
			name: "Empty participant ID",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       uuid.Nil,
			},
			wantErr: true,
		},
		{
			name: "Valid agent",
			agent: &Agent{
				Name:             "test-agent",
				Status:           AgentConnected,
				LastStatusUpdate: validTime,
				AgentTypeID:      validID,
				ProviderID:       validID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgent_UpdateStatus(t *testing.T) {
	now := time.Now()
	agent := &Agent{
		Status:           AgentDisconnected,
		LastStatusUpdate: now.Add(-time.Hour), // Set to 1 hour ago
	}

	agent.UpdateStatus(AgentConnected)

	assert.Equal(t, AgentConnected, agent.Status)
	assert.True(t, agent.LastStatusUpdate.After(now), "LastStatusUpdate should be updated to a newer time")
}

func TestAgent_UpdateHeartbeat(t *testing.T) {
	now := time.Now()
	agent := &Agent{
		Status:           AgentConnected,
		LastStatusUpdate: now.Add(-time.Hour), // Set to 1 hour ago
	}

	agent.UpdateHeartbeat()

	assert.Equal(t, AgentConnected, agent.Status, "Status should not change")
	assert.True(t, agent.LastStatusUpdate.After(now), "LastStatusUpdate should be updated to a newer time")
}

func TestAgent_RegisterMetadata(t *testing.T) {
	agent := &Agent{
		Name: "original-name",
	}

	// Test updating the name
	newName := "new-name"
	agent.RegisterMetadata(&newName)

	assert.Equal(t, newName, agent.Name)

	// Test updating to another name
	newerName := "newer-name"
	agent.RegisterMetadata(&newerName)

	assert.Equal(t, newerName, agent.Name)
}

func TestNewAgent(t *testing.T) {
	validID := uuid.New()
	agentTypeID := uuid.New()
	tags := []string{"tag1", "tag2"}

	tests := []struct {
		name   string
		params CreateAgentParams
	}{
		{
			name: "Agent without configuration",
			params: CreateAgentParams{
				Name:        "test-agent",
				ProviderID:  validID,
				AgentTypeID: agentTypeID,
				Tags:        tags,
			},
		},
		{
			name: "Agent with configuration",
			params: CreateAgentParams{
				Name:        "test-agent-with-config",
				ProviderID:  validID,
				AgentTypeID: agentTypeID,
				Tags:        tags,
				Configuration: &properties.JSON{
					"timeout":     30,
					"retries":     3,
					"environment": "production",
				},
			},
		},
		{
			name: "Agent with empty configuration",
			params: CreateAgentParams{
				Name:          "test-agent-empty-config",
				ProviderID:    validID,
				AgentTypeID:   agentTypeID,
				Tags:          tags,
				Configuration: &properties.JSON{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent(tt.params)

			assert.Equal(t, tt.params.Name, agent.Name)
			assert.Equal(t, AgentDisconnected, agent.Status)
			assert.Equal(t, tt.params.ProviderID, agent.ProviderID)
			assert.Equal(t, tt.params.AgentTypeID, agent.AgentTypeID)
			assert.Equal(t, tt.params.Tags, []string(agent.Tags))
			assert.Equal(t, tt.params.Configuration, agent.Configuration)
			assert.False(t, agent.LastStatusUpdate.IsZero())
		})
	}
}

func TestAgent_Update(t *testing.T) {
	agent := &Agent{
		Name: "original-name",
		Tags: []string{"tag1", "tag2"},
		Configuration: &properties.JSON{
			"timeout": 30,
		},
	}

	tests := []struct {
		name          string
		updateName    *string
		updateTags    *[]string
		updateConfig  *properties.JSON
		expectUpdated bool
	}{
		{
			name:          "Update name only",
			updateName:    helpers.StringPtr("new-name"),
			expectUpdated: true,
		},
		{
			name:          "Update tags only",
			updateTags:    &[]string{"new-tag1", "new-tag2"},
			expectUpdated: true,
		},
		{
			name: "Update configuration only",
			updateConfig: &properties.JSON{
				"timeout":     60,
				"retries":     5,
				"environment": "staging",
			},
			expectUpdated: true,
		},
		{
			name:       "Update all fields",
			updateName: helpers.StringPtr("updated-name"),
			updateTags: &[]string{"updated-tag"},
			updateConfig: &properties.JSON{
				"timeout": 120,
			},
			expectUpdated: true,
		},
		{
			name:          "Update with nil configuration",
			updateConfig:  &properties.JSON{},
			expectUpdated: true,
		},
		{
			name:          "No updates",
			expectUpdated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset agent to initial state
			agent.Name = "original-name"
			agent.Tags = []string{"tag1", "tag2"}
			agent.Configuration = &properties.JSON{
				"timeout": 30,
			}

			updated := agent.Update(tt.updateName, tt.updateTags, tt.updateConfig)
			assert.Equal(t, tt.expectUpdated, updated)

			if tt.updateName != nil {
				assert.Equal(t, *tt.updateName, agent.Name)
			}
			if tt.updateTags != nil {
				assert.Equal(t, *tt.updateTags, []string(agent.Tags))
			}
			if tt.updateConfig != nil {
				assert.Equal(t, tt.updateConfig, agent.Configuration)
			}
		})
	}
}
