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

// ReadRepository defines the generic repository interface
type ReadRepository[T any] interface {
	FindByID(ctx context.Context, id domain.UUID) (*T, error)
	List(ctx context.Context, page *domain.PageRequest) (*domain.PageResponse[T], error)
	Count(ctx context.Context, conditions ...interface{}) (int64, error)
	Exists(ctx context.Context, id domain.UUID) (bool, error)
}

// Repository defines the generic repository interface
type WriteRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Save(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id domain.UUID) error
}

type AuthzFilterApplier func(scope *domain.AuthScope, db *gorm.DB) *gorm.DB

type Tabler interface {
	TableName() string
}

// GormRepository provides a base implementation of Repository using GORM
type GormRepository[T Tabler] struct {
	db                 *gorm.DB
	filterApplier      PageFilterApplier
	sortApplier        PageFilterApplier
	findPreloadPaths   []string
	listPreloadPaths   []string
	authzFilterApplier AuthzFilterApplier
}

// NewGormRepository creates a new instance of GormRepository
func NewGormRepository[T Tabler](
	db *gorm.DB,
	filterApplier PageFilterApplier,
	sortApplier PageFilterApplier,
	authzFilterApplier AuthzFilterApplier,
	findPreloadPaths []string,
	listPreloadPaths []string,
) *GormRepository[T] {
	return &GormRepository[T]{
		db:                 db,
		filterApplier:      filterApplier,
		sortApplier:        sortApplier,
		findPreloadPaths:   findPreloadPaths,
		listPreloadPaths:   listPreloadPaths,
		authzFilterApplier: authzFilterApplier,
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
	entityValue := *entity
	db := r.db.WithContext(ctx)

	for _, path := range r.findPreloadPaths {
		db = db.Preload(path)
	}

	if r.authzFilterApplier != nil {
		if id := domain.GetAuthIdentity(ctx); id != nil {
			db = r.authzFilterApplier(id.Scope(), db)
		}
	}

	err := db.Take(entity, entityValue.TableName()+".id = ?", id).Error
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
		r.authzFilterApplier,
		r.listPreloadPaths,
	)
}

func (r *GormRepository[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	db := r.db.WithContext(ctx).Model(new(T))

	if r.authzFilterApplier != nil {
		if id := domain.GetAuthIdentity(ctx); id != nil {
			db = r.authzFilterApplier(id.Scope(), db)
		}
	}

	result := db.Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

func (r *GormRepository[T]) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	var exists bool
	entity := new(T)
	entityValue := *entity
	db := r.db.WithContext(ctx)

	if r.authzFilterApplier != nil {
		if authID := domain.GetAuthIdentity(ctx); authID != nil {
			db = r.authzFilterApplier(authID.Scope(), db)
		}
	}

	query := db.Select("1").
		Table(entityValue.TableName()).
		Where(entityValue.TableName()+".id = ?", id).
		Limit(1)

	err := query.Find(&exists).Error
	if err != nil {
		return false, err
	}

	return exists, nil
}
