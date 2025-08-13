package database

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/google/uuid"
)

func TestEventRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewEventRepository(testDB.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			// Setup
			eventEntry := &domain.Event{
				InitiatorType: domain.InitiatorTypeUser,
				InitiatorID:   "test-agent-id",
				Type:          domain.EventTypeAgentUpdated,
				Payload: properties.JSON{
					"old_status": "new",
					"new_status": "active",
				},
			}

			// Execute
			err := repo.Create(ctx, eventEntry)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, eventEntry.ID)
			assert.NotZero(t, eventEntry.CreatedAt)
			assert.NotZero(t, eventEntry.UpdatedAt)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()
			// Create multiple event entries
			entries := []struct {
				initiatorType domain.InitiatorType
				initiatorID   string
				entryType     domain.EventType
			}{
				{domain.InitiatorTypeUser, "agent-1", domain.EventTypeAgentUpdated},
				{domain.InitiatorTypeUser, "service-1", domain.EventTypeServiceUpdated},
				{domain.InitiatorTypeUser, "agent-2", domain.EventTypeAgentUpdated},
			}

			for _, e := range entries {
				entry := &domain.Event{
					InitiatorType: e.initiatorType,
					InitiatorID:   e.initiatorID,
					Type:          e.entryType,
					Payload: properties.JSON{
						"test": "data",
					},
				}
				err := repo.Create(ctx, entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
		})

		t.Run("success - list with initiator type filter", func(t *testing.T) {
			ctx := context.Background()

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"initiatorType": {string(domain.InitiatorTypeUser)}},
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for _, item := range result.Items {
				assert.Equal(t, domain.InitiatorTypeUser, item.InitiatorType)
			}
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			ctx := context.Background()

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"type": {string(domain.EventTypeAgentUpdated)}},
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for _, item := range result.Items {
				assert.Equal(t, domain.EventTypeAgentUpdated, item.Type)
			}
		})

		t.Run("success - list with sorting by created_at", func(t *testing.T) {
			ctx := context.Background()

			// Create entries with different timestamps
			for i := 0; i < 3; i++ {
				entry := &domain.Event{
					InitiatorType: domain.InitiatorTypeUser,
					InitiatorID:   "agent-sort",
					Type:          domain.EventTypeAgentUpdated,
					Payload:       properties.JSON{"test": "data"},
				}
				err := repo.Create(ctx, entry)
				require.NoError(t, err)
				time.Sleep(time.Millisecond * 100) // Ensure different timestamps
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "createdAt",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.True(t, result.Items[i-1].CreatedAt.After(result.Items[i].CreatedAt))
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Create multiple event entries
			for i := 0; i < 5; i++ {
				entry := &domain.Event{
					InitiatorType: domain.InitiatorTypeUser,
					InitiatorID:   "agent-page",
					Type:          domain.EventTypeAgentCreated,
					Payload:       properties.JSON{"test": "data"},
				}
				err := repo.Create(ctx, entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns correct auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create an event entry with all scope IDs set
			providerID := properties.NewUUID()
			agentID := properties.NewUUID()
			consumerID := properties.NewUUID()

			eventEntry := &domain.Event{
				InitiatorType: domain.InitiatorTypeUser,
				InitiatorID:   "admin-test",
				Type:          domain.EventTypeAgentCreated,
				Payload:       properties.JSON{"test": "scoped event entry"},
				ProviderID:    &providerID,
				AgentID:       &agentID,
				ConsumerID:    &consumerID,
			}

			err := repo.Create(ctx, eventEntry)
			require.NoError(t, err)

			// Execute
			scope, err := repo.AuthScope(ctx, eventEntry.ID)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, scope, "AuthScope should not return nil")

			// Check that the returned scope is a auth.DefaultObjectScope
			defaultScope, ok := scope.(*auth.DefaultObjectScope)
			require.True(t, ok, "AuthScope should return a auth.DefaultObjectScope")
			require.NotNil(t, defaultScope.ProviderID, "ProviderID should be present")
			require.NotNil(t, defaultScope.ConsumerID, "ConsumerID should be present")
			require.NotNil(t, defaultScope.AgentID, "AgentID should be present")
			assert.Equal(t, providerID, *defaultScope.ProviderID, "Should return the correct provider ID")
			assert.Equal(t, consumerID, *defaultScope.ConsumerID, "Should return the correct consumer ID")
			assert.Equal(t, agentID, *defaultScope.AgentID, "Should return the correct agent ID")

			// Test with non-existent entry
			nonExistentID := properties.NewUUID()
			nonExistentScope, err := repo.AuthScope(ctx, nonExistentID)

			// The implementation appears to return an empty scope rather than an error for non-existent IDs
			require.Error(t, err)
			assert.Nil(t, nonExistentScope)
		})
	})
}

