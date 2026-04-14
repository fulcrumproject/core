package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentPoolValue(t *testing.T) {
	params := CreateAgentPoolValueParams{
		Name:        "127.0.0.0",
		Value:       "127.0.0.0",
		AgentPoolID: properties.NewUUID(),
	}
	poolValue := NewAgentPoolValue(params)

	assert.NoError(t, poolValue.Validate())
	assert.Equal(t, params.Name, poolValue.Name)
	assert.Equal(t, params.Value, poolValue.Value)
	assert.Equal(t, params.AgentPoolID, poolValue.AgentPoolID)
	assert.Nil(t, poolValue.AgentID)
	assert.Nil(t, poolValue.PropertyName)
	assert.Nil(t, poolValue.AllocatedAt)
}

func TestAgentPoolValue_Validate(t *testing.T) {
	tests := []struct {
		name      string
		pool      *AgentPoolValue
		wantError bool
		errorMsg  string
	}{
		{
			name: "empty name",
			pool: &AgentPoolValue{
				Name:        "",
				Value:       "127.0.0.0",
				AgentPoolID: properties.NewUUID(),
			},
			wantError: true,
			errorMsg:  "agent pool value name is required",
		},
		{
			name: "nil value",
			pool: &AgentPoolValue{
				Name:        "127.0.0.0",
				Value:       nil,
				AgentPoolID: properties.NewUUID(),
			},
			wantError: true,
			errorMsg:  "agent pool value is required",
		},
		{
			name: "empty agent pool id",
			pool: &AgentPoolValue{
				Name:        "127.0.0.0",
				Value:       "127.0.0.0",
				AgentPoolID: properties.UUID{},
			},
			wantError: true,
			errorMsg:  "agent pool ID cannot be empty",
		},
		{
			name: "valid",
			pool: &AgentPoolValue{
				Name:        "127.0.0.0",
				Value:       "127.0.0.0",
				AgentPoolID: properties.NewUUID(),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pool.Validate()
			if tt.wantError {
				assert.ErrorContains(t, err, tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgentPoolValue_TableName(t *testing.T) {
	assert.Equal(t, "agent_pool_values", AgentPoolValue{}.TableName())
}

func TestAgentPoolValue_IsAllocated(t *testing.T) {
	tests := []struct {
		name     string
		agentID  *properties.UUID
		expected bool
	}{
		{name: "not allocated", agentID: nil, expected: false},
		{name: "allocated", agentID: helpers.UUIDPtr(properties.NewUUID()), expected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &AgentPoolValue{AgentID: tt.agentID}
			assert.Equal(t, tt.expected, v.IsAllocated())
		})
	}
}

func TestAgentPoolValue_Allocate(t *testing.T) {
	tests := []struct {
		name         string
		initial      *AgentPoolValue
		agentID      properties.UUID
		propertyName string
	}{
		{
			name:         "allocate fresh value",
			initial:      &AgentPoolValue{},
			agentID:      properties.NewUUID(),
			propertyName: "ip_address",
		},
		{
			name: "re-allocate already allocated value",
			initial: func() *AgentPoolValue {
				id := properties.NewUUID()
				now := time.Now()
				return &AgentPoolValue{
					AgentID:      &id,
					PropertyName: helpers.StringPtr("old_prop"),
					AllocatedAt:  &now,
				}
			}(),
			agentID:      properties.NewUUID(),
			propertyName: "new_prop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Allocate(tt.agentID, tt.propertyName)

			assert.Equal(t, &tt.agentID, tt.initial.AgentID)
			assert.Equal(t, helpers.StringPtr(tt.propertyName), tt.initial.PropertyName)
			assert.NotNil(t, tt.initial.AllocatedAt)
			assert.True(t, tt.initial.IsAllocated())
		})
	}
}

func TestAgentPoolValue_Release(t *testing.T) {
	tests := []struct {
		name    string
		initial *AgentPoolValue
	}{
		{
			name: "release allocated value",
			initial: func() *AgentPoolValue {
				id := properties.NewUUID()
				now := time.Now()
				return &AgentPoolValue{
					AgentID:      &id,
					PropertyName: helpers.StringPtr("ip_address"),
					AllocatedAt:  &now,
				}
			}(),
		},
		{
			name:    "release already released value",
			initial: &AgentPoolValue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Release()

			assert.Nil(t, tt.initial.AgentID)
			assert.Nil(t, tt.initial.PropertyName)
			assert.Nil(t, tt.initial.AllocatedAt)
			assert.False(t, tt.initial.IsAllocated())
		})
	}
}

