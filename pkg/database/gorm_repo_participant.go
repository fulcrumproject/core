package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormParticipantRepository struct {
	*GormRepository[domain.Participant]
}

var applyParticipantFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":   StringContainsInsensitiveFilterFieldApplier("name"),
	"status": ParserInFilterFieldApplier("status", domain.ParseParticipantStatus),
})

var applyParticipantSort = MapSortApplier(map[string]string{
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
func (r *GormParticipantRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "id as participant_id", "null", "null", "null")
}
