// ServicePool entity tests
package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewServicePool(t *testing.T) {
	poolSetID := properties.UUID(uuid.New())

	params := CreateServicePoolParams{
		ServicePoolSetID: poolSetID,
		Name:             "Test Pool",
		Type:             "publicIp",
		PropertyType:     "string",
		GeneratorType:    PoolGeneratorList,
	}

	pool := NewServicePool(params)

	assert.Equal(t, "Test Pool", pool.Name)
	assert.Equal(t, "publicIp", pool.Type)
	assert.Equal(t, "string", pool.PropertyType)
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
				PropertyType:     "string",
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
				PropertyType:     "json",
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
				PropertyType:     "string",
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
				PropertyType:     "string",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			},
			wantError: true,
			errorMsg:  "pool type cannot be empty",
		},
		{
			name: "invalid property type",
			pool: &ServicePool{
				Name:             "Test Pool",
				Type:             "publicIp",
				PropertyType:     "invalid",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			},
			wantError: true,
			errorMsg:  "invalid property type",
		},
		{
			name: "empty property type",
			pool: &ServicePool{
				Name:             "Test Pool",
				Type:             "publicIp",
				PropertyType:     "",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			},
			wantError: true,
			errorMsg:  "invalid property type",
		},
		{
			name: "invalid generator type",
			pool: &ServicePool{
				Name:             "Test Pool",
				Type:             "publicIp",
				PropertyType:     "string",
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
				PropertyType:     "string",
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

func TestServicePoolCommander_Delete(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	poolSetID := properties.UUID(uuid.New())

	tests := []struct {
		name        string
		count       int64
		countErr    error
		expectEvent bool
		expectDel   bool
		wantErr     bool
		errContains string
		wantInvalid bool
	}{
		{
			name:        "success when no values",
			count:       0,
			expectEvent: true,
			expectDel:   true,
		},
		{
			name:        "refuses delete when values exist",
			count:       2,
			wantErr:     true,
			errContains: "dependent value(s) exist",
			wantInvalid: true,
		},
		{
			name:        "propagates count error",
			countErr:    errors.New("db boom"),
			wantErr:     true,
			errContains: "db boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			poolRepo := NewMockServicePoolRepository(t)
			poolRepo.On("Get", mock.Anything, poolID).Return(&ServicePool{
				BaseEntity:       BaseEntity{ID: poolID},
				Name:             "p",
				Type:             "publicIp",
				PropertyType:     "string",
				GeneratorType:    PoolGeneratorList,
				ServicePoolSetID: poolSetID,
			}, nil).Once()
			if tt.expectDel {
				poolRepo.On("Delete", mock.Anything, poolID).Return(nil).Once()
			}
			ms.On("ServicePoolRepo").Return(poolRepo).Maybe()

			valueRepo := NewMockServicePoolValueRepository(t)
			valueRepo.On("CountByPool", mock.Anything, poolID).Return(tt.count, tt.countErr).Once()
			ms.On("ServicePoolValueRepo").Return(valueRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			if tt.expectEvent {
				eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
			}
			ms.On("EventRepo").Return(eventRepo).Maybe()

			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
			cmd := NewServicePoolCommander(ms)
			err := cmd.Delete(ctx, poolID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				if tt.wantInvalid {
					assert.ErrorAs(t, err, &InvalidInputError{})
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
