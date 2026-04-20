// List-based pool generator implementation
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ListGenerator allocates values from a pre-configured list
type ListGenerator struct {
	*PoolListGenerator[*ServicePoolValue]
	repo ServicePoolValueRepository
}

// NewListGenerator creates a new list-based generator
func NewListGenerator(valueRepo ServicePoolValueRepository, poolID properties.UUID) *ListGenerator {
	return &ListGenerator{
		PoolListGenerator: NewPoolListGenerator(valueRepo, poolID),
		repo:              valueRepo,
	}
}

func (g *ListGenerator) Release(ctx context.Context, serviceID properties.UUID) error {
	allocatedValues, err := g.repo.FindByService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to query allocated values: %w", err)
	}
	return g.PoolListGenerator.Release(ctx, allocatedValues)
}

var _ PoolGenerator = (*ListGenerator)(nil)
