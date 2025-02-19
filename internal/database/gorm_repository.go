package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

// Entity defines the interface that all entities must implement
type Entity interface {
	GetID() domain.UUID
}

// Repository defines the generic repository interface
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Save(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id domain.UUID) error
	FindByID(ctx context.Context, id domain.UUID) (*T, error)
	List(ctx context.Context, page *domain.PageRequest) (*domain.PageResponse[T], error)
	Count(ctx context.Context, conditions ...interface{}) (int64, error)
}

// GormRepository provides a base implementation of Repository using GORM
type GormRepository[T any] struct {
	db               *gorm.DB
	filterApplier    PageApplier
	sortApplier      PageApplier
	findPreloadPaths []string
	listPreloadPaths []string
}

// NewGormRepository creates a new instance of GormRepository
func NewGormRepository[T any](
	db *gorm.DB,
	filterApplier PageApplier,
	sortApplier PageApplier,
	findPreloadPaths []string,
	listPreloadPaths []string,
) *GormRepository[T] {
	return &GormRepository[T]{
		db:               db,
		filterApplier:    filterApplier,
		sortApplier:      sortApplier,
		findPreloadPaths: findPreloadPaths,
		listPreloadPaths: listPreloadPaths,
	}
}

func (r *GormRepository[T]) Create(ctx context.Context, entity *T) error {
	result := r.db.WithContext(ctx).Create(entity)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *GormRepository[T]) Save(ctx context.Context, entity *T) error {
	result := r.db.WithContext(ctx).Save(entity)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *GormRepository[T]) Delete(ctx context.Context, id domain.UUID) error {
	result := r.db.WithContext(ctx).Delete(new(T), id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *GormRepository[T]) FindByID(ctx context.Context, id domain.UUID) (*T, error) {
	entity := new(T)
	query := r.db.WithContext(ctx)

	for _, path := range r.findPreloadPaths {
		query = query.Preload(path)
	}

	err := query.First(entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return entity, nil
}

func (r *GormRepository[T]) List(ctx context.Context, page *domain.PageRequest) (*domain.PageResponse[T], error) {
	return list[T](
		ctx,
		r.db,
		page,
		r.filterApplier,
		r.sortApplier,
		r.listPreloadPaths,
	)
}

func (r *GormRepository[T]) Count(ctx context.Context, conditions ...interface{}) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(new(T))

	if len(conditions) > 0 {
		query = query.Where(conditions[0], conditions[1:]...)
	}

	result := query.Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}
