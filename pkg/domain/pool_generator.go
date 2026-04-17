package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

type PoolListItem interface {
	PoolID() properties.UUID
	RawValue() any
	Allocate(poolID properties.UUID, propertyName string)
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

func (p *PoolListGenerator[V]) Allocate(ctx context.Context, poolID properties.UUID, propertyName string) (any, error) {
	availableValues, err := p.valueRepo.FindAvailable(ctx, p.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available values: %w", err)
	}

	if len(availableValues) == 0 {
		return nil, NewInvalidInputErrorf("no available values in pool")
	}

	value := availableValues[0]
	value.Allocate(poolID, propertyName)

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
