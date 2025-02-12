package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository creates a new instance of AgentRepository
func NewAgentRepository(db *gorm.DB) domain.AgentRepository {
	return &agentRepository{db: db}
}

func (r *agentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	if err := agent.Validate(); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Create(agent)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentRepository) Update(ctx context.Context, agent *domain.Agent) error {
	if err := agent.Validate(); err != nil {
		return err
	}

	// First verify that the Agent exists
	exists := r.db.WithContext(ctx).Select("id").First(&domain.Agent{}, agent.ID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Save(agent)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentRepository) Delete(ctx context.Context, id domain.UUID) error {
	// First verify that the Agent exists
	exists := r.db.WithContext(ctx).Select("id").First(&domain.Agent{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Delete(&domain.Agent{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
	var agent domain.Agent
	err := r.db.WithContext(ctx).First(&agent, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &agent, nil
}

func (r *agentRepository) List(ctx context.Context, filters map[string]interface{}) ([]domain.Agent, error) {
	var agents []domain.Agent

	query := r.db.WithContext(ctx)
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Find(&agents).Error; err != nil {
		return nil, err
	}

	return agents, nil
}

func (r *agentRepository) FindByProvider(ctx context.Context, providerID domain.UUID) ([]domain.Agent, error) {
	var agents []domain.Agent

	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Find(&agents).Error
	if err != nil {
		return nil, err
	}

	return agents, nil
}

func (r *agentRepository) FindByAgentType(ctx context.Context, agentTypeID domain.UUID) ([]domain.Agent, error) {
	var agents []domain.Agent

	err := r.db.WithContext(ctx).
		Where("agent_type_id = ?", agentTypeID).
		Find(&agents).Error
	if err != nil {
		return nil, err
	}

	return agents, nil
}

func (r *agentRepository) UpdateState(ctx context.Context, id domain.UUID, state domain.AgentState) error {
	// First verify that the Agent exists
	exists := r.db.WithContext(ctx).Select("id").First(&domain.Agent{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).
		Model(&domain.Agent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":      state,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if result.Error != nil {
		return result.Error
	}

	return nil
}
