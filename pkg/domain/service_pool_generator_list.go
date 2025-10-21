// List-based pool generator implementation
package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ListGenerator allocates values from a pre-configured list
type ListGenerator struct {
	valueRepo ServicePoolValueRepository
	poolID    properties.UUID
}

// NewListGenerator creates a new list-based generator
func NewListGenerator(valueRepo ServicePoolValueRepository, poolID properties.UUID) *ListGenerator {
	return &ListGenerator{
		valueRepo: valueRepo,
		poolID:    poolID,
	}
}

// Allocate allocates an available value from the list
func (g *ListGenerator) Allocate(ctx context.Context, poolID properties.UUID, serviceID properties.UUID, propertyName string) (any, error) {
	// Find available values
	availableValues, err := g.valueRepo.FindAvailable(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available values: %w", err)
	}

	if len(availableValues) == 0 {
		return nil, NewInvalidInputErrorf("no available values in pool")
	}

	// Take the first available value
	value := availableValues[0]

	// Mark as allocated
	now := time.Now()
	value.ServiceID = &serviceID
	value.PropertyName = &propertyName
	value.AllocatedAt = &now

	// Update the value
	err = g.valueRepo.Update(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate value: %w", err)
	}

	// Return the actual value for copying to service property
	return value.Value, nil
}

// Release releases all allocations for the given service
func (g *ListGenerator) Release(ctx context.Context, serviceID properties.UUID) error {
	// Find all values allocated to this service
	allocatedValues, err := g.valueRepo.FindByService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to query allocated values: %w", err)
	}

	// Release each value
	for _, value := range allocatedValues {
		// Only release values from this pool
		if value.ServicePoolID != g.poolID {
			continue
		}

		value.ServiceID = nil
		value.PropertyName = nil
		value.AllocatedAt = nil

		err = g.valueRepo.Update(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to release value: %w", err)
		}
	}

	return nil
}
