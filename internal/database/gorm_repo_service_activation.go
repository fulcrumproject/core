package database

import (
	"context"
	"errors"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormServiceActivationRepository struct {
	*GormRepository[domain.ServiceActivation]
}

var applyServiceActivationFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"provider_id":     uuidFilterFieldApplier("provider_id"),
	"service_type_id": uuidFilterFieldApplier("service_type_id"),
	"tag":             arrayContainsAllFilterFieldApplier("tags"),
})

var applyServiceActivationSort = mapSortApplier(map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
})

// NewServiceActivationRepository creates a new instance of ServiceActivationRepository
func NewServiceActivationRepository(db *gorm.DB) *GormServiceActivationRepository {
	repo := &GormServiceActivationRepository{
		GormRepository: NewGormRepository[domain.ServiceActivation](
			db,
			applyServiceActivationFilter,
			applyServiceActivationSort,
			serviceActivationAuthzFilterApplier,
			[]string{"Agents"}, // Preload agents for FindByID
			[]string{"Agents"}, // Preload agents for List
		),
	}
	return repo
}

func serviceActivationAuthzFilterApplier(s *domain.AuthIdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("provider_id = ?", s.ParticipantID)
	}
	return q
}

func (r *GormServiceActivationRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	return r.getAuthScope(ctx, id, "provider_id")
}

func (r *GormServiceActivationRepository) FindByServiceTypeAndTags(ctx context.Context, serviceTypeID domain.UUID, tags []string) ([]*domain.ServiceActivation, error) {
	var activations []*domain.ServiceActivation

	query := r.db.WithContext(ctx).Preload("Agents").Where("service_type_id = ?", serviceTypeID)

	if len(tags) > 0 {
		query = query.Where("tags @> ?", pq.StringArray(tags))
	}

	if err := query.Find(&activations).Error; err != nil {
		return nil, err
	}

	return activations, nil
}

func (r *GormServiceActivationRepository) FindByAgentAndServiceType(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID) (*domain.ServiceActivation, error) {
	var activation domain.ServiceActivation

	err := r.db.WithContext(ctx).
		Preload("Agents").
		Where("service_type_id = ?", serviceTypeID).
		Joins("JOIN service_activation_agents ON service_activations.id = service_activation_agents.service_activation_id").
		Where("service_activation_agents.agent_id = ?", agentID).
		First(&activation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: errors.New("service activation not found")}
		}
		return nil, err
	}

	return &activation, nil
}
