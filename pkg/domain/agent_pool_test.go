package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentPool(t *testing.T) {
	config := properties.JSON{"values": []string{"a", "b"}}
	params := CreateAgentPoolParams{
		Name:            "Test Pool",
		Type:            "publicIp",
		PropertyType:    "string",
		GeneratorType:   PoolGeneratorList,
		GeneratorConfig: &config,
	}

	pool := NewAgentPool(params)

	assert.Equal(t, "Test Pool", pool.Name)
	assert.Equal(t, "publicIp", pool.Type)
	assert.Equal(t, "string", pool.PropertyType)
	assert.Equal(t, PoolGeneratorList, pool.GeneratorType)
	assert.Equal(t, &config, pool.GeneratorConfig)
}

func TestAgentPool_Validate(t *testing.T) {
	tests := []struct {
		name      string
		pool      *AgentPool
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid list pool",
			pool: &AgentPool{
				Name:          "Valid Pool",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			wantError: false,
		},
		{
			name: "invalid subnet generator type",
			pool: &AgentPool{
				Name:          "Valid Subnet Pool",
				Type:          "internalIp",
				PropertyType:  "json",
				GeneratorType: PoolGeneratorSubnet,
			},
			wantError: true,
			errorMsg:  "invalid generator type for agent pool",
		},
		{
			name: "empty name",
			pool: &AgentPool{
				Name:          "",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			wantError: true,
			errorMsg:  "agent pool name cannot be empty",
		},
		{
			name: "empty type",
			pool: &AgentPool{
				Name:          "Test Pool",
				Type:          "",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			wantError: true,
			errorMsg:  "agent pool type cannot be empty",
		},
		{
			name: "invalid property type",
			pool: &AgentPool{
				Name:          "Test Pool",
				Type:          "publicIp",
				PropertyType:  "invalid",
				GeneratorType: PoolGeneratorList,
			},
			wantError: true,
			errorMsg:  "invalid property type",
		},
		{
			name: "empty property type",
			pool: &AgentPool{
				Name:          "Test Pool",
				Type:          "publicIp",
				PropertyType:  "",
				GeneratorType: PoolGeneratorList,
			},
			wantError: true,
			errorMsg:  "invalid property type",
		},
		{
			name: "invalid generator type",
			pool: &AgentPool{
				Name:          "Test Pool",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: "invalid",
			},
			wantError: true,
			errorMsg:  "invalid generator type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pool.Validate()
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAgentPool_TableName(t *testing.T) {
	pool := &AgentPool{}
	assert.Equal(t, "agent_pools", pool.TableName())
}

func TestAgentPool_Update(t *testing.T) {
	tests := []struct {
		name           string
		params         UpdateAgentPoolParams
		expectedName   string
		expectedConfig *properties.JSON
	}{
		{
			name:           "update name only",
			params:         UpdateAgentPoolParams{Name: strPtr("New Name")},
			expectedName:   "New Name",
			expectedConfig: nil,
		},
		{
			name:           "update config only",
			params:         UpdateAgentPoolParams{GeneratorConfig: &properties.JSON{"key": "val"}},
			expectedName:   "Original",
			expectedConfig: &properties.JSON{"key": "val"},
		},
		{
			name:           "update both",
			params:         UpdateAgentPoolParams{Name: strPtr("Updated"), GeneratorConfig: &properties.JSON{"a": "b"}},
			expectedName:   "Updated",
			expectedConfig: &properties.JSON{"a": "b"},
		},
		{
			name:           "update nothing",
			params:         UpdateAgentPoolParams{},
			expectedName:   "Original",
			expectedConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &AgentPool{Name: "Original"}
			pool.Update(tt.params)
			assert.Equal(t, tt.expectedName, pool.Name)
			assert.Equal(t, tt.expectedConfig, pool.GeneratorConfig)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
