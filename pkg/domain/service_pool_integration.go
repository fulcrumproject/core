// Service pool integration with service lifecycle
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

// AllocateServicePoolProperties allocates pool values for properties with servicePool validators
// Returns updated properties with allocated values
func AllocateServicePoolProperties(
	ctx context.Context,
	store Store,
	serviceID properties.UUID,
	poolSetID properties.UUID,
	propertySchema ServicePropertySchema,
	properties map[string]any,
) (map[string]any, error) {
	if propertySchema == nil {
		return properties, nil
	}

	// If no pool set, no allocation needed
	if poolSetID == uuid.Nil {
		return properties, nil
	}

	// Create a copy of properties to modify
	result := make(map[string]any)
	for k, v := range properties {
		result[k] = v
	}

	// Find properties with servicePool validators and allocate
	for propName, propDef := range propertySchema {
		// Check if this property has a servicePool validator
		var poolType string
		for _, validator := range propDef.Validators {
			if validator.Type == SchemaValidatorServicePool {
				// Get the pool type from validator value
				if poolTypeStr, ok := validator.Value.(string); ok {
					poolType = poolTypeStr
					break
				}
			}
		}

		if poolType == "" {
			continue // No servicePool validator for this property
		}

		// Find the pool in the pool set with matching type
		pools, err := store.ServicePoolRepo().ListByPoolSet(ctx, poolSetID)
		if err != nil {
			return nil, fmt.Errorf("failed to list pools for pool set: %w", err)
		}

		var targetPool *ServicePool
		for _, pool := range pools {
			if pool.Type == poolType {
				targetPool = pool
				break
			}
		}

		if targetPool == nil {
			return nil, fmt.Errorf("no pool found with type %s in pool set", poolType)
		}

		// Create generator factory and allocate
		factory := NewDefaultGeneratorFactory(store.ServicePoolValueRepo())
		generator, err := factory.CreateGenerator(targetPool)
		if err != nil {
			return nil, fmt.Errorf("failed to create generator for pool %s: %w", targetPool.ID, err)
		}

		// Allocate value from pool
		allocatedValue, err := generator.Allocate(ctx, targetPool.ID, serviceID, propName)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate from pool %s for property %s: %w", targetPool.ID, propName, err)
		}

		// Store allocated value in properties
		result[propName] = allocatedValue
	}

	return result, nil
}

// ReleaseServicePoolAllocations releases all pool allocations for a service
func ReleaseServicePoolAllocations(
	ctx context.Context,
	store Store,
	serviceID properties.UUID,
) error {
	// Find all ServicePoolValues allocated to this service
	allocatedValues, err := store.ServicePoolValueRepo().FindByService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to find allocated pool values: %w", err)
	}

	// Release each value
	for _, value := range allocatedValues {
		value.ServiceID = nil
		value.PropertyName = nil
		value.AllocatedAt = nil

		if err := store.ServicePoolValueRepo().Update(ctx, value); err != nil {
			return fmt.Errorf("failed to release pool value %s: %w", value.ID, err)
		}
	}

	return nil
}
