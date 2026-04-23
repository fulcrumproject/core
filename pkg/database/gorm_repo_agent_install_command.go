package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormAgentInstallCommandRepository struct {
	*GormRepository[domain.AgentInstallCommand]
}

// NewAgentInstallCommandRepository creates a new repository for agent install commands.
// Install commands are accessed 1:1 per agent, not listed, so List/Count/Exists on the
// embedded base repository are unused. Custom methods below apply preloads explicitly.
func NewAgentInstallCommandRepository(db *gorm.DB) *GormAgentInstallCommandRepository {
	return &GormAgentInstallCommandRepository{
		GormRepository: NewGormRepository[domain.AgentInstallCommand](
			db,
			nil,
			nil,
			nil,
			[]string{"Agent.AgentType"},
			[]string{"Agent.AgentType"},
		),
	}
}

func (r *GormAgentInstallCommandRepository) GetByAgentID(ctx context.Context, agentID properties.UUID) (*domain.AgentInstallCommand, error) {
	var cmd domain.AgentInstallCommand
	err := r.db.WithContext(ctx).
		Preload("Agent.AgentType").
		Where("agent_id = ?", agentID).
		First(&cmd).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &cmd, nil
}

func (r *GormAgentInstallCommandRepository) FindByHashedToken(ctx context.Context, hashed string) (*domain.AgentInstallCommand, error) {
	var cmd domain.AgentInstallCommand
	err := r.db.WithContext(ctx).
		Preload("Agent.AgentType").
		Where("token_hashed = ?", hashed).
		First(&cmd).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &cmd, nil
}

func (r *GormAgentInstallCommandRepository) DeleteByAgentID(ctx context.Context, agentID properties.UUID) error {
	return r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Delete(&domain.AgentInstallCommand{}).Error
}
