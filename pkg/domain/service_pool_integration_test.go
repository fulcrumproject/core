// Service pool integration tests
package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAllocateServicePoolProperties(t *testing.T) {
	ctx := context.Background()
	serviceID := properties.UUID(uuid.New())
	poolSetID := properties.UUID(uuid.New())
	poolID := properties.UUID(uuid.New())

	// Define a property schema with a servicePool validator
	schema := ServicePropertySchema{
		"publicIp": {
			Type:   "string",
			Source: "system",
			Validators: []ServicePropertyValidatorDefinition{
				{
					Type:  SchemaValidatorServicePool,
					Value: "public_ip",
				},
			},
		},
		"cpu": {
			Type:     "integer",
			Source:   "input",
			Required: true,
		},
	}

	// Create mocks
	store := NewMockStore(t)
	poolRepo := NewMockServicePoolRepository(t)
	valueRepo := NewMockServicePoolValueRepository(t)

	// Set up expectations
	store.On("ServicePoolRepo").Return(poolRepo)
	store.On("ServicePoolValueRepo").Return(valueRepo)

	// Mock the pool lookup
	pool := &ServicePool{
		BaseEntity: BaseEntity{
			ID: poolID,
		},
		Name:             "Public IPs",
		Type:             "public_ip",
		GeneratorType:    PoolGeneratorList,
		ServicePoolSetID: poolSetID,
	}
	poolRepo.On("ListByPoolSet", ctx, poolSetID).Return([]*ServicePool{pool}, nil)

	// Mock the value allocation
	availableValue := &ServicePoolValue{
		BaseEntity: BaseEntity{
			ID: properties.UUID(uuid.New()),
		},
		Name:          "IP-001",
		Value:         "192.168.1.10",
		ServicePoolID: poolID,
	}
	valueRepo.On("FindAvailable", ctx, poolID).Return([]*ServicePoolValue{availableValue}, nil)
	valueRepo.On("Update", ctx, mock.AnythingOfType("*domain.ServicePoolValue")).Return(nil)

	// Test allocation
	inputProps := map[string]any{
		"cpu": 4,
	}

	result, err := AllocateServicePoolProperties(ctx, store, serviceID, poolSetID, schema, inputProps)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result["cpu"])
	assert.Equal(t, "192.168.1.10", result["publicIp"])

	// Verify mocks were called
	store.AssertExpectations(t)
	poolRepo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestAllocateServicePoolProperties_NoPoolSet(t *testing.T) {
	ctx := context.Background()
	serviceID := properties.UUID(uuid.New())

	schema := ServicePropertySchema{
		"publicIp": {
			Type:   "string",
			Source: "system",
			Validators: []ServicePropertyValidatorDefinition{
				{
					Type:  SchemaValidatorServicePool,
					Value: "public_ip",
				},
			},
		},
	}

	store := NewMockStore(t)

	// No pool set (nil UUID)
	result, err := AllocateServicePoolProperties(ctx, store, serviceID, uuid.Nil, schema, map[string]any{})

	// Should succeed without allocation
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestAllocateServicePoolProperties_NilSchema(t *testing.T) {
	ctx := context.Background()
	serviceID := properties.UUID(uuid.New())
	poolSetID := properties.UUID(uuid.New())

	store := NewMockStore(t)

	inputProps := map[string]any{"cpu": 4}
	result, err := AllocateServicePoolProperties(ctx, store, serviceID, poolSetID, nil, inputProps)

	// Should succeed and return original properties
	assert.NoError(t, err)
	assert.Equal(t, inputProps, result)
}
