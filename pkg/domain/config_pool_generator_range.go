package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ConfigPoolRangeGenerator allocates the lowest free integer in a [min,max] range,
// skipping excluded values. It mints a ConfigPoolValue row per allocation.
type ConfigPoolRangeGenerator struct {
	repo   ConfigPoolValueRepository
	poolID properties.UUID
	config properties.JSON
}

func NewConfigPoolRangeGenerator(repo ConfigPoolValueRepository, poolID properties.UUID, config properties.JSON) *ConfigPoolRangeGenerator {
	return &ConfigPoolRangeGenerator{repo: repo, poolID: poolID, config: config}
}

func (g *ConfigPoolRangeGenerator) Allocate(ctx context.Context, entityType ConfigPoolValueEntityType, entityID properties.UUID, propertyName string) (any, error) {
	min, max, exclude, err := parseRangeConfig(g.config)
	if err != nil {
		return nil, err
	}

	existing, err := g.repo.FindByPool(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pool values: %w", err)
	}
	used := make(map[int]bool, len(existing))
	for _, v := range existing {
		if n, ok := toInt(v.Value); ok {
			used[n] = true
		}
	}

	for n := min; n <= max; n++ {
		if exclude[n] || used[n] {
			continue
		}
		value := &ConfigPoolValue{Name: strconv.Itoa(n), Value: n, ConfigPoolID: g.poolID}
		value.Allocate(entityType, entityID, propertyName)
		if err := g.repo.Create(ctx, value); err != nil {
			return nil, fmt.Errorf("failed to allocate value: %w", err)
		}
		return value.RawValue(), nil
	}
	return nil, NewInvalidInputErrorf("range exhausted: no available values in pool")
}

func (g *ConfigPoolRangeGenerator) Release(ctx context.Context, values []*ConfigPoolValue) error {
	return releasePoolValues(ctx, g.repo, g.poolID, values)
}

func validateRangeGeneratorConfig(cfg properties.JSON) error {
	_, _, _, err := parseRangeConfig(cfg)
	return err
}

func parseRangeConfig(cfg properties.JSON) (int, int, map[int]bool, error) {
	min, ok := toInt(cfg["min"])
	if !ok {
		return 0, 0, nil, fmt.Errorf("range generator config requires integer 'min'")
	}
	max, ok := toInt(cfg["max"])
	if !ok {
		return 0, 0, nil, fmt.Errorf("range generator config requires integer 'max'")
	}
	if min > max {
		return 0, 0, nil, fmt.Errorf("range generator config 'min' (%d) must be <= 'max' (%d)", min, max)
	}
	exclude := map[int]bool{}
	if raw, present := cfg["exclude"]; present {
		list, ok := raw.([]any)
		if !ok {
			return 0, 0, nil, fmt.Errorf("range generator config 'exclude' must be an array")
		}
		for _, e := range list {
			n, ok := toInt(e)
			if !ok {
				return 0, 0, nil, fmt.Errorf("range generator config 'exclude' entries must be integers")
			}
			exclude[n] = true
		}
	}
	return min, max, exclude, nil
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

// releasePoolValues deletes, in a single query, the passed values that belong to
// poolID. Shared by the algorithmic generators (range, subnet).
func releasePoolValues(ctx context.Context, repo ConfigPoolValueRepository, poolID properties.UUID, values []*ConfigPoolValue) error {
	ids := make([]properties.UUID, 0, len(values))
	for _, v := range values {
		if v.PoolID() == poolID {
			ids = append(ids, v.ID)
		}
	}
	if err := repo.DeleteByIDs(ctx, ids); err != nil {
		return fmt.Errorf("failed to release values: %w", err)
	}
	return nil
}

var _ ConfigPoolGenerator = (*ConfigPoolRangeGenerator)(nil)
