package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigPoolValue(t *testing.T) {
	params := CreateConfigPoolValueParams{
		Name:         "127.0.0.0",
		Value:        "127.0.0.0",
		ConfigPoolID: properties.NewUUID(),
	}
	poolValue := NewConfigPoolValue(params)

	assert.NoError(t, poolValue.Validate())
	assert.Equal(t, params.Name, poolValue.Name)
	assert.Equal(t, params.Value, poolValue.Value)
	assert.Equal(t, params.ConfigPoolID, poolValue.ConfigPoolID)
	assert.Nil(t, poolValue.AgentID)
	assert.Nil(t, poolValue.PropertyName)
	assert.Nil(t, poolValue.AllocatedAt)
}

func TestConfigPoolValue_Validate(t *testing.T) {
	tests := []struct {
		name      string
		pool      *ConfigPoolValue
		wantError bool
		errorMsg  string
	}{
		{
			name: "empty name",
			pool: &ConfigPoolValue{
				Name:         "",
				Value:        "127.0.0.0",
				ConfigPoolID: properties.NewUUID(),
			},
			wantError: true,
			errorMsg:  "config pool value name is required",
		},
		{
			name: "nil value",
			pool: &ConfigPoolValue{
				Name:         "127.0.0.0",
				Value:        nil,
				ConfigPoolID: properties.NewUUID(),
			},
			wantError: true,
			errorMsg:  "config pool value is required",
		},
		{
			name: "empty config pool id",
			pool: &ConfigPoolValue{
				Name:         "127.0.0.0",
				Value:        "127.0.0.0",
				ConfigPoolID: properties.UUID{},
			},
			wantError: true,
			errorMsg:  "config pool ID cannot be empty",
		},
		{
			name: "valid",
			pool: &ConfigPoolValue{
				Name:         "127.0.0.0",
				Value:        "127.0.0.0",
				ConfigPoolID: properties.NewUUID(),
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

func TestConfigPoolValue_TableName(t *testing.T) {
	assert.Equal(t, "config_pool_values", ConfigPoolValue{}.TableName())
}

func TestConfigPoolValue_IsAllocated(t *testing.T) {
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
			v := &ConfigPoolValue{AgentID: tt.agentID}
			assert.Equal(t, tt.expected, v.IsAllocated())
		})
	}
}

func TestConfigPoolValue_Allocate(t *testing.T) {
	tests := []struct {
		name         string
		initial      *ConfigPoolValue
		agentID      properties.UUID
		propertyName string
	}{
		{
			name:         "allocate fresh value",
			initial:      &ConfigPoolValue{},
			agentID:      properties.NewUUID(),
			propertyName: "ip_address",
		},
		{
			name: "re-allocate already allocated value",
			initial: func() *ConfigPoolValue {
				id := properties.NewUUID()
				now := time.Now()
				return &ConfigPoolValue{
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

func TestConfigPoolValue_Release(t *testing.T) {
	tests := []struct {
		name    string
		initial *ConfigPoolValue
	}{
		{
			name: "release allocated value",
			initial: func() *ConfigPoolValue {
				id := properties.NewUUID()
				now := time.Now()
				return &ConfigPoolValue{
					AgentID:      &id,
					PropertyName: helpers.StringPtr("ip_address"),
					AllocatedAt:  &now,
				}
			}(),
		},
		{
			name:    "release already released value",
			initial: &ConfigPoolValue{},
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
