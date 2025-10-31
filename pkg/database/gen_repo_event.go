// Event repository implementation using GORM Gen
// Provides type-safe database operations for Event entities
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenEventRepository struct {
	q *Query
}

func NewGenEventRepository(db *gorm.DB) *GenEventRepository {
	return &GenEventRepository{q: Use(db)}
}

func (r *GenEventRepository) Create(ctx context.Context, entity *domain.Event) error {
	return r.q.Event.WithContext(ctx).Create(entity)
}

func (r *GenEventRepository) Save(ctx context.Context, entity *domain.Event) error {
	result, err := r.q.Event.WithContext(ctx).Where(r.q.Event.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenEventRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.Event.WithContext(ctx).Where(r.q.Event.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenEventRepository) Get(ctx context.Context, id properties.UUID) (*domain.Event, error) {
	entity, err := r.q.Event.WithContext(ctx).Where(r.q.Event.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenEventRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.Event.WithContext(ctx).Where(r.q.Event.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenEventRepository) Count(ctx context.Context) (int64, error) {
	return r.q.Event.WithContext(ctx).Count()
}

func (r *GenEventRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.Event], error) {
	query := r.q.Event.WithContext(ctx)
	query = applyGenEventAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenEventFilters,
		applyGenEventSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Event, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.Event]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenEventRepository) ListFromSequence(ctx context.Context, fromSequenceNumber int64, limit int) ([]*domain.Event, error) {
	return r.q.Event.WithContext(ctx).
		Where(r.q.Event.SequenceNumber.Gt(fromSequenceNumber)).
		Order(r.q.Event.SequenceNumber).
		Limit(limit).
		Find()
}

func (r *GenEventRepository) ServiceUptime(ctx context.Context, serviceID properties.UUID, start time.Time, end time.Time) (uptimeSeconds uint64, downtimeSeconds uint64, err error) {
	// Use UnderlyingDB for complex uptime calculation queries
	db := r.q.Event.WithContext(ctx).UnderlyingDB()
	
	var service domain.Service
	if err := db.Where("id = ?", serviceID).First(&service).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to load service: %w", err)
	}

	var serviceType domain.ServiceType
	if err := db.Where("id = ?", service.ServiceTypeID).First(&serviceType).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to load service type: %w", err)
	}

	currentStatus, err := r.getServiceStatusAtTime(ctx, serviceID, start)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get initial service status: %w", err)
	}

	var event domain.Event
	rows, err := db.
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

	var totalUptime time.Duration
	totalDuration := end.Sub(start)
	currentTime := start
	hasEvents := false

	for rows.Next() {
		hasEvents = true
		var event domain.Event
		if err := db.ScanRows(rows, &event); err != nil {
			return 0, 0, fmt.Errorf("failed to scan event row: %w", err)
		}

		eventTime := event.CreatedAt
		if serviceType.LifecycleSchema.IsRunningStatus(currentStatus) {
			totalUptime += eventTime.Sub(currentTime)
		}

		newStatus, err := r.extractServiceStatusFromEvent(&event)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to extract service status from event %s: %w", event.ID, err)
		}

		currentStatus = newStatus
		currentTime = eventTime
	}

	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("error iterating through event rows: %w", err)
	}

	if !hasEvents {
		currentTime = start
	}

	if serviceType.LifecycleSchema.IsRunningStatus(currentStatus) {
		totalUptime += end.Sub(currentTime)
	}

	uptimeSeconds = uint64(totalUptime.Seconds())
	downtimeSeconds = uint64(totalDuration.Seconds()) - uptimeSeconds
	return uptimeSeconds, downtimeSeconds, nil
}

func (r *GenEventRepository) getServiceStatusAtTime(ctx context.Context, serviceID properties.UUID, t time.Time) (string, error) {
	db := r.q.Event.WithContext(ctx).UnderlyingDB()
	var event domain.Event
	err := db.
		Where("entity_id = ?", serviceID).
		Where("type = ?", domain.EventTypeServiceTransitioned).
		Where("created_at <= ?", t).
		Order("created_at DESC").
		First(&event).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			var service domain.Service
			if err := db.Where("id = ?", serviceID).First(&service).Error; err != nil {
				return "", err
			}
			return service.Status, nil
		}
		return "", err
	}

	return r.extractServiceStatusFromEvent(&event)
}

func (r *GenEventRepository) extractServiceStatusFromEvent(event *domain.Event) (string, error) {
	statusVal, ok := event.Payload["status"]
	if !ok {
		return "", fmt.Errorf("status not found in event payload")
	}
	status, ok := statusVal.(string)
	if !ok {
		return "", fmt.Errorf("status is not a string")
	}
	return status, nil
}

func (r *GenEventRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	entity, err := r.q.Event.WithContext(ctx).
		Select(r.q.Event.ProviderID, r.q.Event.AgentID, r.q.Event.ConsumerID).
		Where(r.q.Event.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &auth.DefaultObjectScope{
		ProviderID: entity.ProviderID,
		AgentID:    entity.AgentID,
		ConsumerID: entity.ConsumerID,
	}, nil
}

func applyGenEventAuthz(query IEventDo, scope *auth.IdentityScope) IEventDo {
	q := Use(nil).Event
	if scope.ParticipantID != nil {
		return query.Where(q.ConsumerID.Eq(*scope.ParticipantID)).
			Or(q.ProviderID.Eq(*scope.ParticipantID))
	}
	if scope.AgentID != nil {
		return query.Where(q.AgentID.Eq(*scope.AgentID))
	}
	return query
}

func applyGenEventFilters(query IEventDo, pageReq *domain.PageReq) IEventDo {
	q := Use(nil).Event
	if values, ok := pageReq.Filters["initiatorType"]; ok && len(values) > 0 {
		query = query.Where(q.InitiatorType.In(values...))
	}
	if values, ok := pageReq.Filters["initiatorId"]; ok && len(values) > 0 {
		query = query.Where(q.InitiatorID.In(values...))
	}
	if values, ok := pageReq.Filters["type"]; ok && len(values) > 0 {
		query = query.Where(q.Type.In(values...))
	}
	return query
}

func applyGenEventSort(query IEventDo, pageReq *domain.PageReq) IEventDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).Event
	if pageReq.SortBy == "createdAt" {
		if pageReq.SortAsc {
			return query.Order(q.CreatedAt)
		}
		return query.Order(q.CreatedAt.Desc())
	}
	return query
}

