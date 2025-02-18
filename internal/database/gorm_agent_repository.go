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
	result := r.db.WithContext(ctx).Create(agent)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentRepository) Save(ctx context.Context, agent *domain.Agent) error {
	result := r.db.WithContext(ctx).Save(agent)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentRepository) Delete(ctx context.Context, id domain.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Agent{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
	var agent domain.Agent

	err := r.db.WithContext(ctx).Preload("Provider").Preload("AgentType").First(&agent, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &agent, nil
}

var agentFilterConfigs = map[string]FilterConfig{
	"name":        {},
	"state":       {Query: "state", Valuer: func(v string) (interface{}, error) { return domain.ParseAgentState(v) }},
	"countryCode": {Query: "country_code", Valuer: func(v string) (interface{}, error) { return domain.ParseCountryCode(v) }},
	"providerId":  {Query: "provider_id", Valuer: func(v string) (interface{}, error) { return domain.ParseID(v) }},
	"agentTypeId": {Query: "agent_type_id", Valuer: func(v string) (interface{}, error) { return domain.ParseID(v) }},
}

func (r *agentRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.Agent], error) {
	var agents []domain.Agent

	query := r.db.WithContext(ctx).Model(&domain.Agent{})
	query, totalItems, err := applyFindAndCount(query, filter, agentFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Preload("Provider").Preload("AgentType").Find(&agents).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(agents, totalItems, pagination), nil
}

func (r *agentRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Agent{})
	_, count, err := applyFilterAndCount(query, filter, agentFilterConfigs)
	if err != nil {
		return 0, err
	}
	return count, nil
}
