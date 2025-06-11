package domain

import (
	"testing"
	"time"

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
