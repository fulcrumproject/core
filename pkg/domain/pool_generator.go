package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

// retentionConfigKey is the optional generatorConfig key holding the reuse
// cooldown, in seconds.
const retentionConfigKey = "retentionSeconds"

// neverReallocateConfigKey is the optional generatorConfig boolean: when true a
// released value is never re-allocated, taking precedence over retentionSeconds.
const neverReallocateConfigKey = "neverReallocate"

// parseRetention reads the optional retentionSeconds key. Absent (or 0) means a
// freed value may be reallocated immediately; a positive value is the cooldown
// that must elapse since release. A non-integer or negative value is rejected.
func parseRetention(cfg properties.JSON) (time.Duration, error) {
	raw, present := cfg[retentionConfigKey]
	if !present {
		return 0, nil
	}
	n, ok := toInt(raw)
	if !ok {
		return 0, NewInvalidInputError("{key} must be an integer number of seconds", MsgData{"key": retentionConfigKey})
	}
	if n < 0 {
		return 0, NewInvalidInputError("{key} must be higher than 0", MsgData{"key": retentionConfigKey})
	}
	return time.Duration(n) * time.Second, nil
}

// parseNeverReallocate reads the optional neverReallocate key. Absent means
// false; a non-boolean value is rejected.
func parseNeverReallocate(cfg properties.JSON) (bool, error) {
	raw, present := cfg[neverReallocateConfigKey]
	if !present {
		return false, nil
	}
	b, ok := raw.(bool)
	if !ok {
		return false, NewInvalidInputError("{key} must be a boolean", MsgData{"key": neverReallocateConfigKey})
	}
	return b, nil
}

// retentionAllows reports whether a freed value may be reallocated now: a value
// that was never released is always allocatable; neverReallocate keeps a released
// value out of circulation forever; otherwise its cooldown must have elapsed.
func retentionAllows(retention time.Duration, neverReallocate bool, releasedAt *time.Time, now time.Time) bool {
	if releasedAt == nil {
		return true
	}
	if neverReallocate {
		return false
	}
	return now.Sub(*releasedAt) >= retention
}

type PoolListItem interface {
	PoolID() properties.UUID
	RawValue() any
	Allocate(entityID properties.UUID, propertyName string)
	Release()
}

type PoolListRepo[V PoolListItem] interface {
	FindAvailable(ctx context.Context, entityID properties.UUID) ([]V, error)
	Update(ctx context.Context, value V) error
}

type PoolListGenerator[V PoolListItem] struct {
	valueRepo PoolListRepo[V]
	poolID    properties.UUID
}

func NewPoolListGenerator[V PoolListItem](valueRepo PoolListRepo[V], poolID properties.UUID) *PoolListGenerator[V] {
	return &PoolListGenerator[V]{valueRepo: valueRepo, poolID: poolID}
}

func (p *PoolListGenerator[V]) Allocate(ctx context.Context, entityID properties.UUID, propertyName string) (any, error) {
	availableValues, err := p.valueRepo.FindAvailable(ctx, p.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available values: %w", err)
	}

	if len(availableValues) == 0 {
		return nil, NewInvalidInputError("no available values in pool", nil)
	}

	value := availableValues[0]
	value.Allocate(entityID, propertyName)

	err = p.valueRepo.Update(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate value: %w", err)
	}

	return value.RawValue(), nil
}

func (p *PoolListGenerator[V]) Release(ctx context.Context, values []V) error {
	for _, v := range values {
		// Continue if pool id doesn't match
		if v.PoolID() != p.poolID {
			continue
		}

		v.Release()
		if err := p.valueRepo.Update(ctx, v); err != nil {
			return fmt.Errorf("failed to release value: %w", err)
		}
	}
	return nil
}
