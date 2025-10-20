// ServicePoolSet entity tests
package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServicePoolSet(t *testing.T) {
	providerID := properties.UUID(uuid.New())

	params := CreateServicePoolSetParams{
		Name:       "Test Pool Set",
		ProviderID: providerID,
	}

	poolSet := NewServicePoolSet(params)

	assert.Equal(t, "Test Pool Set", poolSet.Name)
	assert.Equal(t, providerID, poolSet.ProviderID)
}

func TestServicePoolSet_Validate(t *testing.T) {
	providerID := properties.UUID(uuid.New())

	tests := []struct {
		name      string
		poolSet   *ServicePoolSet
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid pool set",
			poolSet: &ServicePoolSet{
				Name:       "Valid Pool Set",
				ProviderID: providerID,
			},
			wantError: false,
		},
		{
			name: "empty name",
			poolSet: &ServicePoolSet{
				Name:       "",
				ProviderID: providerID,
			},
			wantError: true,
			errorMsg:  "pool set name cannot be empty",
		},
		{
			name: "empty provider ID",
			poolSet: &ServicePoolSet{
				Name:       "Test Pool Set",
				ProviderID: properties.UUID{},
			},
			wantError: true,
			errorMsg:  "provider ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.poolSet.Validate()
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServicePoolSet_TableName(t *testing.T) {
	poolSet := &ServicePoolSet{}
	assert.Equal(t, "service_pool_sets", poolSet.TableName())
}