func TestGormEventRepository_Uptime(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewEventRepository(testDB.DB)

	// Test time range
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	start := baseTime
	end := baseTime.Add(1 * time.Hour)

	t.Run("No events - service never existed", func(t *testing.T) {
		serviceID := properties.UUID(uuid.New())
		uptimeSeconds, downtimeSeconds, err := repo.ServiceUptime(context.Background(), serviceID, start, end)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), uptimeSeconds)
		assert.Equal(t, uint64(3600), downtimeSeconds) // 1 hour = 3600 seconds
	})

	t.Run("Service running entire period", func(t *testing.T) {
		serviceID := properties.UUID(uuid.New())

		// Create a service transition event before the start time showing service was started
		createTestServiceEvent(t, repo, serviceID, domain.ServiceStarted, baseTime.Add(-30*time.Minute))

		uptimeSeconds, downtimeSeconds, err := repo.ServiceUptime(context.Background(), serviceID, start, end)
		require.NoError(t, err)
		assert.Equal(t, uint64(3600), uptimeSeconds) // 1 hour = 3600 seconds
		assert.Equal(t, uint64(0), downtimeSeconds)
	})

	t.Run("Service stopped entire period", func(t *testing.T) {
		serviceID := properties.UUID(uuid.New())

		// Create a service transition event before the start time showing service was stopped
		createTestServiceEvent(t, repo, serviceID, domain.ServiceStopped, baseTime.Add(-30*time.Minute))

		uptimeSeconds, downtimeSeconds, err := repo.ServiceUptime(context.Background(), serviceID, start, end)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), uptimeSeconds)
		assert.Equal(t, uint64(3600), downtimeSeconds) // 1 hour = 3600 seconds
	})

	t.Run("Service started during period", func(t *testing.T) {
		serviceID := properties.UUID(uuid.New())

		// Service starts 20 minutes into the period
		createTestServiceEvent(t, repo, serviceID, domain.ServiceStarted, start.Add(20*time.Minute))

		uptimeSeconds, downtimeSeconds, err := repo.ServiceUptime(context.Background(), serviceID, start, end)
		require.NoError(t, err)

		// Service was running for 40 minutes out of 60 minutes
		// From 12:20-13:00 (ServiceStarted) = 40 minutes = 2400 seconds
		// Total uptime: 40 minutes = 2400 seconds
		// Total downtime: 20 minutes = 1200 seconds
		assert.Equal(t, uint64(2400), uptimeSeconds)
		assert.Equal(t, uint64(1200), downtimeSeconds)
	})
}

// TODO this test is failing if it runs with the other test, but passes if it runs alone
func TestGormEventRepository_Uptime_2(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewEventRepository(testDB.DB)

	// Test time range
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	start := baseTime
	end := baseTime.Add(1 * time.Hour)

	t.Run("Service starts and stops during period", func(t *testing.T) {
		serviceID := properties.UUID(uuid.New())

		// Service starts 15 minutes into the period
		createTestServiceEvent(t, repo, serviceID, domain.ServiceStarted, start.Add(15*time.Minute))

		// Service stops 45 minutes into the period
		createTestServiceEvent(t, repo, serviceID, domain.ServiceStopped, start.Add(45*time.Minute))

		uptimeSeconds, downtimeSeconds, err := repo.ServiceUptime(context.Background(), serviceID, start, end)
		require.NoError(t, err)

		// Service was running for 30 minutes out of 60 minutes
		// 30 minutes = 1800 seconds uptime
		// 30 minutes = 1800 seconds downtime
		assert.Equal(t, uint64(1800), uptimeSeconds)
		assert.Equal(t, uint64(1800), downtimeSeconds)
	})
}

// Helper function to create test service events
func createTestServiceEvent(t *testing.T, repo *GormEventRepository, serviceID properties.UUID, newStatus domain.ServiceStatus, createdAt time.Time) {
	t.Helper()
	event := &domain.Event{
		BaseEntity: domain.BaseEntity{
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		// Don't set SequenceNumber explicitly - let GORM auto-increment it
		InitiatorType: domain.InitiatorTypeSystem,
		InitiatorID:   "test-system",
		Type:          domain.EventTypeServiceTransitioned,
		EntityID:      &serviceID,
		Payload: properties.JSON{
			"diff": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "/currentStatus",
					"value": string(newStatus),
				},
			},
		},
	}
	err := repo.Create(context.Background(), event)
	require.NoError(t, err)
}
