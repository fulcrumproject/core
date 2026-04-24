package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormAgentInstallTokenRepository struct {
	*GormRepository[domain.AgentInstallToken]
}

// NewAgentInstallTokenRepository creates a new repository for agent install tokens.
// Install tokens are accessed 1:1 per agent, not listed, so List/Count/Exists on the
// embedded base repository are unused. Custom methods below apply preloads explicitly.
func NewAgentInstallTokenRepository(db *gorm.DB) *GormAgentInstallTokenRepository {
	return &GormAgentInstallTokenRepository{
		GormRepository: NewGormRepository[domain.AgentInstallToken](
			db,
			nil,
			nil,
			nil,
			[]string{"Agent.AgentType"},
			[]string{"Agent.AgentType"},
		),
	}
}

func (r *GormAgentInstallTokenRepository) GetByAgentID(ctx context.Context, agentID properties.UUID) (*domain.AgentInstallToken, error) {
	var tok domain.AgentInstallToken
	err := r.db.WithContext(ctx).
		Preload("Agent.AgentType").
		Where("agent_id = ?", agentID).
		First(&tok).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &tok, nil
}

func (r *GormAgentInstallTokenRepository) FindByHashedToken(ctx context.Context, hashed string) (*domain.AgentInstallToken, error) {
	var tok domain.AgentInstallToken
	err := r.db.WithContext(ctx).
		Preload("Agent.AgentType").
		Where("token_hashed = ?", hashed).
		First(&tok).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &tok, nil
}

func (r *GormAgentInstallTokenRepository) DeleteByAgentID(ctx context.Context, agentID properties.UUID) error {
	return r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Delete(&domain.AgentInstallToken{}).Error
}
