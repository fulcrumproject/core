package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
)

// EventSubscription represents a subscription for external systems to consume events
type EventSubscription struct {
	BaseEntity

	SubscriberID               string     `json:"subscriber_id" gorm:"not null;uniqueIndex"`
	LastEventSequenceProcessed int64      `json:"last_event_sequence_processed" gorm:"not null;default:0"`
	LeaseOwnerInstanceID       *string    `json:"lease_owner_instance_id,omitempty" gorm:"index"`
	LeaseAcquiredAt            *time.Time `json:"lease_acquired_at,omitempty"`
	LeaseExpiresAt             *time.Time `json:"lease_expires_at,omitempty" gorm:"index"`
	IsActive                   bool       `json:"is_active" gorm:"not null;default:true"`
}

// NewEventSubscription creates a new EventSubscription without validation
func NewEventSubscription(subscriberID string) *EventSubscription {
	return &EventSubscription{
		SubscriberID:               subscriberID,
		LastEventSequenceProcessed: 0,
		IsActive:                   true,
	}
}

// TableName returns the table name for the event subscription
func (EventSubscription) TableName() string {
	return "event_subscriptions"
}

// Validate ensures all EventSubscription fields are valid
func (es *EventSubscription) Validate() error {
	if es.SubscriberID == "" {
		return fmt.Errorf("subscriber_id cannot be empty")
	}
	if es.LastEventSequenceProcessed < 0 {
		return fmt.Errorf("last_event_sequence_processed cannot be negative")
	}
	// Validate lease consistency
	if es.LeaseOwnerInstanceID != nil {
		if es.LeaseAcquiredAt == nil {
			return fmt.Errorf("lease_acquired_at must be set when lease_owner_instance_id is set")
		}
		if es.LeaseExpiresAt == nil {
			return fmt.Errorf("lease_expires_at must be set when lease_owner_instance_id is set")
		}
		if es.LeaseExpiresAt.Before(*es.LeaseAcquiredAt) {
			return fmt.Errorf("lease_expires_at must be after lease_acquired_at")
		}
	} else {
		if es.LeaseAcquiredAt != nil || es.LeaseExpiresAt != nil {
			return fmt.Errorf("lease_acquired_at and lease_expires_at must be nil when lease_owner_instance_id is nil")
		}
	}
	return nil
}

// Update updates the event subscription fields if the pointers are non-nil
func (es *EventSubscription) Update(
	lastEventSequenceProcessed *int64,
	leaseOwnerInstanceID *string,
	leaseAcquiredAt *time.Time,
	leaseExpiresAt *time.Time,
	isActive *bool,
) {
	if lastEventSequenceProcessed != nil {
		es.LastEventSequenceProcessed = *lastEventSequenceProcessed
	}
	if leaseOwnerInstanceID != nil {
		es.LeaseOwnerInstanceID = leaseOwnerInstanceID
	}
	if leaseAcquiredAt != nil {
		es.LeaseAcquiredAt = leaseAcquiredAt
	}
	if leaseExpiresAt != nil {
		es.LeaseExpiresAt = leaseExpiresAt
	}
	if isActive != nil {
		es.IsActive = *isActive
	}
}

// AcquireLease sets the lease fields for the subscription
func (es *EventSubscription) AcquireLease(instanceID string, duration time.Duration) {
	now := time.Now()
	es.LeaseOwnerInstanceID = &instanceID
	es.LeaseAcquiredAt = &now
	expiresAt := now.Add(duration)
	es.LeaseExpiresAt = &expiresAt
}

// ReleaseLease clears the lease fields for the subscription
func (es *EventSubscription) ReleaseLease() {
	es.LeaseOwnerInstanceID = nil
	es.LeaseAcquiredAt = nil
	es.LeaseExpiresAt = nil
}

// IsLeaseExpired checks if the current lease has expired
func (es *EventSubscription) IsLeaseExpired() bool {
	if es.LeaseExpiresAt == nil {
		return false
	}
	return time.Now().After(*es.LeaseExpiresAt)
}

// HasActiveLease checks if the subscription has an active (non-expired) lease
func (es *EventSubscription) HasActiveLease() bool {
	return es.LeaseOwnerInstanceID != nil && !es.IsLeaseExpired()
}

// EventSubscriptionCommander defines the interface for event subscription command operations
type EventSubscriptionCommander interface {
	// UpdateProgress updates the last event sequence processed
	UpdateProgress(ctx context.Context, subscriberID string, lastEventSequenceProcessed int64) (*EventSubscription, error)

	// AcquireLease attempts to acquire a lease for processing events
	AcquireLease(ctx context.Context, subscriberID string, instanceID string, duration time.Duration) (*EventSubscription, error)

	// RenewLease renews an existing lease
	RenewLease(ctx context.Context, subscriberID string, instanceID string, duration time.Duration) (*EventSubscription, error)

	// ReleaseLease releases the lease for the subscription
	ReleaseLease(ctx context.Context, subscriberID string, instanceID string) (*EventSubscription, error)

	// AcknowledgeEvents acknowledges processed events by updating progress, but only if the instance holds a valid lease
	AcknowledgeEvents(ctx context.Context, subscriberID string, instanceID string, lastEventSequenceProcessed int64) (*EventSubscription, error)

	// SetActive sets the active status of the subscription
	SetActive(ctx context.Context, subscriberID string, isActive bool) (*EventSubscription, error)

	// Delete removes an event subscription
	Delete(ctx context.Context, subscriberID string) error
}

// eventSubscriptionCommander is the concrete implementation of EventSubscriptionCommander
type eventSubscriptionCommander struct {
	store Store
}

