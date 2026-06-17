package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ConfigPoolRangeGenerator allocates the lowest free integer in a [min,max] range,
// skipping excluded values. It reuses a freed value once its retention cooldown has
// elapsed, otherwise mints a new ConfigPoolValue row.
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
	retention, err := parseRetention(g.config)
	if err != nil {
		return nil, err
	}
	neverReallocate, err := parseNeverReallocate(g.config)
	if err != nil {
		return nil, err
	}

	existing, err := g.repo.FindByPool(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pool values: %w", err)
	}

	// Partition existing rows: allocated or still-cooling values are reserved (skipped);
	// freed values past their cooldown are reusable and re-allocated in place. With
	// neverReallocate a freed value stays reserved permanently.
	now := time.Now()
	reserved := make(map[int]bool, len(existing))
	reusable := make(map[int]*ConfigPoolValue, len(existing))
	for _, v := range existing {
		n, ok := toInt(v.Value)
		if !ok {
			continue
		}
		if v.IsAllocated() || !retentionAllows(retention, neverReallocate, v.ReleasedAt, now) {
			reserved[n] = true
		} else {
			reusable[n] = v
		}
	}

	for n := min; n <= max; n++ {
		if exclude[n] || reserved[n] {
			continue
		}
		if row, found := reusable[n]; found {
			row.Allocate(entityType, entityID, propertyName)
			if err := g.repo.Update(ctx, row); err != nil {
				return nil, fmt.Errorf("failed to allocate value: %w", err)
			}
			return row.RawValue(), nil
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

// releasePoolValues clears the allocation fields of the passed values that belong to
// poolID and stamps released_at, keeping the rows so they can be reused once their
// retention cooldown elapses. Shared by the algorithmic generators (range, subnet).
func releasePoolValues(ctx context.Context, repo ConfigPoolValueRepository, poolID properties.UUID, values []*ConfigPoolValue) error {
	for _, v := range values {
		if v.PoolID() != poolID {
			continue
		}
		v.Release()
		if err := repo.Update(ctx, v); err != nil {
			return fmt.Errorf("failed to release value: %w", err)
		}
	}
	return nil
}

var _ ConfigPoolGenerator = (*ConfigPoolRangeGenerator)(nil)
