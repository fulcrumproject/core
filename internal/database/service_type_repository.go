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

// NewServiceTypeRepository crea una nuova istanza di ServiceTypeRepository
func NewServiceTypeRepository(db *gorm.DB) domain.ServiceTypeRepository {
	return &serviceTypeRepository{db: db}
}

func (r *serviceTypeRepository) Create(ctx context.Context, serviceType *domain.ServiceType) error {
	if err := serviceType.Validate(); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Create(serviceType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceTypeRepository) Update(ctx context.Context, serviceType *domain.ServiceType) error {
	if err := serviceType.Validate(); err != nil {
		return err
	}

	// Prima verifichiamo che il ServiceType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.ServiceType{}, serviceType.ID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Save(serviceType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceTypeRepository) Delete(ctx context.Context, id domain.UUID) error {
	// Prima verifichiamo che il ServiceType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.ServiceType{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Delete(&domain.ServiceType{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceTypeRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
	var serviceType domain.ServiceType
	err := r.db.WithContext(ctx).First(&serviceType, id).Error
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

	query := r.db.WithContext(ctx)
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Find(&serviceTypes).Error; err != nil {
		return nil, err
	}

	return serviceTypes, nil
}

func (r *serviceTypeRepository) FindByAgentType(ctx context.Context, agentTypeID domain.UUID) ([]domain.ServiceType, error) {
	var serviceTypes []domain.ServiceType

	err := r.db.WithContext(ctx).
		Joins("JOIN agent_type_service_types atst ON atst.service_type_id = service_types.id").
		Where("atst.agent_type_id = ?", agentTypeID).
		Find(&serviceTypes).Error
	if err != nil {
		return nil, err
	}

	return serviceTypes, nil
}

func (r *serviceTypeRepository) UpdateResourceDefinitions(ctx context.Context, id domain.UUID, definitions domain.JSON) error {
	// Prima verifichiamo che il ServiceType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.ServiceType{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	// Aggiorniamo solo il campo resource_definitions
	result := r.db.WithContext(ctx).
		Model(&domain.ServiceType{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"resource_definitions": definitions,
			"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if result.Error != nil {
		return result.Error
	}

	return nil
}
