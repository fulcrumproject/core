// ServicePoolValue entity tests
package domain

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
				PropertyName:  helpers.StringPtr("publicIp"),
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
		PropertyName:  helpers.StringPtr("publicIp"),
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

func TestServicePoolValueCommander_Create(t *testing.T) {
	participantID := properties.UUID(uuid.New())
	poolID := properties.UUID(uuid.New())

	tests := []struct {
		name              string
		pool              *ServicePool
		poolGetErr        error
		wantErr           bool
		errContains       string
		expectParticipant *properties.UUID
	}{
		{
			name: "stamps ParticipantID from parent pool",
			pool: &ServicePool{
				BaseEntity:    BaseEntity{ID: poolID},
				Name:          "Scoped",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
				ParticipantID: &participantID,
			},
			expectParticipant: &participantID,
		},
		{
			name:        "returns NotFound when parent pool missing",
			poolGetErr:  NewNotFoundErrorf("missing"),
			wantErr:     true,
			errContains: "service pool with id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			poolRepo := NewMockServicePoolRepository(t)
			if tt.poolGetErr != nil {
				poolRepo.On("Get", mock.Anything, poolID).Return(nil, tt.poolGetErr).Once()
			} else {
				poolRepo.On("Get", mock.Anything, poolID).Return(tt.pool, nil).Once()
			}
			ms.On("ServicePoolRepo").Return(poolRepo).Maybe()

			valueRepo := NewMockServicePoolValueRepository(t)
			if !tt.wantErr {
				valueRepo.On("Create", mock.Anything, mock.MatchedBy(func(v *ServicePoolValue) bool {
					if tt.expectParticipant == nil {
						return v.ParticipantID == nil
					}
					return v.ParticipantID != nil && *v.ParticipantID == *tt.expectParticipant
				})).Return(nil).Once()
			}
			ms.On("ServicePoolValueRepo").Return(valueRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			if !tt.wantErr {
				eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
			}
			ms.On("EventRepo").Return(eventRepo).Maybe()

			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
			cmd := NewServicePoolValueCommander(ms)

			value, err := cmd.Create(ctx, CreateServicePoolValueParams{
				Name:          "203.0.113.10",
				Value:         "203.0.113.10",
				ServicePoolID: poolID,
			})

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, value)
			require.NotNil(t, value.ParticipantID)
			assert.Equal(t, *tt.expectParticipant, *value.ParticipantID)
		})
	}
}
