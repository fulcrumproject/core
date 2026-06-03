// List-based config pool generator implementation
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ConfigPoolListGenerator allocates values from a pre-configured list. Unlike the
// service pool list generator it can't reuse the generic PoolListGenerator because
// allocation is owner-type aware (agent or infrastructure), so it owns the small
// FindAvailable/set/Update loop itself.
type ConfigPoolListGenerator struct {
	repo   ConfigPoolValueRepository
	poolID properties.UUID
}

func NewConfigPoolListGenerator(valueRepo ConfigPoolValueRepository, poolID properties.UUID) *ConfigPoolListGenerator {
	return &ConfigPoolListGenerator{repo: valueRepo, poolID: poolID}
}

func (g *ConfigPoolListGenerator) Allocate(ctx context.Context, entityType ConfigPoolValueEntityType, entityID properties.UUID, propertyName string) (any, error) {
	available, err := g.repo.FindAvailable(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available values: %w", err)
	}
	if len(available) == 0 {
		return nil, NewInvalidInputErrorf("no available values in pool")
	}

	value := available[0]
	value.Allocate(entityType, entityID, propertyName)
	if err := g.repo.Update(ctx, value); err != nil {
		return nil, fmt.Errorf("failed to allocate value: %w", err)
	}
	return value.RawValue(), nil
}

func (g *ConfigPoolListGenerator) Release(ctx context.Context, values []*ConfigPoolValue) error {
	return releasePoolValues(ctx, g.repo, g.poolID, values)
}

var _ ConfigPoolGenerator = (*ConfigPoolListGenerator)(nil)
