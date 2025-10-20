// ServicePool entity tests
package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServicePool(t *testing.T) {
	poolSetID := properties.UUID(uuid.New())

	params := CreateServicePoolParams{
		ServicePoolSetID: poolSetID,
		Name:             "Test Pool",
		Type:             "publicIp",
		GeneratorType:    PoolGeneratorList,
	}

	pool := NewServicePool(params)

	assert.Equal(t, "Test Pool", pool.Name)
	assert.Equal(t, "publicIp", pool.Type)
	assert.Equal(t, PoolGeneratorList, pool.GeneratorType)
	assert.Equal(t, poolSetID, pool.ServicePoolSetID)
}

func TestServicePool_Validate(t *testing.T) {
	poolSetID := properties.UUID(uuid.New())

	tests := []struct {
		name      string
		pool      *ServicePool
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid list pool",
			pool: &ServicePool{
				Name:             "Valid Pool",
				Type:             "publicIp",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			},
			wantError: false,
		},
		{
			name: "valid subnet pool",
			pool: &ServicePool{
				Name:             "Valid Subnet Pool",
				Type:             "internalIp",
				GeneratorType:    PoolGeneratorSubnet,
				ServicePoolSetID: poolSetID,
			},
			wantError: false,
		},
		{
			name: "empty name",
			pool: &ServicePool{
				Name:             "",
				Type:             "publicIp",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			},
			wantError: true,
			errorMsg:  "pool name cannot be empty",
		},
		{
			name: "empty type",
			pool: &ServicePool{
				Name:             "Test Pool",
				Type:             "",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			},
			wantError: true,
			errorMsg:  "pool type cannot be empty",
		},
		{
			name: "invalid generator type",
			pool: &ServicePool{
				Name:             "Test Pool",
				Type:             "publicIp",
				GeneratorType:    "invalid",
				ServicePoolSetID: poolSetID,
			},
			wantError: true,
			errorMsg:  "invalid generator type",
		},
		{
			name: "empty pool set ID",
			pool: &ServicePool{
				Name:             "Test Pool",
				Type:             "publicIp",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: properties.UUID{},
			},
			wantError: true,
			errorMsg:  "service pool set ID cannot be empty",
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

func TestServicePool_TableName(t *testing.T) {
	pool := &ServicePool{}
	assert.Equal(t, "service_pools", pool.TableName())
}

func TestPoolGeneratorType_Validate(t *testing.T) {
	tests := []struct {
		name      string
		genType   PoolGeneratorType
		wantError bool
	}{
		{
			name:      "valid list type",
			genType:   PoolGeneratorList,
			wantError: false,
		},
		{
			name:      "valid subnet type",
			genType:   PoolGeneratorSubnet,
			wantError: false,
		},
		{
			name:      "invalid type",
			genType:   PoolGeneratorType("invalid"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.genType.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

