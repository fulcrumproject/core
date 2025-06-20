package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormEventSubscriptionRepository struct {
	*GormRepository[domain.EventSubscription]
}

var applyEventSubscriptionFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"subscriber_id": stringInFilterFieldApplier("subscriber_id"),
	"is_active":     parserInFilterFieldApplier("is_active", parseBool),
})

// parseBool parses a string to boolean
func parseBool(s string) (bool, error) {
	switch s {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}

var applyEventSubscriptionSort = mapSortApplier(map[string]string{
	"subscriber_id":                 "subscriber_id",
	"last_event_sequence_processed": "last_event_sequence_processed",
	"lease_expires_at":              "lease_expires_at",
	"created_at":                    "created_at",
	"updated_at":                    "updated_at",
})

// eventSubscriptionAuthzFilterApplier applies authorization filters for event subscriptions
// For now, we'll allow access to all event subscriptions (system-level resource)
func eventSubscriptionAuthzFilterApplier(scope *auth.IdentityScope, db *gorm.DB) *gorm.DB {
	// Event subscriptions are system-level resources
	// Only fulcrum_admin should have access, but we'll implement basic filtering here
	return db
}

// NewEventSubscriptionRepository creates a new instance of EventSubscriptionRepository
func NewEventSubscriptionRepository(db *gorm.DB) *GormEventSubscriptionRepository {
	repo := &GormEventSubscriptionRepository{
		GormRepository: NewGormRepository[domain.EventSubscription](
			db,
			applyEventSubscriptionFilter,
			applyEventSubscriptionSort,
			eventSubscriptionAuthzFilterApplier,
			[]string{}, // Find preload paths - no specific preloads required
			[]string{}, // List preload paths - no specific preloads required
		),
	}
	return repo
}

// FindBySubscriberID retrieves an event subscription by subscriber ID
func (r *GormEventSubscriptionRepository) FindBySubscriberID(ctx context.Context, subscriberID string) (*domain.EventSubscription, error) {
	var subscription domain.EventSubscription
	result := r.db.WithContext(ctx).Where("subscriber_id = ?", subscriberID).First(&subscription)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundErrorf("event subscription with subscriber_id %s", subscriberID)
		}
		return nil, result.Error
	}
	return &subscription, nil
}

// ExistsBySubscriberID checks if an event subscription exists by subscriber ID
func (r *GormEventSubscriptionRepository) ExistsBySubscriberID(ctx context.Context, subscriberID string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.EventSubscription{}).Where("subscriber_id = ?", subscriberID).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// DeleteBySubscriberID removes an event subscription by subscriber ID
func (r *GormEventSubscriptionRepository) DeleteBySubscriberID(ctx context.Context, subscriberID string) error {
	result := r.db.WithContext(ctx).Where("subscriber_id = ?", subscriberID).Delete(&domain.EventSubscription{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.NewNotFoundErrorf("event subscription with subscriber_id %s", subscriberID)
	}
	return nil
}

// ListExpiredLeases retrieves subscriptions with expired leases
func (r *GormEventSubscriptionRepository) ListExpiredLeases(ctx context.Context) ([]*domain.EventSubscription, error) {
	var subscriptions []*domain.EventSubscription
	result := r.db.WithContext(ctx).
		Where("lease_expires_at IS NOT NULL AND lease_expires_at < NOW()").
		Find(&subscriptions)
	if result.Error != nil {
		return nil, result.Error
	}
	return subscriptions, nil
}

// AuthScope returns the auth scope for the event subscription
func (r *GormEventSubscriptionRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Event subscriptions are system-level resources, no specific participant scope
	return &auth.AllwaysMatchObjectScope{}, nil
}
