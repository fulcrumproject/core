package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type agentTypeRepository struct {
	db *gorm.DB
}

// NewAgentTypeRepository creates a new instance of AgentTypeRepository
func NewAgentTypeRepository(db *gorm.DB) domain.AgentTypeRepository {
	return &agentTypeRepository{db: db}
}

func (r *agentTypeRepository) Create(ctx context.Context, agentType *domain.AgentType) error {
	result := r.db.WithContext(ctx).Create(agentType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentTypeRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.AgentType, error) {
	var agentType domain.AgentType

	err := r.db.WithContext(ctx).
		Preload("ServiceTypes").
		First(&agentType, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &agentType, nil
}

var agentTypeFilterConfigs = map[string]FilterConfig{
	"name": {},
}

func (r *agentTypeRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.AgentType], error) {
	var agentTypes []domain.AgentType
	var totalItems int64

	query := r.db.WithContext(ctx).
		Preload("ServiceTypes").
		Model(&domain.AgentType{})
	query, err := applySimpleFilter(query, filter, agentTypeFilterConfigs)
	if err != nil {
		return nil, err
	}
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, domain.NewInternalError(err)
	}
	query, err = applySorting(query, sorting)
	if err != nil {
		return nil, err
	}
	query, err = applyPagination(query, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&agentTypes).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(agentTypes, totalItems, pagination), nil
}

func (r *agentTypeRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&domain.AgentType{})
	query, err := applySimpleFilter(query, filter, agentTypeFilterConfigs)
	if err != nil {
		return 0, err
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, domain.NewInternalError(err)
	}

	return count, nil
}
