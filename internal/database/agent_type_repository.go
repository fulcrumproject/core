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

func (r *agentTypeRepository) List(ctx context.Context, filters map[string]interface{}) ([]domain.AgentType, error) {
	var agentTypes []domain.AgentType

	query := r.db.WithContext(ctx)
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Find(&agentTypes).Error; err != nil {
		return nil, err
	}

	return agentTypes, nil
}

func (r *agentTypeRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&domain.AgentType{})
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
