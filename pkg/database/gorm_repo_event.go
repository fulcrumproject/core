package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormEventRepository struct {
	*GormRepository[domain.Event]
}

var applyEventFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"initiatorType": StringInFilterFieldApplier("initiator_type"),
	"initiatorId":   ParserInFilterFieldApplier("initiator_id", properties.ParseUUID),
	"type":          StringContainsInsensitiveFilterFieldApplier("type"),
})

var applyEventSort = MapSortApplier(map[string]string{
	"createdAt":      "created_at",
	"sequenceNumber": "sequence_number",
})

// NewEventRepository creates a new instance of EventRepository
func NewEventRepository(db *gorm.DB) *GormEventRepository {
	repo := &GormEventRepository{
		GormRepository: NewGormRepository[domain.Event](
			db,
			applyEventFilter,
			applyEventSort,
			providerConsumerAgentAuthzFilterApplier,
			[]string{"Participant", "Provider", "Agent", "Consumer"},
			[]string{"Participant", "Provider", "Agent", "Consumer"},
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

func (r *GormEventRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}

// ServiceUptime returns the uptime and downtime in seconds of a service in a given time range
// It streams service transition events and calculates uptime progressively using a result set
func (r *GormEventRepository) ServiceUptime(ctx context.Context, serviceID properties.UUID, start time.Time, end time.Time) (uptimeSeconds uint64, downtimeSeconds uint64, err error) {
	// Load service and its lifecycle schema for uptime calculation
	var service domain.Service
	if err := r.db.WithContext(ctx).Where("id = ?", serviceID).First(&service).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to load service: %w", err)
	}

	var serviceType domain.ServiceType
	if err := r.db.WithContext(ctx).Where("id = ?", service.ServiceTypeID).First(&serviceType).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to load service type: %w", err)
	}

	// Get initial status at the start of the time range
	currentStatus, err := r.getServiceStatusAtTime(ctx, serviceID, start)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get initial service status: %w", err)
	}

	// Load all in-window service transition events. We don't stream with
	// Rows/ScanRows because ScanRows mutates the shared *gorm.DB statement and
	// leaks the events table into other callers using the same DB instance.
	var events []domain.Event
	if err := r.db.WithContext(ctx).
		Where("entity_id = ?", serviceID).
		Where("type = ?", domain.EventTypeServiceTransitioned).
		Where("created_at >= ?", start).
		Where("created_at <= ?", end).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to query service transition events: %w", err)
	}

	var totalUptime time.Duration
	totalDuration := end.Sub(start)
	currentTime := start
	hasEvents := false

	for i := range events {
		event := &events[i]

		// service.transitioned events are emitted on every job completion (see
		// jobCommander), so the diff may carry only property/heartbeat changes
		// with no /status patch — those events represent no actual status change
		// and must be skipped.
		newStatus, hasStatus, err := r.extractServiceStatusFromEvent(event)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to extract service status from event %s: %w", event.ID, err)
		}
		if !hasStatus {
			continue
		}

		hasEvents = true
		eventTime := event.CreatedAt

		// Add uptime for the period before this event if service was running
		if serviceType.LifecycleSchema.IsRunningStatus(currentStatus) {
			totalUptime += eventTime.Sub(currentTime)
		}

		currentStatus = newStatus
		currentTime = eventTime
	}

	// If no transition events found within the range, check if service was running the entire period
	if !hasEvents {
		if serviceType.LifecycleSchema.IsRunningStatus(currentStatus) {
			return uint64(totalDuration.Seconds()), 0, nil
		}
		return 0, uint64(totalDuration.Seconds()), nil
	}

	// Handle the final period from last event to end time
	if serviceType.LifecycleSchema.IsRunningStatus(currentStatus) {
		totalUptime += end.Sub(currentTime)
	}

	// Calculate uptime and downtime in seconds
	uptimeSeconds = uint64(totalUptime.Seconds())
	downtimeSeconds = uint64(totalDuration.Seconds()) - uptimeSeconds

	return uptimeSeconds, downtimeSeconds, nil
}

// getServiceStatusAtTime retrieves the service status at a specific point in
// time by looking for the most recent prior service.transitioned event that
// actually carries a /status patch. Property-only events (heartbeats / property
// updates emitted by jobCommander) are filtered out at the DB level via a
// JSONB containment predicate, so no in-memory walk-back is needed.
func (r *GormEventRepository) getServiceStatusAtTime(ctx context.Context, serviceID properties.UUID, timestamp time.Time) (string, error) {
	var event domain.Event
	err := r.db.WithContext(ctx).
		Where("entity_id = ?", serviceID).
		Where("type = ?", domain.EventTypeServiceTransitioned).
		Where("created_at <= ?", timestamp).
		Where("payload->'diff' @> ?::jsonb", `[{"path":"/status"}]`).
		Order("created_at DESC").
		First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	status, _, err := r.extractServiceStatusFromEvent(&event)
	if err != nil {
		return "", fmt.Errorf("failed to extract status from event: %w", err)
	}
	return status, nil
}

// extractServiceStatusFromEvent extracts the service status from a service transition event.
//
// service.transitioned events are emitted on every job completion with a diff
// of all changed fields, not only status — see jobCommander.Complete/Fail. An
// event without a /status patch means "no status change at this point in time"
// (e.g. heartbeat or property-only update) and is signalled by hasStatus=false
// with a nil error so callers can skip it without aborting.
func (r *GormEventRepository) extractServiceStatusFromEvent(event *domain.Event) (status string, hasStatus bool, err error) {
	if event.Payload == nil {
		return "", false, fmt.Errorf("event payload is nil")
	}

	diffInterface, exists := event.Payload["diff"]
	if !exists {
		return "", false, fmt.Errorf("no diff found in event payload")
	}

	patchesSlice, ok := diffInterface.([]any)
	if !ok {
		return "", false, fmt.Errorf("diff is not in expected array format, got type: %T", diffInterface)
	}

	for _, patchInterface := range patchesSlice {
		patch, ok := patchInterface.(map[string]any)
		if !ok {
			continue
		}
		if op, ok := patch["op"].(string); ok && (op == "replace" || op == "add") {
			if path, ok := patch["path"].(string); ok && path == "/status" {
				if value, ok := patch["value"].(string); ok {
					return value, true, nil
				}
			}
		}
	}

	return "", false, nil
}
