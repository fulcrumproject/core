package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
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
func (es *EventSubscription) AcquireLease(params LeaseParams) {
	now := time.Now()
	es.LeaseOwnerInstanceID = &params.InstanceID
	es.LeaseAcquiredAt = &now
	expiresAt := now.Add(params.Duration)
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
	UpdateProgress(ctx context.Context, params UpdateProgressParams) (*EventSubscription, error)

	// AcquireLease attempts to acquire a lease for processing events
	AcquireLease(ctx context.Context, params LeaseParams) (*EventSubscription, error)

	// RenewLease renews an existing lease
	RenewLease(ctx context.Context, params LeaseParams) (*EventSubscription, error)

	// ReleaseLease releases the lease for the subscription
	ReleaseLease(ctx context.Context, params ReleaseLeaseParams) (*EventSubscription, error)

	// AcknowledgeEvents acknowledges processed events by updating progress, but only if the instance holds a valid lease
	AcknowledgeEvents(ctx context.Context, params AcknowledgeEventsParams) (*EventSubscription, error)

	// SetActive sets the active status of the subscription
	SetActive(ctx context.Context, params SetActiveParams) (*EventSubscription, error)

	// Delete removes an event subscription
	Delete(ctx context.Context, subscriberID string) error
}

type UpdateProgressParams struct {
	SubscriberID               string
	LastEventSequenceProcessed int64
}

type LeaseParams struct {
	SubscriberID string
	InstanceID   string
	Duration     time.Duration
}

type ReleaseLeaseParams struct {
	SubscriberID string
	InstanceID   string
}

type AcknowledgeEventsParams struct {
	SubscriberID               string
	InstanceID                 string
	LastEventSequenceProcessed int64
}

type SetActiveParams struct {
	SubscriberID string
	IsActive     bool
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
	params UpdateProgressParams,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, params.SubscriberID)
	if err != nil {
		return nil, err
	}

	subscription.Update(&params.LastEventSequenceProcessed, nil, nil, nil, nil)
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
	params LeaseParams,
) (*EventSubscription, error) {
	// First try to find existing subscription
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, params.SubscriberID)
	if err != nil {
		var notFoundErr NotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, err
		}
		// Create new subscription if not found
		subscription = NewEventSubscription(params.SubscriberID)
		if err := subscription.Validate(); err != nil {
			return nil, InvalidInputError{Err: err}
		}
		if err := c.store.EventSubscriptionRepo().Create(ctx, subscription); err != nil {
			return nil, err
		}
	}

	// Check if lease can be acquired
	if subscription.HasActiveLease() && subscription.LeaseOwnerInstanceID != nil && *subscription.LeaseOwnerInstanceID != params.InstanceID {
		return nil, NewInvalidInputErrorf("lease is already held by instance %s", *subscription.LeaseOwnerInstanceID)
	}

	subscription.AcquireLease(params)
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
	params LeaseParams,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, params.SubscriberID)
	if err != nil {
		return nil, err
	}

	// Check if the instance owns the lease
	if subscription.LeaseOwnerInstanceID == nil || *subscription.LeaseOwnerInstanceID != params.InstanceID {
		return nil, NewInvalidInputErrorf("lease is not owned by instance %s", params.InstanceID)
	}

	subscription.AcquireLease(params)
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
	params ReleaseLeaseParams,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, params.SubscriberID)
	if err != nil {
		return nil, err
	}

	// Check if the instance owns the lease
	if subscription.LeaseOwnerInstanceID == nil || *subscription.LeaseOwnerInstanceID != params.InstanceID {
		return nil, NewInvalidInputErrorf("lease is not owned by instance %s", params.InstanceID)
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
	params AcknowledgeEventsParams,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, params.SubscriberID)
	if err != nil {
		return nil, err
	}

	// Check if the instance owns a valid lease
	if !subscription.HasActiveLease() {
		return nil, NewInvalidInputErrorf("no active lease found for subscriber %s", params.SubscriberID)
	}
	if subscription.LeaseOwnerInstanceID == nil || *subscription.LeaseOwnerInstanceID != params.InstanceID {
		return nil, NewInvalidInputErrorf("lease is not owned by instance %s", params.InstanceID)
	}

	// Only update if the new sequence is greater than current (prevent regression)
	if params.LastEventSequenceProcessed <= subscription.LastEventSequenceProcessed {
		return nil, NewInvalidInputErrorf("cannot acknowledge sequence %d: must be greater than current sequence %d",
			params.LastEventSequenceProcessed, subscription.LastEventSequenceProcessed)
	}

	subscription.Update(&params.LastEventSequenceProcessed, nil, nil, nil, nil)
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
	params SetActiveParams,
) (*EventSubscription, error) {
	subscription, err := c.store.EventSubscriptionRepo().FindBySubscriberID(ctx, params.SubscriberID)
	if err != nil {
		return nil, err
	}

	subscription.Update(nil, nil, nil, nil, &params.IsActive)
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
	BaseEntityRepository[EventSubscription]

	// DeleteBySubscriberID removes an entity by subscriber ID
	DeleteBySubscriberID(ctx context.Context, subscriberID string) error
}

// EventSubscriptionQuerier defines the interface for event subscription query operations
type EventSubscriptionQuerier interface {
	BaseEntityRepository[EventSubscription]

	// FindBySubscriberID retrieves an entity by subscriber ID
	FindBySubscriberID(ctx context.Context, subscriberID string) (*EventSubscription, error)

	// ExistsBySubscriberID checks if an entity with the given subscriber ID exists
	ExistsBySubscriberID(ctx context.Context, subscriberID string) (bool, error)

	// ListExpiredLeases retrieves subscriptions with expired leases
	ListExpiredLeases(ctx context.Context) ([]*EventSubscription, error)
}
