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
		name             string
		agentID          *properties.UUID
		infrastructureID *properties.UUID
		expected         bool
	}{
		{name: "not allocated", expected: false},
		{name: "allocated to agent", agentID: helpers.UUIDPtr(properties.NewUUID()), expected: true},
		{name: "allocated to infrastructure", infrastructureID: helpers.UUIDPtr(properties.NewUUID()), expected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &ConfigPoolValue{AgentID: tt.agentID, InfrastructureID: tt.infrastructureID}
			assert.Equal(t, tt.expected, v.IsAllocated())
		})
	}
}

func TestConfigPoolValue_Allocate(t *testing.T) {
	tests := []struct {
		name         string
		initial      *ConfigPoolValue
		entityType   ConfigPoolValueEntityType
		entityID     properties.UUID
		propertyName string
	}{
		{
			name:         "allocate fresh value to agent",
			initial:      &ConfigPoolValue{},
			entityType:   ConfigPoolValueEntityTypeAgent,
			entityID:     properties.NewUUID(),
			propertyName: "ip_address",
		},
		{
			name:         "allocate fresh value to infrastructure",
			initial:      &ConfigPoolValue{},
			entityType:   ConfigPoolValueEntityTypeInfrastructure,
			entityID:     properties.NewUUID(),
			propertyName: "ptp",
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
			entityType:   ConfigPoolValueEntityTypeAgent,
			entityID:     properties.NewUUID(),
			propertyName: "new_prop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Allocate(tt.entityType, tt.entityID, tt.propertyName)

			switch tt.entityType {
			case ConfigPoolValueEntityTypeAgent:
				assert.Equal(t, &tt.entityID, tt.initial.AgentID)
				assert.Nil(t, tt.initial.InfrastructureID)
			case ConfigPoolValueEntityTypeInfrastructure:
				assert.Equal(t, &tt.entityID, tt.initial.InfrastructureID)
				assert.Nil(t, tt.initial.AgentID)
			}
			assert.Equal(t, helpers.StringPtr(tt.propertyName), tt.initial.PropertyName)
			assert.NotNil(t, tt.initial.AllocatedAt)
			assert.True(t, tt.initial.IsAllocated())
		})
	}
}

func TestConfigPoolValueCommander_Create(t *testing.T) {
	participantID := properties.UUID(uuid.New())
	poolID := properties.UUID(uuid.New())

	tests := []struct {
		name              string
		pool              *ConfigPool
		poolGetErr        error
		wantErr           bool
		errContains       string
		expectParticipant *properties.UUID
	}{
		{
			name: "stamps nil ParticipantID from global pool",
			pool: &ConfigPool{
				BaseEntity:    BaseEntity{ID: poolID},
				Name:          "Global",
				Type:          "ipv4",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			expectParticipant: nil,
		},
		{
			name: "stamps ParticipantID from participant-scoped pool",
			pool: &ConfigPool{
				BaseEntity:    BaseEntity{ID: poolID},
				Name:          "Scoped",
				Type:          "ipv4",
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
			errContains: "config pool with id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			poolRepo := NewMockConfigPoolRepository(t)
			if tt.poolGetErr != nil {
				poolRepo.On("Get", mock.Anything, poolID).Return(nil, tt.poolGetErr).Once()
			} else {
				poolRepo.On("Get", mock.Anything, poolID).Return(tt.pool, nil).Once()
			}
			ms.On("ConfigPoolRepo").Return(poolRepo).Maybe()

			valueRepo := NewMockConfigPoolValueRepository(t)
			if !tt.wantErr {
				valueRepo.On("Create", mock.Anything, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					if tt.expectParticipant == nil {
						return v.ParticipantID == nil
					}
					return v.ParticipantID != nil && *v.ParticipantID == *tt.expectParticipant
				})).Return(nil).Once()
			}
			ms.On("ConfigPoolValueRepo").Return(valueRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			if !tt.wantErr {
				eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
			}
			ms.On("EventRepo").Return(eventRepo).Maybe()

			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
			cmd := NewConfigPoolValueCommander(ms)

			value, err := cmd.Create(ctx, CreateConfigPoolValueParams{
				Name:         "10.0.0.1",
				Value:        "10.0.0.1",
				ConfigPoolID: poolID,
			})

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, value)
			if tt.expectParticipant == nil {
				assert.Nil(t, value.ParticipantID)
			} else {
				require.NotNil(t, value.ParticipantID)
				assert.Equal(t, *tt.expectParticipant, *value.ParticipantID)
			}
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
			name: "release infrastructure-allocated value",
			initial: func() *ConfigPoolValue {
				id := properties.NewUUID()
				now := time.Now()
				return &ConfigPoolValue{
					InfrastructureID: &id,
					PropertyName:     helpers.StringPtr("ptp"),
					AllocatedAt:      &now,
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
			assert.Nil(t, tt.initial.InfrastructureID)
			assert.Nil(t, tt.initial.PropertyName)
			assert.Nil(t, tt.initial.AllocatedAt)
			assert.False(t, tt.initial.IsAllocated())
		})
	}
}
