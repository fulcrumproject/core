package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
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
	"type":          StringInFilterFieldApplier("type"),
})

var applyEventSort = MapSortApplier(map[string]string{
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
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}

// ServiceUptime returns the uptime and downtime in seconds of a service in a given time range
// It streams service transition events and calculates uptime progressively using a result set
func (r *GormEventRepository) ServiceUptime(ctx context.Context, serviceID properties.UUID, start time.Time, end time.Time) (uptimeSeconds uint64, downtimeSeconds uint64, err error) {
	// Get initial status at the start of the time range
	currentStatus, err := r.getServiceStatusAtTime(ctx, serviceID, start)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get initial service status: %w", err)
	}

	// Stream through service transition events within the time range using Rows()
	// Use a fresh session to avoid inheriting conditions from previous queries

	var event domain.Event
	rows, err := r.db.WithContext(ctx).
		Model(&event).
		Where("entity_id = ?", serviceID).
		Where("type = ?", domain.EventTypeServiceTransitioned).
		Where("created_at >= ?", start).
		Where("created_at <= ?", end).
		Order("created_at ASC").
		Rows()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query service transition events: %w", err)
	}
	defer rows.Close()

	// Calculate uptime progressively by streaming through events
	var totalUptime time.Duration
	totalDuration := end.Sub(start)
	currentTime := start
	hasEvents := false

	// Stream through each event
	for rows.Next() {
		hasEvents = true
		var event domain.Event

		// Scan the row into the event struct
		if err := r.db.ScanRows(rows, &event); err != nil {
			return 0, 0, fmt.Errorf("failed to scan event row: %w", err)
		}

		eventTime := event.CreatedAt

		// Add uptime for the period before this event if service was running
		if r.isRunningStatus(currentStatus) {
			totalUptime += eventTime.Sub(currentTime)
		}

		// Extract new status from event payload
		newStatus, err := r.extractServiceStatusFromEvent(&event)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to extract service status from event %s: %w", event.ID, err)
		}

		currentStatus = newStatus
		currentTime = eventTime
	}

	// Check for any errors that occurred during iteration
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("error occurred while iterating through events: %w", err)
	}

	// If no transition events found within the range, check if service was running the entire period
	if !hasEvents {
		if r.isRunningStatus(currentStatus) {
			return uint64(totalDuration.Seconds()), 0, nil
		}
		return 0, uint64(totalDuration.Seconds()), nil
	}

	// Handle the final period from last event to end time
	if r.isRunningStatus(currentStatus) {
		totalUptime += end.Sub(currentTime)
	}

	// Calculate uptime and downtime in seconds
	uptimeSeconds = uint64(totalUptime.Seconds())
	downtimeSeconds = uint64(totalDuration.Seconds()) - uptimeSeconds

	return uptimeSeconds, downtimeSeconds, nil
}

// getServiceStatusAtTime retrieves the service status at a specific point in time
// by looking for the most recent transition event before that time
func (r *GormEventRepository) getServiceStatusAtTime(ctx context.Context, serviceID properties.UUID, timestamp time.Time) (domain.ServiceStatus, error) {
	var event domain.Event
	result := r.db.WithContext(ctx).
		Where("entity_id = ?", serviceID).
		Where("type = ?", domain.EventTypeServiceTransitioned).
		Where("created_at <= ?", timestamp).
		Order("created_at DESC").
		First(&event)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// No transition events found before this time, assume service was in Created state
			return domain.ServiceCreated, nil
		}
		return "", fmt.Errorf("failed to query service status: %w", result.Error)
	}

	// Extract status from the event
	status, err := r.extractServiceStatusFromEvent(&event)
	if err != nil {
		return "", fmt.Errorf("failed to extract status from event: %w", err)
	}

	return status, nil
}

// extractServiceStatusFromEvent extracts the service status from a service transition event
func (r *GormEventRepository) extractServiceStatusFromEvent(event *domain.Event) (domain.ServiceStatus, error) {
	// The event payload contains a diff with the service status change
	// We need to extract the "after" state of the currentStatus field

	if event.Payload == nil {
		return "", fmt.Errorf("event payload is nil")
	}

	// Parse the diff from the payload
	diffInterface, exists := event.Payload["diff"]
	if !exists {
		return "", fmt.Errorf("no diff found in event payload")
	}

	// Type cast diff to JSON patches (optimized - avoids marshal/unmarshal)
	patchesSlice, ok := diffInterface.([]any)
	if !ok {
		return "", fmt.Errorf("diff is not in expected array format, got type: %T", diffInterface)
	}

	// Look for patches that modify the currentStatus field
	for _, patchInterface := range patchesSlice {
		patch, ok := patchInterface.(map[string]any)
		if !ok {
			continue // Skip invalid patch entries
		}

		if op, ok := patch["op"].(string); ok && (op == "replace" || op == "add") {
			if path, ok := patch["path"].(string); ok && path == "/currentStatus" {
				if value, ok := patch["value"].(string); ok {
					status := domain.ServiceStatus(value)
					if err := status.Validate(); err != nil {
						return "", fmt.Errorf("invalid service status in event: %w", err)
					}
					return status, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no currentStatus change found in event diff")
}

// isRunningStatus determines if a service status represents a "running" state for uptime calculation
func (r *GormEventRepository) isRunningStatus(status domain.ServiceStatus) bool {
	switch status {
	case domain.ServiceStarted:
		return true
	case domain.ServiceHotUpdating:
		// Service is still considered running during hot updates
		return true
	default:
		return false
	}
}