// NewEventSubscriptionCommander creates a new default EventSubscriptionCommander
func NewEventSubscriptionCommander(store Store) EventSubscriptionCommander {
	return &eventSubscriptionCommander{
		store: store,
	}
}

func (c *eventSubscriptionCommander) UpdateProgress(
	ctx context.Context,
	subscriberID string,
	lastEventSequenceProcessed int64,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		return nil, err
	}

	subscription.Update(&lastEventSequenceProcessed, nil, nil, nil, nil)
	if err := subscription.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := c.store.EventSubscriptionRepo().Save(ctx, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *eventSubscriptionCommander) AcquireLease(
	ctx context.Context,
	subscriberID string,
	instanceID string,
	duration time.Duration,
) (*EventSubscription, error) {
	// First try to find existing subscription
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		var notFoundErr NotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, err
		}
		// Create new subscription if not found
		subscription = NewEventSubscription(subscriberID)
		if err := subscription.Validate(); err != nil {
			return nil, InvalidInputError{Err: err}
		}
		if err := c.store.EventSubscriptionRepo().Create(ctx, subscription); err != nil {
			return nil, err
		}
	}

	// Check if lease can be acquired
	if subscription.HasActiveLease() && subscription.LeaseOwnerInstanceID != nil && *subscription.LeaseOwnerInstanceID != instanceID {
		return nil, NewInvalidInputErrorf("lease is already held by instance %s", *subscription.LeaseOwnerInstanceID)
	}

	subscription.AcquireLease(instanceID, duration)
	if err := subscription.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := c.store.EventSubscriptionRepo().Save(ctx, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *eventSubscriptionCommander) RenewLease(
	ctx context.Context,
	subscriberID string,
	instanceID string,
	duration time.Duration,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		return nil, err
	}

	// Check if the instance owns the lease
	if subscription.LeaseOwnerInstanceID == nil || *subscription.LeaseOwnerInstanceID != instanceID {
		return nil, NewInvalidInputErrorf("lease is not owned by instance %s", instanceID)
	}

	subscription.AcquireLease(instanceID, duration)
	if err := subscription.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := c.store.EventSubscriptionRepo().Save(ctx, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *eventSubscriptionCommander) ReleaseLease(
	ctx context.Context,
	subscriberID string,
	instanceID string,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		return nil, err
	}

	// Check if the instance owns the lease
	if subscription.LeaseOwnerInstanceID == nil || *subscription.LeaseOwnerInstanceID != instanceID {
		return nil, NewInvalidInputErrorf("lease is not owned by instance %s", instanceID)
	}

	subscription.ReleaseLease()
	if err := subscription.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := c.store.EventSubscriptionRepo().Save(ctx, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *eventSubscriptionCommander) AcknowledgeEvents(
	ctx context.Context,
	subscriberID string,
	instanceID string,
	lastEventSequenceProcessed int64,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		return nil, err
	}

	// Check if the instance owns a valid lease
	if !subscription.HasActiveLease() {
		return nil, NewInvalidInputErrorf("no active lease found for subscriber %s", subscriberID)
	}
	if subscription.LeaseOwnerInstanceID == nil || *subscription.LeaseOwnerInstanceID != instanceID {
		return nil, NewInvalidInputErrorf("lease is not owned by instance %s", instanceID)
	}

	// Only update if the new sequence is greater than current (prevent regression)
	if lastEventSequenceProcessed <= subscription.LastEventSequenceProcessed {
		return nil, NewInvalidInputErrorf("cannot acknowledge sequence %d: must be greater than current sequence %d",
			lastEventSequenceProcessed, subscription.LastEventSequenceProcessed)
	}

	subscription.Update(&lastEventSequenceProcessed, nil, nil, nil, nil)
	if err := subscription.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := c.store.EventSubscriptionRepo().Save(ctx, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *eventSubscriptionCommander) SetActive(
	ctx context.Context,
	subscriberID string,
	isActive bool,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		return nil, err
	}

	subscription.Update(nil, nil, nil, nil, &isActive)
	if err := subscription.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := c.store.EventSubscriptionRepo().Save(ctx, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *eventSubscriptionCommander) Delete(ctx context.Context, subscriberID string) error {
	_, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, subscriberID)
	if err != nil {
		return err
	}

	return c.store.EventSubscriptionRepo().DeleteBySubscriberID(ctx, subscriberID)
}

// EventSubscriptionRepository defines the interface for event subscription data operations
type EventSubscriptionRepository interface {
	EventSubscriptionQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *EventSubscription) error

	// Save updates an existing entity
	Save(ctx context.Context, entity *EventSubscription) error

	// DeleteBySubscriberID removes an entity by subscriber ID
	DeleteBySubscriberID(ctx context.Context, subscriberID string) error
}

// EventSubscriptionQuerier defines the interface for event subscription query operations
type EventSubscriptionQuerier interface {
	// Get retrieves an entity by ID
	Get(ctx context.Context, id properties.UUID) (*EventSubscription, error)

	// FindBySubscriberID retrieves an entity by subscriber ID
	FindBySubscriberID(ctx context.Context, subscriberID string) (*EventSubscription, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id properties.UUID) (bool, error)

	// ExistsBySubscriberID checks if an entity with the given subscriber ID exists
	ExistsBySubscriberID(ctx context.Context, subscriberID string) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *auth.IdentityScope, req *PageRequest) (*PageResponse[EventSubscription], error)

	// ListExpiredLeases retrieves subscriptions with expired leases
	ListExpiredLeases(ctx context.Context) ([]*EventSubscription, error)

	// AuthScope retrieves the auth scope for the entity
	AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}
