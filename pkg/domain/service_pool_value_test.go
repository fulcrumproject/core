// ServicePoolValue entity tests
package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServicePoolValue(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	value := properties.JSON{"ip": "203.0.113.10"}

	params := CreateServicePoolValueParams{
		ServicePoolID: poolID,
		Name:          "IP 1",
		Value:         value,
	}

	poolValue := NewServicePoolValue(params)

	assert.Equal(t, "IP 1", poolValue.Name)
	assert.Equal(t, value, poolValue.Value)
	assert.Equal(t, poolID, poolValue.ServicePoolID)
	assert.Nil(t, poolValue.ServiceID)
	assert.Nil(t, poolValue.PropertyName)
	assert.Nil(t, poolValue.AllocatedAt)
}

func TestServicePoolValue_Validate(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	value := properties.JSON{"ip": "203.0.113.10"}

	tests := []struct {
		name      string
		poolValue *ServicePoolValue
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid pool value",
			poolValue: &ServicePoolValue{
				Name:          "Valid Value",
				Value:         value,
				ServicePoolID: poolID,
			},
			wantError: false,
		},
		{
			name: "empty name",
			poolValue: &ServicePoolValue{
				Name:          "",
				Value:         value,
				ServicePoolID: poolID,
			},
			wantError: true,
			errorMsg:  "pool value name cannot be empty",
		},
		{
			name: "nil value",
			poolValue: &ServicePoolValue{
				Name:          "Test Value",
				Value:         nil,
				ServicePoolID: poolID,
			},
			wantError: true,
			errorMsg:  "pool value cannot be nil",
		},
		{
			name: "empty pool ID",
			poolValue: &ServicePoolValue{
				Name:          "Test Value",
				Value:         value,
				ServicePoolID: properties.UUID{},
			},
			wantError: true,
			errorMsg:  "service pool ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.poolValue.Validate()
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServicePoolValue_IsAllocated(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	now := time.Now()

	tests := []struct {
		name      string
		poolValue *ServicePoolValue
		want      bool
	}{
		{
			name: "not allocated",
			poolValue: &ServicePoolValue{
				ServicePoolID: poolID,
				ServiceID:     nil,
				PropertyName:  nil,
				AllocatedAt:   nil,
			},
			want: false,
		},
		{
			name: "allocated",
			poolValue: &ServicePoolValue{
				ServicePoolID: poolID,
				ServiceID:     &serviceID,
				PropertyName:  stringPtr("publicIp"),
				AllocatedAt:   &now,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.poolValue.IsAllocated())
		})
	}
}

func TestServicePoolValue_Allocate(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	value := properties.JSON{"ip": "203.0.113.10"}

	poolValue := &ServicePoolValue{
		ServicePoolID: poolID,
		Name:          "IP 1",
		Value:         value,
		ServiceID:     nil,
		PropertyName:  nil,
		AllocatedAt:   nil,
	}

	assert.False(t, poolValue.IsAllocated())

	poolValue.Allocate(serviceID, "publicIp")

	assert.True(t, poolValue.IsAllocated())
	assert.Equal(t, serviceID, *poolValue.ServiceID)
	assert.Equal(t, "publicIp", *poolValue.PropertyName)
	assert.NotNil(t, poolValue.AllocatedAt)
}

func TestServicePoolValue_Release(t *testing.T) {
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	now := time.Now()
	value := properties.JSON{"ip": "203.0.113.10"}

	poolValue := &ServicePoolValue{
		ServicePoolID: poolID,
		Name:          "IP 1",
		Value:         value,
		ServiceID:     &serviceID,
		PropertyName:  stringPtr("publicIp"),
		AllocatedAt:   &now,
	}

	assert.True(t, poolValue.IsAllocated())

	poolValue.Release()

	assert.False(t, poolValue.IsAllocated())
	assert.Nil(t, poolValue.ServiceID)
	assert.Nil(t, poolValue.PropertyName)
	assert.Nil(t, poolValue.AllocatedAt)
}

func TestServicePoolValue_TableName(t *testing.T) {
	poolValue := &ServicePoolValue{}
	assert.Equal(t, "service_pool_values", poolValue.TableName())
}

