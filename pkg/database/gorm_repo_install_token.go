package database

import (
	"context"
	"errors"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormInstallTokenRepository struct {
	*GormRepository[domain.InstallToken]
}

// NewInstallTokenRepository creates a new repository for install tokens.
// Install tokens are accessed 1:1 per (entity_type, entity_id), not listed,
// so List/Count/Exists on the embedded base repository are unused. The
// embedded Agent / Infrastructure on InstallToken is polymorphic (gorm:"-")
// — the repo hydrates exactly one of them after lookup by delegating to the
// owning entity's repository so its preloads stay in one place.
func NewInstallTokenRepository(db *gorm.DB) *GormInstallTokenRepository {
	return &GormInstallTokenRepository{
		GormRepository: NewGormRepository[domain.InstallToken](
			db,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
	}
}

func (r *GormInstallTokenRepository) GetByEntity(ctx context.Context, entityType domain.InstallTokenEntityType, entityID properties.UUID) (*domain.InstallToken, error) {
	var tok domain.InstallToken
	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		First(&tok).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	if err := r.hydrateEntity(ctx, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (r *GormInstallTokenRepository) FindByHashedToken(ctx context.Context, hashed string) (*domain.InstallToken, error) {
	var tok domain.InstallToken
	err := r.db.WithContext(ctx).
		Where("token_hashed = ?", hashed).
		First(&tok).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	if err := r.hydrateEntity(ctx, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (r *GormInstallTokenRepository) DeleteByEntity(ctx context.Context, entityType domain.InstallTokenEntityType, entityID properties.UUID) error {
	return r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Delete(&domain.InstallToken{}).Error
}

// hydrateEntity populates the matching pointer field on tok by calling the
// owning entity's repository — keeping each entity's preload chain authored in
// one place rather than re-declaring strings here.
func (r *GormInstallTokenRepository) hydrateEntity(ctx context.Context, tok *domain.InstallToken) error {
	switch tok.EntityType {
	case domain.InstallTokenEntityTypeAgent:
		agent, err := NewAgentRepository(r.db).Get(ctx, tok.EntityID)
		if err != nil {
			return err
		}
		tok.Agent = agent
	case domain.InstallTokenEntityTypeInfrastructure:
		infra, err := NewInfrastructureRepository(r.db).Get(ctx, tok.EntityID)
		if err != nil {
			return err
		}
		tok.Infrastructure = infra
	}
	return nil
}
