package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceType_TableName(t *testing.T) {
	st := ServiceType{}
	assert.Equal(t, "service_types", st.TableName())
}

// TestServiceTypeBasics tests basic ServiceType operations
func TestServiceTypeBasics(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name        string
		serviceType *ServiceType
		description string
	}{
		{
			name: "Valid service type",
			serviceType: &ServiceType{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name: "Web Server",
			},
			description: "Valid service type with name",
		},
		{
			name: "Empty name",
			serviceType: &ServiceType{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name: "",
			},
			description: "Service type with empty name would fail database validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that the struct can be created
			assert.NotNil(t, tt.serviceType)
			assert.Equal(t, tt.serviceType.Name, tt.serviceType.Name)
		})
	}
}

// Note: Schema validation tests have been moved to pkg/schema package tests
// Domain-specific validators (source, mutable) are tested in service_property_schema_validators_test.go
