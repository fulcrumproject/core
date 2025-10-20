// Pool allocation logic tests
package domain

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestAllocateFromList tests allocating a value from a list-type pool
func TestAllocateFromList(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	propertyName := "ipAddress"

	tests := []struct {
		name          string
		setupMock     func(*MockServicePoolValueRepository)
		expectedValue any
		expectErr     bool
		errMsg        string
	}{
		{
			name: "Success - allocate first available value",
			setupMock: func(repo *MockServicePoolValueRepository) {
				availableValue := &ServicePoolValue{
					BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
					Name:          "IP 1",
					Value:         "192.168.1.10",
					ServicePoolID: poolID,
				}
				repo.EXPECT().
					FindAvailable(ctx, poolID).
					Return([]*ServicePoolValue{availableValue}, nil)

				repo.EXPECT().
					Update(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						return v.ServiceID != nil && *v.ServiceID == serviceID &&
							v.PropertyName != nil && *v.PropertyName == propertyName &&
							v.AllocatedAt != nil
					})).
					Return(nil)
			},
			expectedValue: "192.168.1.10",
			expectErr:     false,
		},
		{
			name: "Error - no available values",
			setupMock: func(repo *MockServicePoolValueRepository) {
				repo.EXPECT().
					FindAvailable(ctx, poolID).
					Return([]*ServicePoolValue{}, nil)
			},
			expectErr: true,
			errMsg:    "no available values in pool",
		},
		{
			name: "Error - repository query fails",
			setupMock: func(repo *MockServicePoolValueRepository) {
				repo.EXPECT().
					FindAvailable(ctx, poolID).
					Return(nil, NewInvalidInputErrorf("database error"))
			},
			expectErr: true,
			errMsg:    "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockServicePoolValueRepository(t)
			tt.setupMock(repo)

			value, err := AllocateFromList(ctx, repo, poolID, serviceID, propertyName)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, value)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

// TestAllocateFromSubnet tests allocating an IP from a subnet-type pool
func TestAllocateFromSubnet(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())
	propertyName := "ipAddress"

	tests := []struct {
		name            string
		generatorConfig properties.JSON
		setupMock       func(*MockServicePoolValueRepository)
		expectedValue   string // IP address as string for easier testing
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
				// First call: get existing allocations
				repo.EXPECT().
					FindByPool(ctx, poolID).
					Return([]*ServicePoolValue{}, nil)

				// Second call: create new value
				repo.EXPECT().
					Create(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						// Verify it creates with allocation info
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
				// First IP (10.0.0.1) is already allocated
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

				// Should create with next available IP (10.0.0.2)
				repo.EXPECT().
					Create(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						if v.Value == nil {
							return false
						}
						// Check that the IP is 10.0.0.2
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
				"cidr":         "10.0.0.0/31", // Only 2 IPs total
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

			value, err := AllocateFromSubnet(ctx, repo, poolID, serviceID, propertyName, tt.generatorConfig)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, value)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, value)
				// Extract IP string from value
				ipStr, ok := value.(string)
				require.True(t, ok, "value should be a string")
				assert.Equal(t, tt.expectedValue, ipStr)
			}
		})
	}
}

