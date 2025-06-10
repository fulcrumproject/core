package database

import (
	"context"

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
