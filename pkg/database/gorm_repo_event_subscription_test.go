package database

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestEventSubscription(t *testing.T, subscriberID string) *domain.EventSubscription {
	t.Helper()
	return &domain.EventSubscription{
		SubscriberID:               subscriberID,
		LastEventSequenceProcessed: 0,
		IsActive:                   true,
	}
}

func createTestEventSubscriptionWithLease(t *testing.T, subscriberID, instanceID string, duration time.Duration) *domain.EventSubscription {
	t.Helper()
	subscription := createTestEventSubscription(t, subscriberID)
	params := domain.LeaseParams{
		SubscriberID: subscriberID,
		InstanceID:   instanceID,
		Duration:     duration,
	}
	subscription.AcquireLease(params)
	return subscription
}

func TestEventSubscriptionRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewEventSubscriptionRepository(tdb.DB)

	t.Run("create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscription := createTestEventSubscription(t, "test-subscriber-1")

			// Execute
			err := repo.Create(ctx, subscription)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, subscription.ID)
			assert.NotZero(t, subscription.CreatedAt)
			assert.NotZero(t, subscription.UpdatedAt)
		})

		t.Run("duplicate subscriber_id should fail", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create first subscription
			subscription1 := createTestEventSubscription(t, "duplicate-subscriber")
			require.NoError(t, repo.Create(ctx, subscription1))

			// Setup - create second subscription with same subscriber_id
			subscription2 := createTestEventSubscription(t, "duplicate-subscriber")

			// Execute
			err := repo.Create(ctx, subscription2)

			// Assert
			assert.Error(t, err)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscription := createTestEventSubscription(t, "test-subscriber-2")
			require.NoError(t, repo.Create(ctx, subscription))

			// Execute
			found, err := repo.Get(ctx, subscription.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, subscription.ID, found.ID)
			assert.Equal(t, subscription.SubscriberID, found.SubscriberID)
			assert.Equal(t, subscription.LastEventSequenceProcessed, found.LastEventSequenceProcessed)
			assert.Equal(t, subscription.IsActive, found.IsActive)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			_, err := repo.Get(ctx, properties.NewUUID())

			// Assert
			var notFoundErr domain.NotFoundError
			assert.ErrorAs(t, err, &notFoundErr)
		})
	})

	t.Run("FindBySubscriberID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscriberID := "test-subscriber-3"
			subscription := createTestEventSubscription(t, subscriberID)
			require.NoError(t, repo.Create(ctx, subscription))

			// Execute
			found, err := repo.FindBySubscriberID(ctx, subscriberID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, subscription.ID, found.ID)
			assert.Equal(t, subscriberID, found.SubscriberID)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			_, err := repo.FindBySubscriberID(ctx, "non-existent-subscriber")

			// Assert
			var notFoundErr domain.NotFoundError
			assert.ErrorAs(t, err, &notFoundErr)
		})
	})

	t.Run("ExistsBySubscriberID", func(t *testing.T) {
		t.Run("exists", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscriberID := "test-subscriber-4"
			subscription := createTestEventSubscription(t, subscriberID)
			require.NoError(t, repo.Create(ctx, subscription))

			// Execute
			exists, err := repo.ExistsBySubscriberID(ctx, subscriberID)

			// Assert
			require.NoError(t, err)
			assert.True(t, exists)
		})

		t.Run("does not exist", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			exists, err := repo.ExistsBySubscriberID(ctx, "non-existent-subscriber")

			// Assert
			require.NoError(t, err)
			assert.False(t, exists)
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscription := createTestEventSubscription(t, "test-subscriber-5")
			require.NoError(t, repo.Create(ctx, subscription))

			// Modify the subscription
			subscription.LastEventSequenceProcessed = 100
			subscription.IsActive = false

			// Execute
			err := repo.Save(ctx, subscription)

			// Assert
			require.NoError(t, err)

			// Verify changes were saved
			found, err := repo.Get(ctx, subscription.ID)
			require.NoError(t, err)
			assert.Equal(t, int64(100), found.LastEventSequenceProcessed)
			assert.False(t, found.IsActive)
		})
	})

	t.Run("DeleteBySubscriberID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscriberID := "test-subscriber-6"
			subscription := createTestEventSubscription(t, subscriberID)
			require.NoError(t, repo.Create(ctx, subscription))

			// Execute
			err := repo.DeleteBySubscriberID(ctx, subscriberID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			_, err = repo.FindBySubscriberID(ctx, subscriberID)
			var notFoundErr domain.NotFoundError
			assert.ErrorAs(t, err, &notFoundErr)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			err := repo.DeleteBySubscriberID(ctx, "non-existent-subscriber")

			// Assert
			var notFoundErr domain.NotFoundError
			assert.ErrorAs(t, err, &notFoundErr)
		})
	})

	t.Run("ListExpiredLeases", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create subscription with expired lease
			expiredSubscription := createTestEventSubscriptionWithLease(t, "expired-subscriber", "instance-1", -time.Hour)
			require.NoError(t, repo.Create(ctx, expiredSubscription))

			// Setup - create subscription with active lease
			activeSubscription := createTestEventSubscriptionWithLease(t, "active-subscriber", "instance-2", time.Hour)
			require.NoError(t, repo.Create(ctx, activeSubscription))

			// Setup - create subscription without lease
			noLeaseSubscription := createTestEventSubscription(t, "no-lease-subscriber")
			require.NoError(t, repo.Create(ctx, noLeaseSubscription))

			// Execute
			expiredLeases, err := repo.ListExpiredLeases(ctx)

			// Assert
			require.NoError(t, err)
			assert.Len(t, expiredLeases, 1)
			assert.Equal(t, expiredSubscription.ID, expiredLeases[0].ID)
		})

		t.Run("no expired leases", func(t *testing.T) {
			ctx := context.Background()

			// Clean up any existing expired leases first
			expiredLeases, err := repo.ListExpiredLeases(ctx)
			require.NoError(t, err)
			for _, lease := range expiredLeases {
				_ = repo.DeleteBySubscriberID(ctx, lease.SubscriberID)
			}

			// Setup - create subscription with active lease
			activeSubscription := createTestEventSubscriptionWithLease(t, "active-subscriber-2", "instance-3", time.Hour)
			require.NoError(t, repo.Create(ctx, activeSubscription))

			// Execute
			expiredLeases, err = repo.ListExpiredLeases(ctx)

			// Assert
			require.NoError(t, err)
			assert.Empty(t, expiredLeases)
		})
	})

	t.Run("Lease Operations", func(t *testing.T) {
		t.Run("acquire and release lease", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscriberID := "lease-test-subscriber"
			subscription := createTestEventSubscription(t, subscriberID)
			require.NoError(t, repo.Create(ctx, subscription))

			// Acquire lease
			instanceID := "test-instance"
			duration := time.Hour
			params := domain.LeaseParams{
				SubscriberID: subscriberID,
				InstanceID:   instanceID,
				Duration:     duration,
			}
			subscription.AcquireLease(params)
			require.NoError(t, repo.Save(ctx, subscription))

			// Verify lease was acquired
			found, err := repo.FindBySubscriberID(ctx, subscriberID)
			require.NoError(t, err)
			assert.Equal(t, instanceID, *found.LeaseOwnerInstanceID)
			assert.NotNil(t, found.LeaseAcquiredAt)
			assert.NotNil(t, found.LeaseExpiresAt)
			assert.True(t, found.HasActiveLease())

			// Release lease
			found.ReleaseLease()
			require.NoError(t, repo.Save(ctx, found))

			// Verify lease was released
			found, err = repo.FindBySubscriberID(ctx, subscriberID)
			require.NoError(t, err)
			assert.Nil(t, found.LeaseOwnerInstanceID)
			assert.Nil(t, found.LeaseAcquiredAt)
			assert.Nil(t, found.LeaseExpiresAt)
			assert.False(t, found.HasActiveLease())
		})
	})

	t.Run("Exists", func(t *testing.T) {
		t.Run("exists", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscription := createTestEventSubscription(t, "exists-test-subscriber")
			require.NoError(t, repo.Create(ctx, subscription))

			// Execute
			exists, err := repo.Exists(ctx, subscription.ID)

			// Assert
			require.NoError(t, err)
			assert.True(t, exists)
		})

		t.Run("does not exist", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			exists, err := repo.Exists(ctx, properties.NewUUID())

			// Assert
			require.NoError(t, err)
			assert.False(t, exists)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			subscription := createTestEventSubscription(t, "authscope-test-subscriber")
			require.NoError(t, repo.Create(ctx, subscription))

			// Execute
			scope, err := repo.AuthScope(ctx, subscription.ID)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, scope)
			// EventSubscriptions are system-level resources, so scope should be empty
		})
	})
}
