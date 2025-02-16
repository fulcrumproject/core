package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type serviceTypeRepository struct {
	db *gorm.DB
}

// NewServiceTypeRepository creates a new instance of ServiceTypeRepository
func NewServiceTypeRepository(db *gorm.DB) domain.ServiceTypeRepository {
	return &serviceTypeRepository{db: db}
}

func (r *serviceTypeRepository) Create(ctx context.Context, serviceType *domain.ServiceType) error {
	result := r.db.WithContext(ctx).Create(serviceType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceTypeRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
	var serviceType domain.ServiceType
	err := r.db.WithContext(ctx).
		Preload("AgentTypes").
		First(&serviceType, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &serviceType, nil
}

func (r *serviceTypeRepository) List(ctx context.Context, filters map[string]interface{}) ([]domain.ServiceType, error) {
	var serviceTypes []domain.ServiceType

	query := r.db.WithContext(ctx).Preload("AgentTypes")
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Find(&serviceTypes).Error; err != nil {
		return nil, err
	}

	return serviceTypes, nil
}
