package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEventSubscription(t *testing.T) {
	subscriberID := "test-subscriber"
	subscription := NewEventSubscription(subscriberID)

	assert.Equal(t, subscriberID, subscription.SubscriberID)
	assert.Equal(t, int64(0), subscription.LastEventSequenceProcessed)
	assert.True(t, subscription.IsActive)
	assert.Nil(t, subscription.LeaseOwnerInstanceID)
	assert.Nil(t, subscription.LeaseAcquiredAt)
	assert.Nil(t, subscription.LeaseExpiresAt)
}

func TestEventSubscription_TableName(t *testing.T) {
	subscription := EventSubscription{}
	assert.Equal(t, "event_subscriptions", subscription.TableName())
}

func TestEventSubscription_Validate(t *testing.T) {
	tests := []struct {
		name         string
		subscription *EventSubscription
		wantErr      bool
		errContains  string
	}{
		{
			name: "valid subscription",
			subscription: &EventSubscription{
				SubscriberID:               "test-subscriber",
				LastEventSequenceProcessed: 100,
				IsActive:                   true,
			},
			wantErr: false,
		},
		{
			name: "empty subscriber ID",
			subscription: &EventSubscription{
				SubscriberID: "",
				IsActive:     true,
			},
			wantErr:     true,
			errContains: "subscriber_id cannot be empty",
		},
		{
			name: "negative sequence number",
			subscription: &EventSubscription{
				SubscriberID:               "test-subscriber",
				LastEventSequenceProcessed: -1,
				IsActive:                   true,
			},
			wantErr:     true,
			errContains: "last_event_sequence_processed cannot be negative",
		},
		{
			name: "valid lease",
			subscription: &EventSubscription{
				SubscriberID:               "test-subscriber",
				LastEventSequenceProcessed: 0,
				LeaseOwnerInstanceID:       stringPtr("instance-1"),
				LeaseAcquiredAt:            timePtr(time.Now()),
				LeaseExpiresAt:             timePtr(time.Now().Add(time.Hour)),
				IsActive:                   true,
			},
			wantErr: false,
		},
		{
			name: "lease with missing acquired time",
			subscription: &EventSubscription{
				SubscriberID:         "test-subscriber",
				LeaseOwnerInstanceID: stringPtr("instance-1"),
				LeaseExpiresAt:       timePtr(time.Now().Add(time.Hour)),
				IsActive:             true,
			},
			wantErr:     true,
			errContains: "lease_acquired_at must be set when lease_owner_instance_id is set",
		},
		{
			name: "lease with missing expires time",
			subscription: &EventSubscription{
				SubscriberID:         "test-subscriber",
				LeaseOwnerInstanceID: stringPtr("instance-1"),
				LeaseAcquiredAt:      timePtr(time.Now()),
				IsActive:             true,
			},
			wantErr:     true,
			errContains: "lease_expires_at must be set when lease_owner_instance_id is set",
		},
		{
			name: "lease expires before acquired",
			subscription: &EventSubscription{
				SubscriberID:         "test-subscriber",
				LeaseOwnerInstanceID: stringPtr("instance-1"),
				LeaseAcquiredAt:      timePtr(time.Now()),
				LeaseExpiresAt:       timePtr(time.Now().Add(-time.Hour)),
				IsActive:             true,
			},
			wantErr:     true,
			errContains: "lease_expires_at must be after lease_acquired_at",
		},
		{
			name: "acquired time without owner",
			subscription: &EventSubscription{
				SubscriberID:    "test-subscriber",
				LeaseAcquiredAt: timePtr(time.Now()),
				IsActive:        true,
			},
			wantErr:     true,
			errContains: "lease_acquired_at and lease_expires_at must be nil when lease_owner_instance_id is nil",
		},
		{
			name: "expires time without owner",
			subscription: &EventSubscription{
				SubscriberID:   "test-subscriber",
				LeaseExpiresAt: timePtr(time.Now()),
				IsActive:       true,
			},
			wantErr:     true,
			errContains: "lease_acquired_at and lease_expires_at must be nil when lease_owner_instance_id is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.subscription.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventSubscription_Update(t *testing.T) {
	subscription := NewEventSubscription("test-subscriber")

	// Test updating sequence number
	newSequence := int64(100)
	subscription.Update(&newSequence, nil, nil, nil, nil)
	assert.Equal(t, newSequence, subscription.LastEventSequenceProcessed)

	// Test updating lease owner
	instanceID := "instance-1"
	subscription.Update(nil, &instanceID, nil, nil, nil)
	assert.Equal(t, instanceID, *subscription.LeaseOwnerInstanceID)

	// Test updating acquired time
	acquiredTime := time.Now()
	subscription.Update(nil, nil, &acquiredTime, nil, nil)
	assert.Equal(t, acquiredTime, *subscription.LeaseAcquiredAt)

	// Test updating expires time
	expiresTime := time.Now().Add(time.Hour)
	subscription.Update(nil, nil, nil, &expiresTime, nil)
	assert.Equal(t, expiresTime, *subscription.LeaseExpiresAt)

	// Test updating active status
	isActive := false
	subscription.Update(nil, nil, nil, nil, &isActive)
	assert.False(t, subscription.IsActive)

	// Test nil values don't change existing values
	originalSequence := subscription.LastEventSequenceProcessed
	subscription.Update(nil, nil, nil, nil, nil)
	assert.Equal(t, originalSequence, subscription.LastEventSequenceProcessed)
}

func TestEventSubscription_AcquireLease(t *testing.T) {
	subscription := NewEventSubscription("test-subscriber")
	instanceID := "instance-1"
	duration := time.Hour

	beforeTime := time.Now()
	subscription.AcquireLease(instanceID, duration)
	afterTime := time.Now()

	assert.Equal(t, instanceID, *subscription.LeaseOwnerInstanceID)
	assert.True(t, subscription.LeaseAcquiredAt.After(beforeTime) || subscription.LeaseAcquiredAt.Equal(beforeTime))
	assert.True(t, subscription.LeaseAcquiredAt.Before(afterTime) || subscription.LeaseAcquiredAt.Equal(afterTime))
	assert.True(t, subscription.LeaseExpiresAt.After(subscription.LeaseAcquiredAt.Add(duration).Add(-time.Second)))
	assert.True(t, subscription.LeaseExpiresAt.Before(subscription.LeaseAcquiredAt.Add(duration).Add(time.Second)))
}

func TestEventSubscription_ReleaseLease(t *testing.T) {
	subscription := NewEventSubscription("test-subscriber")

	// First acquire a lease
	subscription.AcquireLease("instance-1", time.Hour)
	assert.NotNil(t, subscription.LeaseOwnerInstanceID)
	assert.NotNil(t, subscription.LeaseAcquiredAt)
	assert.NotNil(t, subscription.LeaseExpiresAt)

	// Then release it
	subscription.ReleaseLease()
	assert.Nil(t, subscription.LeaseOwnerInstanceID)
	assert.Nil(t, subscription.LeaseAcquiredAt)
	assert.Nil(t, subscription.LeaseExpiresAt)
}

func TestEventSubscription_IsLeaseExpired(t *testing.T) {
	subscription := NewEventSubscription("test-subscriber")

	// No lease should not be expired
	assert.False(t, subscription.IsLeaseExpired())

	// Expired lease
	pastTime := time.Now().Add(-time.Hour)
	subscription.LeaseExpiresAt = &pastTime
	assert.True(t, subscription.IsLeaseExpired())

	// Future lease
	futureTime := time.Now().Add(time.Hour)
	subscription.LeaseExpiresAt = &futureTime
	assert.False(t, subscription.IsLeaseExpired())
}

func TestEventSubscription_HasActiveLease(t *testing.T) {
	subscription := NewEventSubscription("test-subscriber")

	// No lease
	assert.False(t, subscription.HasActiveLease())

	// Lease without owner
	futureTime := time.Now().Add(time.Hour)
	subscription.LeaseExpiresAt = &futureTime
	assert.False(t, subscription.HasActiveLease())

	// Active lease
	instanceID := "instance-1"
	subscription.LeaseOwnerInstanceID = &instanceID
	assert.True(t, subscription.HasActiveLease())

	// Expired lease
	pastTime := time.Now().Add(-time.Hour)
	subscription.LeaseExpiresAt = &pastTime
	assert.False(t, subscription.HasActiveLease())
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}
