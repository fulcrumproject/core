package database

import (
	"context"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"gorm.io/gorm"

	"fulcrumproject.org/core/pkg/domain"
)

type GormEventRepository struct {
	*GormRepository[domain.Event]
}

var applyEventFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"initiatorType": stringInFilterFieldApplier("initiator_type"),
	"initiatorId":   parserInFilterFieldApplier("initiator_id", properties.ParseUUID),
	"type":          stringInFilterFieldApplier("type"),
})

var applyEventSort = mapSortApplier(map[string]string{
	"createdAt": "created_at",
})

// NewEventRepository creates a new instance of EventRepository
func NewEventRepository(db *gorm.DB) *GormEventRepository {
	repo := &GormEventRepository{
		GormRepository: NewGormRepository[domain.Event](
			db,
			applyEventFilter,
			applyEventSort,
			providerConsumerAgentAuthzFilterApplier,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// ListFromSequence retrieves events starting from a specific sequence number
func (r *GormEventRepository) ListFromSequence(ctx context.Context, fromSequenceNumber int64, limit int) ([]*domain.Event, error) {
	var events []*domain.Event
	result := r.db.WithContext(ctx).
		Where("sequence_number > ?", fromSequenceNumber).
		Order("sequence_number ASC").
		Limit(limit).
		Find(&events)

	if result.Error != nil {
		return nil, result.Error
	}

	return events, nil
}

func (r *GormEventRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.getAuthScope(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}
