// Subnet generator tests
package domain

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSubnetGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	propertyName := "ipAddress"

	tests := []struct {
		name            string
		generatorConfig properties.JSON
		setupMock       func(*MockServicePoolValueRepository)
		expectedValue   string
		expectErr       bool
		errMsg          string
	}{
		{
			name: "Success - allocate from /24 subnet",
			generatorConfig: properties.JSON{
				"cidr":         "192.168.1.0/24",
				"excludeFirst": 1,
				"excludeLast":  1,
			},
			setupMock: func(repo *MockServicePoolValueRepository) {
				repo.EXPECT().
					FindByPool(ctx, poolID).
					Return([]*ServicePoolValue{}, nil)

				repo.EXPECT().
					Create(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						return v.ServicePoolID == poolID &&
							v.ServiceID != nil && *v.ServiceID == serviceID &&
							v.PropertyName != nil && *v.PropertyName == propertyName &&
							v.AllocatedAt != nil &&
							v.Value != nil
					})).
					Return(nil)
			},
			expectedValue: "192.168.1.1",
			expectErr:     false,
		},
		{
			name: "Success - skip already allocated IPs",
			generatorConfig: properties.JSON{
				"cidr":         "10.0.0.0/30",
				"excludeFirst": 1,
				"excludeLast":  1,
			},
			setupMock: func(repo *MockServicePoolValueRepository) {
				existingValue := &ServicePoolValue{
					BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
					Name:          "10.0.0.1",
					Value:         "10.0.0.1",
					ServicePoolID: poolID,
					ServiceID:     &serviceID,
					PropertyName:  &propertyName,
					AllocatedAt:   timePtr(time.Now()),
				}

				repo.EXPECT().
					FindByPool(ctx, poolID).
					Return([]*ServicePoolValue{existingValue}, nil)

				repo.EXPECT().
					Create(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						if v.Value == nil {
							return false
						}
						if ipStr, ok := v.Value.(string); ok {
							return ipStr == "10.0.0.2"
						}
						return false
					})).
					Return(nil)
			},
			expectedValue: "10.0.0.2",
			expectErr:     false,
		},
		{
			name: "Error - invalid CIDR",
			generatorConfig: properties.JSON{
				"cidr": "invalid",
			},
			setupMock: func(repo *MockServicePoolValueRepository) {},
			expectErr: true,
			errMsg:    "invalid CIDR",
		},
		{
			name: "Error - subnet exhausted",
			generatorConfig: properties.JSON{
				"cidr":         "10.0.0.0/31",
				"excludeFirst": 1,
				"excludeLast":  1,
			},
			setupMock: func(repo *MockServicePoolValueRepository) {
				repo.EXPECT().
					FindByPool(ctx, poolID).
					Return([]*ServicePoolValue{}, nil)
			},
			expectErr: true,
			errMsg:    "subnet exhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockServicePoolValueRepository(t)
			tt.setupMock(repo)

			generator := NewSubnetGenerator(repo, poolID, tt.generatorConfig)
			value, err := generator.Allocate(ctx, serviceID, propertyName)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, value)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, value)
				ipStr, ok := value.(string)
				require.True(t, ok, "value should be a string")
				assert.Equal(t, tt.expectedValue, ipStr)
			}
		})
	}
}

func TestSubnetGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	config := properties.JSON{"cidr": "10.0.0.0/24"}

	tests := []struct {
		name      string
		setupMock func(*MockServicePoolValueRepository)
		expectErr bool
		errMsg    string
	}{
		{
			name: "Success - release values from this pool",
			setupMock: func(repo *MockServicePoolValueRepository) {
				allocatedValue := &ServicePoolValue{
					BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
					Name:          "10.0.0.1",
					Value:         "10.0.0.1",
					ServicePoolID: poolID,
					ServiceID:     &serviceID,
					PropertyName:  helpers.StringPtr("ipAddress"),
					AllocatedAt:   timePtr(time.Now()),
				}

				repo.EXPECT().
					FindByService(ctx, serviceID).
					Return([]*ServicePoolValue{allocatedValue}, nil)

				repo.EXPECT().
					Update(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						return v.ID == allocatedValue.ID &&
							v.ServiceID == nil &&
							v.PropertyName == nil &&
							v.AllocatedAt == nil
					})).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name: "Success - skip values from other pools",
			setupMock: func(repo *MockServicePoolValueRepository) {
				otherPoolID := properties.UUID(uuid.New())
				allocatedValue := &ServicePoolValue{
					BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
					Name:          "10.0.0.1",
					Value:         "10.0.0.1",
					ServicePoolID: otherPoolID,
					ServiceID:     &serviceID,
					PropertyName:  helpers.StringPtr("ipAddress"),
					AllocatedAt:   timePtr(time.Now()),
				}

				repo.EXPECT().
					FindByService(ctx, serviceID).
					Return([]*ServicePoolValue{allocatedValue}, nil)
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockServicePoolValueRepository(t)
			tt.setupMock(repo)

			generator := NewSubnetGenerator(repo, poolID, config)
			err := generator.Release(ctx, serviceID)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// With no retentionSeconds a freed IP is reusable immediately: the next allocation
// re-allocates the existing row (Update) instead of minting a new one.
func TestSubnetGenerator_ReusesReleasedValue(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	releasedAt := time.Now().Add(-time.Hour)

	released := &ServicePoolValue{
		BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
		Name:          "10.0.0.1",
		Value:         "10.0.0.1",
		ServicePoolID: poolID,
		ReleasedAt:    &releasedAt,
	}

	repo := NewMockServicePoolValueRepository(t)
	repo.EXPECT().FindByPool(ctx, poolID).Return([]*ServicePoolValue{released}, nil)
	repo.EXPECT().
		Update(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
			return v.ID == released.ID && v.IsAllocated() &&
				v.ServiceID != nil && *v.ServiceID == serviceID && v.ReleasedAt == nil
		})).
		Return(nil)

	generator := NewSubnetGenerator(repo, poolID, properties.JSON{"cidr": "10.0.0.0/24", "excludeFirst": 1})
	value, err := generator.Allocate(ctx, serviceID, "ipAddress")
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", value)
}

// Within the retentionSeconds cooldown a freed IP stays reserved and the next free
// IP is minted instead.
func TestSubnetGenerator_RetentionHoldsValue(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	releasedAt := time.Now()

	released := &ServicePoolValue{
		BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
		Name:          "10.0.0.1",
		Value:         "10.0.0.1",
		ServicePoolID: poolID,
		ReleasedAt:    &releasedAt,
	}

	repo := NewMockServicePoolValueRepository(t)
	repo.EXPECT().FindByPool(ctx, poolID).Return([]*ServicePoolValue{released}, nil)
	repo.EXPECT().
		Create(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
			return v.Value == "10.0.0.2"
		})).
		Return(nil)

	generator := NewSubnetGenerator(repo, poolID, properties.JSON{"cidr": "10.0.0.0/24", "excludeFirst": 1, "retentionSeconds": float64(3600)})
	value, err := generator.Allocate(ctx, serviceID, "ipAddress")
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.2", value)
}

// neverReallocate keeps a freed IP out of circulation forever: even past any cooldown
// the freed value is skipped and the next free IP is minted.
func TestSubnetGenerator_NeverReallocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	releasedAt := time.Now().Add(-100 * time.Hour)

	released := &ServicePoolValue{
		BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
		Name:          "10.0.0.1",
		Value:         "10.0.0.1",
		ServicePoolID: poolID,
		ReleasedAt:    &releasedAt,
	}

	repo := NewMockServicePoolValueRepository(t)
	repo.EXPECT().FindByPool(ctx, poolID).Return([]*ServicePoolValue{released}, nil)
	repo.EXPECT().
		Create(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
			return v.Value == "10.0.0.2"
		})).
		Return(nil)

	generator := NewSubnetGenerator(repo, poolID, properties.JSON{"cidr": "10.0.0.0/24", "excludeFirst": 1, "neverReallocate": true})
	value, err := generator.Allocate(ctx, serviceID, "ipAddress")
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.2", value)
}
