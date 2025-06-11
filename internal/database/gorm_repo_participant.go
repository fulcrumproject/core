package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormParticipantRepository struct {
	*GormRepository[domain.Participant]
}

var applyParticipantFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":   stringInFilterFieldApplier("name"),
	"status": parserInFilterFieldApplier("status", domain.ParseParticipantStatus),
})

var applyParticipantSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewParticipantRepository creates a new instance of ParticipantRepository
func NewParticipantRepository(db *gorm.DB) *GormParticipantRepository {
	repo := &GormParticipantRepository{
		GormRepository: NewGormRepository[domain.Participant](
			db,
			applyParticipantFilter,
			applyParticipantSort,
			participantAuthzFilterApplier,
			[]string{}, // Find preload paths - no specific preloads required for participants
			[]string{}, // List preload paths - no specific preloads required for participants
		),
	}
	return repo
}

// AuthScope returns the auth scope for the participant
func (r *GormParticipantRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	return r.getAuthScope(ctx, id, "id as participant_id")
}
