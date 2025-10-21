// Generator factory tests
package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGeneratorFactory_CreateGenerator(t *testing.T) {
	valueRepo := NewMockServicePoolValueRepository(t)
	factory := NewDefaultGeneratorFactory(valueRepo)

	tests := []struct {
		name         string
		pool         *ServicePool
		expectedType string
		expectErr    bool
		errMsg       string
	}{
		{
			name: "Success - create list generator",
			pool: &ServicePool{
				BaseEntity:      BaseEntity{ID: properties.UUID(uuid.New())},
				Name:            "Test Pool",
				Type:            "string",
				GeneratorType:   PoolGeneratorList,
				GeneratorConfig: nil,
			},
			expectedType: "*domain.ListGenerator",
			expectErr:    false,
		},
		{
			name: "Success - create subnet generator",
			pool: &ServicePool{
				BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
				Name:          "IP Pool",
				Type:          "ip",
				GeneratorType: PoolGeneratorSubnet,
				GeneratorConfig: &properties.JSON{
					"cidr": "10.0.0.0/24",
				},
			},
			expectedType: "*domain.SubnetGenerator",
			expectErr:    false,
		},
		{
			name: "Error - subnet generator missing config",
			pool: &ServicePool{
				BaseEntity:      BaseEntity{ID: properties.UUID(uuid.New())},
				Name:            "IP Pool",
				Type:            "ip",
				GeneratorType:   PoolGeneratorSubnet,
				GeneratorConfig: nil,
			},
			expectErr: true,
			errMsg:    "subnet pool missing generator config",
		},
		{
			name: "Error - unsupported generator type",
			pool: &ServicePool{
				BaseEntity:      BaseEntity{ID: properties.UUID(uuid.New())},
				Name:            "Test Pool",
				Type:            "unknown",
				GeneratorType:   "unsupported",
				GeneratorConfig: nil,
			},
			expectErr: true,
			errMsg:    "unsupported pool generator type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := factory.CreateGenerator(tt.pool)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, generator)
			} else {
				require.NoError(t, err)
				require.NotNil(t, generator)

				// Verify correct generator type
				switch tt.expectedType {
				case "*domain.ListGenerator":
					_, ok := generator.(*ListGenerator)
					assert.True(t, ok, "expected ListGenerator")
				case "*domain.SubnetGenerator":
					_, ok := generator.(*SubnetGenerator)
					assert.True(t, ok, "expected SubnetGenerator")
				}
			}
		})
	}
}
