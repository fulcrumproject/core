// List generator tests
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

func TestListGenerator_Allocate(t *testing.T) {
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

			generator := NewListGenerator(repo, poolID)
			value, err := generator.Allocate(ctx, poolID, serviceID, propertyName)

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

func TestListGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	serviceID := properties.UUID(uuid.New())

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
					Name:          "IP 1",
					Value:         "192.168.1.10",
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
					Name:          "IP 1",
					Value:         "192.168.1.10",
					ServicePoolID: otherPoolID, // Different pool
					ServiceID:     &serviceID,
					PropertyName:  helpers.StringPtr("ipAddress"),
					AllocatedAt:   timePtr(time.Now()),
				}

				repo.EXPECT().
					FindByService(ctx, serviceID).
					Return([]*ServicePoolValue{allocatedValue}, nil)

				// No Update call expected since it's from a different pool
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

			generator := NewListGenerator(repo, poolID)
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