// TestReleasePoolAllocations tests releasing all pool allocations for a service
func TestReleasePoolAllocations(t *testing.T) {
	ctx := context.Background()
	serviceID := properties.UUID(uuid.New())

	tests := []struct {
		name      string
		setupMock func(*MockServicePoolValueRepository)
		expectErr bool
		errMsg    string
	}{
		{
			name: "Success - release multiple allocations",
			setupMock: func(repo *MockServicePoolValueRepository) {
				allocatedValues := []*ServicePoolValue{
					{
						BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
						Name:          "IP 1",
						Value:         "192.168.1.10",
						ServicePoolID: properties.UUID(uuid.New()),
						ServiceID:     &serviceID,
						PropertyName:  stringPtr("ipAddress"),
						AllocatedAt:   timePtr(time.Now()),
					},
					{
						BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
						Name:          "IP 2",
						Value:         "192.168.1.11",
						ServicePoolID: properties.UUID(uuid.New()),
						ServiceID:     &serviceID,
						PropertyName:  stringPtr("backupIp"),
						AllocatedAt:   timePtr(time.Now()),
					},
				}

				repo.EXPECT().
					FindByService(ctx, serviceID).
					Return(allocatedValues, nil)

				// Expect updates to release each value
				for _, v := range allocatedValues {
					repo.EXPECT().
						Update(ctx, mock.MatchedBy(func(updated *ServicePoolValue) bool {
							return updated.ID == v.ID &&
								updated.ServiceID == nil &&
								updated.PropertyName == nil &&
								updated.AllocatedAt == nil
						})).
						Return(nil)
				}
			},
			expectErr: false,
		},
		{
			name: "Success - no allocations to release",
			setupMock: func(repo *MockServicePoolValueRepository) {
				repo.EXPECT().
					FindByService(ctx, serviceID).
					Return([]*ServicePoolValue{}, nil)
			},
			expectErr: false,
		},
		{
			name: "Error - query fails",
			setupMock: func(repo *MockServicePoolValueRepository) {
				repo.EXPECT().
					FindByService(ctx, serviceID).
					Return(nil, NewInvalidInputErrorf("database error"))
			},
			expectErr: true,
			errMsg:    "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockServicePoolValueRepository(t)
			tt.setupMock(repo)

			err := ReleasePoolAllocations(ctx, repo, serviceID)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestAllocatePoolProperty tests the helper function that routes to the correct generator
func TestAllocatePoolProperty(t *testing.T) {
	ctx := context.Background()
	serviceID := properties.UUID(uuid.New())
	propertyName := "testProperty"

	tests := []struct {
		name          string
		pool          *ServicePool
		setupMock     func(*MockServicePoolRepository, *MockServicePoolValueRepository, properties.UUID)
		expectedValue any
		expectErr     bool
		errMsg        string
	}{
		{
			name: "Success - list pool",
			pool: &ServicePool{
				BaseEntity:      BaseEntity{ID: properties.UUID(uuid.New())},
				Name:            "Test Pool",
				Type:            "ip",
				GeneratorType:   PoolGeneratorList,
				GeneratorConfig: nil,
			},
			setupMock: func(poolRepo *MockServicePoolRepository, valueRepo *MockServicePoolValueRepository, poolID properties.UUID) {
				pool := &ServicePool{
					BaseEntity:      BaseEntity{ID: poolID},
					Name:            "Test Pool",
					Type:            "ip",
					GeneratorType:   PoolGeneratorList,
					GeneratorConfig: nil,
				}
				poolRepo.EXPECT().
					Get(ctx, poolID).
					Return(pool, nil)

				availableValue := &ServicePoolValue{
					BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
					Name:          "Value 1",
					Value:         "test-value",
					ServicePoolID: poolID,
				}
				valueRepo.EXPECT().
					FindAvailable(ctx, poolID).
					Return([]*ServicePoolValue{availableValue}, nil)

				valueRepo.EXPECT().
					Update(ctx, mock.MatchedBy(func(v *ServicePoolValue) bool {
						return v.ID == availableValue.ID &&
							v.ServiceID != nil && *v.ServiceID == serviceID
					})).
					Return(nil)
			},
			expectedValue: "test-value",
			expectErr:     false,
		},
		{
			name: "Success - subnet pool",
			pool: &ServicePool{
				BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
				Name:          "IP Pool",
				Type:          "ip",
				GeneratorType: PoolGeneratorSubnet,
				GeneratorConfig: &properties.JSON{
					"cidr":         "10.0.0.0/24",
					"excludeFirst": 1,
					"excludeLast":  1,
				},
			},
			setupMock: func(poolRepo *MockServicePoolRepository, valueRepo *MockServicePoolValueRepository, poolID properties.UUID) {
				pool := &ServicePool{
					BaseEntity:    BaseEntity{ID: poolID},
					Name:          "IP Pool",
					Type:          "ip",
					GeneratorType: PoolGeneratorSubnet,
					GeneratorConfig: &properties.JSON{
						"cidr":         "10.0.0.0/24",
						"excludeFirst": 1,
						"excludeLast":  1,
					},
				}
				poolRepo.EXPECT().
					Get(ctx, poolID).
					Return(pool, nil)

				valueRepo.EXPECT().
					FindByPool(ctx, poolID).
					Return([]*ServicePoolValue{}, nil)

				valueRepo.EXPECT().
					Create(ctx, mock.Anything).
					Return(nil)
			},
			expectedValue: "10.0.0.1", // First available IP
			expectErr:     false,
		},
		{
			name: "Error - pool not found",
			pool: nil,
			setupMock: func(poolRepo *MockServicePoolRepository, valueRepo *MockServicePoolValueRepository, poolID properties.UUID) {
				poolRepo.EXPECT().
					Get(ctx, poolID).
					Return(nil, NewNotFoundErrorf("pool not found"))
			},
			expectErr: true,
			errMsg:    "not found",
		},
		{
			name: "Error - invalid generator type",
			pool: &ServicePool{
				BaseEntity:      BaseEntity{ID: properties.UUID(uuid.New())},
				Name:            "Test Pool",
				Type:            "ip",
				GeneratorType:   "invalid",
				GeneratorConfig: nil,
			},
			setupMock: func(poolRepo *MockServicePoolRepository, valueRepo *MockServicePoolValueRepository, poolID properties.UUID) {
				pool := &ServicePool{
					BaseEntity:      BaseEntity{ID: poolID},
					Name:            "Test Pool",
					Type:            "ip",
					GeneratorType:   "invalid",
					GeneratorConfig: nil,
				}
				poolRepo.EXPECT().
					Get(ctx, poolID).
					Return(pool, nil)
			},
			expectErr: true,
			errMsg:    "unsupported pool generator type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolRepo := NewMockServicePoolRepository(t)
			valueRepo := NewMockServicePoolValueRepository(t)

			var poolID properties.UUID
			if tt.pool != nil {
				poolID = tt.pool.ID
			} else {
				poolID = properties.UUID(uuid.New())
			}

			tt.setupMock(poolRepo, valueRepo, poolID)

			value, err := AllocatePoolProperty(ctx, poolRepo, valueRepo, poolID, serviceID, propertyName)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, value)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}
