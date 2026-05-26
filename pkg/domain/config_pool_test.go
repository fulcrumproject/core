package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewConfigPool(t *testing.T) {
	config := properties.JSON{"values": []string{"a", "b"}}
	params := CreateConfigPoolParams{
		Name:            "Test Pool",
		Type:            "publicIp",
		PropertyType:    "string",
		GeneratorType:   PoolGeneratorList,
		GeneratorConfig: &config,
	}

	pool := NewConfigPool(params)

	assert.Equal(t, "Test Pool", pool.Name)
	assert.Equal(t, "publicIp", pool.Type)
	assert.Equal(t, "string", pool.PropertyType)
	assert.Equal(t, PoolGeneratorList, pool.GeneratorType)
	assert.Equal(t, &config, pool.GeneratorConfig)
}

func TestConfigPool_Validate(t *testing.T) {
	tests := []struct {
		name      string
		pool      *ConfigPool
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid list pool",
			pool: &ConfigPool{
				Name:          "Valid Pool",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			wantError: false,
		},
		{
			name: "invalid subnet generator type",
			pool: &ConfigPool{
				Name:          "Valid Subnet Pool",
				Type:          "internalIp",
				PropertyType:  "json",
				GeneratorType: PoolGeneratorSubnet,
			},
			wantError: true,
			errorMsg:  "invalid generator type for config pool",
		},
		{
			name: "empty name",
			pool: &ConfigPool{
				Name:          "",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			wantError: true,
			errorMsg:  "config pool name cannot be empty",
		},
		{
			name: "empty type",
			pool: &ConfigPool{
				Name:          "Test Pool",
				Type:          "",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			},
			wantError: true,
			errorMsg:  "config pool type cannot be empty",
		},
		{
			name: "invalid property type",
			pool: &ConfigPool{
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
			pool: &ConfigPool{
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
			pool: &ConfigPool{
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

func TestConfigPool_TableName(t *testing.T) {
	pool := &ConfigPool{}
	assert.Equal(t, "config_pools", pool.TableName())
}

func TestConfigPool_Update(t *testing.T) {
	tests := []struct {
		name           string
		params         UpdateConfigPoolParams
		expectedName   string
		expectedConfig *properties.JSON
	}{
		{
			name:           "update name only",
			params:         UpdateConfigPoolParams{Name: helpers.StringPtr("New Name")},
			expectedName:   "New Name",
			expectedConfig: nil,
		},
		{
			name:           "update config only",
			params:         UpdateConfigPoolParams{GeneratorConfig: &properties.JSON{"key": "val"}},
			expectedName:   "Original",
			expectedConfig: &properties.JSON{"key": "val"},
		},
		{
			name:           "update both",
			params:         UpdateConfigPoolParams{Name: helpers.StringPtr("Updated"), GeneratorConfig: &properties.JSON{"a": "b"}},
			expectedName:   "Updated",
			expectedConfig: &properties.JSON{"a": "b"},
		},
		{
			name:           "update nothing",
			params:         UpdateConfigPoolParams{},
			expectedName:   "Original",
			expectedConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &ConfigPool{Name: "Original"}
			pool.Update(tt.params)
			assert.Equal(t, tt.expectedName, pool.Name)
			assert.Equal(t, tt.expectedConfig, pool.GeneratorConfig)
		})
	}
}

func TestConfigPoolCommander_Create(t *testing.T) {
	participantID := properties.UUID(uuid.New())

	baseParams := func() CreateConfigPoolParams {
		return CreateConfigPoolParams{
			Name:          "Public IP",
			Type:          "publicIp",
			PropertyType:  "string",
			GeneratorType: PoolGeneratorList,
		}
	}

	tests := []struct {
		name              string
		params            CreateConfigPoolParams
		existingSameScope bool // FindByTypeAndParticipant returns an existing row
		wantErr           bool
		errContains       string
		assertOnCreate    func(t *testing.T, p *ConfigPool)
	}{
		{
			name:   "creates global pool when ParticipantID nil",
			params: baseParams(),
			assertOnCreate: func(t *testing.T, p *ConfigPool) {
				assert.Nil(t, p.ParticipantID, "global pool must keep ParticipantID nil")
			},
		},
		{
			name: "creates participant-owned pool when ParticipantID set",
			params: func() CreateConfigPoolParams {
				p := baseParams()
				p.ParticipantID = &participantID
				return p
			}(),
			assertOnCreate: func(t *testing.T, p *ConfigPool) {
				require.NotNil(t, p.ParticipantID)
				assert.Equal(t, participantID, *p.ParticipantID)
			},
		},
		{
			name:              "rejects duplicate within global scope",
			params:            baseParams(),
			existingSameScope: true,
			wantErr:           true,
			errContains:       "already exists",
		},
		{
			name: "rejects duplicate within participant scope",
			params: func() CreateConfigPoolParams {
				p := baseParams()
				p.ParticipantID = &participantID
				return p
			}(),
			existingSameScope: true,
			wantErr:           true,
			errContains:       "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := setupMockStore(t)

			poolRepo := NewMockConfigPoolRepository(t)
			if tt.existingSameScope {
				poolRepo.On("FindByTypeAndParticipant", mock.Anything, tt.params.Type, tt.params.ParticipantID).
					Return(&ConfigPool{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}}, nil).Once()
			} else {
				poolRepo.On("FindByTypeAndParticipant", mock.Anything, tt.params.Type, tt.params.ParticipantID).
					Return(nil, NewNotFoundErrorf("not found")).Once()
				poolRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *ConfigPool) bool {
					if tt.params.ParticipantID == nil {
						return p.ParticipantID == nil
					}
					return p.ParticipantID != nil && *p.ParticipantID == *tt.params.ParticipantID
				})).Return(nil).Once()
			}
			ms.On("ConfigPoolRepo").Return(poolRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			if !tt.wantErr {
				eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
			}
			ms.On("EventRepo").Return(eventRepo).Maybe()

			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
			cmd := NewConfigPoolCommander(ms)

			pool, err := cmd.Create(ctx, tt.params)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, pool)
			if tt.assertOnCreate != nil {
				tt.assertOnCreate(t, pool)
			}
		})
	}
}

func TestConfigPoolCommander_Delete(t *testing.T) {
	poolID := properties.UUID(uuid.New())

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
			count:       3,
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

			poolRepo := NewMockConfigPoolRepository(t)
			poolRepo.On("Get", mock.Anything, poolID).Return(&ConfigPool{
				BaseEntity:    BaseEntity{ID: poolID},
				Name:          "p",
				Type:          "publicIp",
				PropertyType:  "string",
				GeneratorType: PoolGeneratorList,
			}, nil).Once()
			if tt.expectDel {
				poolRepo.On("Delete", mock.Anything, poolID).Return(nil).Once()
			}
			ms.On("ConfigPoolRepo").Return(poolRepo).Maybe()

			valueRepo := NewMockConfigPoolValueRepository(t)
			valueRepo.On("CountByPool", mock.Anything, poolID).Return(tt.count, tt.countErr).Once()
			ms.On("ConfigPoolValueRepo").Return(valueRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			if tt.expectEvent {
				eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
			}
			ms.On("EventRepo").Return(eventRepo).Maybe()

			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
			cmd := NewConfigPoolCommander(ms)
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
