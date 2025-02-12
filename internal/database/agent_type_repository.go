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

// NewAgentTypeRepository crea una nuova istanza di AgentTypeRepository
func NewAgentTypeRepository(db *gorm.DB) domain.AgentTypeRepository {
	return &agentTypeRepository{db: db}
}

func (r *agentTypeRepository) Create(ctx context.Context, agentType *domain.AgentType) error {
	if err := agentType.Validate(); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Create(agentType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentTypeRepository) Update(ctx context.Context, agentType *domain.AgentType) error {
	if err := agentType.Validate(); err != nil {
		return err
	}

	// Prima verifichiamo che l'AgentType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.AgentType{}, agentType.ID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Save(agentType)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentTypeRepository) Delete(ctx context.Context, id domain.UUID) error {
	// Prima verifichiamo che l'AgentType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.AgentType{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Delete(&domain.AgentType{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentTypeRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.AgentType, error) {
	var agentType domain.AgentType
	err := r.db.WithContext(ctx).First(&agentType, id).Error
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

func (r *agentTypeRepository) FindByServiceType(ctx context.Context, serviceTypeID domain.UUID) ([]domain.AgentType, error) {
	var agentTypes []domain.AgentType

	err := r.db.WithContext(ctx).
		Joins("JOIN agent_type_service_types atst ON atst.agent_type_id = agent_types.id").
		Where("atst.service_type_id = ?", serviceTypeID).
		Find(&agentTypes).Error
	if err != nil {
		return nil, err
	}

	return agentTypes, nil
}

func (r *agentTypeRepository) AddServiceType(ctx context.Context, agentTypeID, serviceTypeID domain.UUID) error {
	// Prima verifichiamo che l'AgentType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.AgentType{}, agentTypeID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	// Verify that the ServiceType exists
	exists = r.db.WithContext(ctx).Select("id").First(&domain.ServiceType{}, serviceTypeID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO agent_type_service_types (agent_type_id, service_type_id)
		VALUES (?, ?)
	`, agentTypeID, serviceTypeID)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *agentTypeRepository) RemoveServiceType(ctx context.Context, agentTypeID, serviceTypeID domain.UUID) error {
	// Prima verifichiamo che l'AgentType esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.AgentType{}, agentTypeID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Exec(`
		DELETE FROM agent_type_service_types
		WHERE agent_type_id = ? AND service_type_id = ?
	`, agentTypeID, serviceTypeID)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
