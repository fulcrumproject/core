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
	"name":        stringInFilterFieldApplier("name"),
	"state":       parserInFilterFieldApplier("state", domain.ParseParticipantState),
	"countryCode": parserInFilterFieldApplier("country_code", domain.ParseCountryCode),
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

// participantAuthzFilterApplier applies authorization scoping to participant queries
func participantAuthzFilterApplier(s *domain.AuthScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("id = ?", s.ParticipantID)
	}
	// Allow full access for fulcrum admins
	return q
}

// AuthScope returns the auth scope for the participant
func (r *GormParticipantRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	return r.getAuthScope(ctx, id, "participant_id")
}
