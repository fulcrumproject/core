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
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &agent, nil
}

func (r *agentRepository) List(ctx context.Context, filters domain.Filters, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.Agent], error) {
	var agents []domain.Agent
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.Agent{})
	query = applyFilters(query, filters)
	// Get total count for pagination
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, err
	}
	query = applySorting(query, sorting)
	query = applyPagination(query, pagination)
	if err := query.Preload("Provider").Preload("AgentType").Find(&agents).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(agents, totalItems, pagination), nil
}

func (r *agentRepository) Count(ctx context.Context, filters domain.Filters) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&domain.Agent{})
	query = applyFilters(query, filters)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
