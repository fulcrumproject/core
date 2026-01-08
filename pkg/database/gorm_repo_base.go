package database

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type AuthzFilterApplier func(scope *auth.IdentityScope, db *gorm.DB) *gorm.DB

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

func (r *GormRepository[T]) Delete(ctx context.Context, id properties.UUID) error {
	result := r.db.WithContext(ctx).Delete(new(T), id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *GormRepository[T]) Get(ctx context.Context, id properties.UUID) (*T, error) {
	entity := new(T)
	entityValue := *entity
	db := r.db.WithContext(ctx)

	for _, path := range r.findPreloadPaths {
		db = db.Preload(path)
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

func (r *GormRepository[T]) List(ctx context.Context, authIdentityScope *auth.IdentityScope, page *domain.PageReq) (*domain.PageRes[T], error) {
	return listPaginated[T](
		ctx,
		r.db,
		page,
		r.filterApplier,
		r.sortApplier,
		r.authzFilterApplier,
		r.listPreloadPaths,
		authIdentityScope,
	)
}

func (r *GormRepository[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	db := r.db.WithContext(ctx).Model(new(T))

	result := db.Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

func (r *GormRepository[T]) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	var exists bool
	entity := new(T)
	entityValue := *entity
	db := r.db.WithContext(ctx)

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

func participantAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("participant_id = ?", s.ParticipantID)
	}
	return q
}

func participantSelfAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("id = ?", s.ParticipantID)
	}
	return q
}

func providerConsumerAgentAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("consumer_id = ? OR provider_id = ?", s.ParticipantID, s.ParticipantID)
	}
	if s.AgentID != nil {
		return q.Where("agent_id = ?", s.AgentID)
	}
	return q
}

// AuthScopeByFields retrieves auth scope for an entity with specified scope fields
func (r *GormRepository[T]) AuthScopeByFields(ctx context.Context, id properties.UUID, scopeFields ...string) (authz.ObjectScope, error) {
	var scope authz.DefaultObjectScope
	entity := new(T)
	entityValue := *entity

	slog.Info(entityValue.TableName())
	err := r.db.
		WithContext(ctx).
		Table(entityValue.TableName()).
		Select(scopeFields).
		Where("id = ?", id).
		Row().
		Scan(&scope.ParticipantID, &scope.ProviderID, &scope.AgentID, &scope.ConsumerID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &scope, nil
}
