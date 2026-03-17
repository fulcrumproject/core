package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fulcrumproject/core/pkg/domain"
)

func TestMetricEntryRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewMetricEntryRepository(testDB.MetricDB)

	// Create repository instances
	participantRepo := NewParticipantRepository(testDB.DB)
	serviceTypeRepo := NewServiceTypeRepository(testDB.DB)
	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	agentRepo := NewAgentRepository(testDB.DB)
	serviceGroupRepo := NewServiceGroupRepository(testDB.DB)
	serviceRepo := NewServiceRepository(testDB.DB)
	metricTypeRepo := NewMetricTypeRepository(testDB.DB)

	ctx := context.Background()

	// Create participants (for consumer and provider roles)
	consumer := createTestParticipant(t, domain.ParticipantEnabled)
	err := participantRepo.Create(ctx, consumer)
	require.NoError(t, err)

	// Create provider participant
	provider := createTestParticipant(t, domain.ParticipantEnabled)
	provider.Name = "Test Provider"
	err = participantRepo.Create(ctx, provider)
	require.NoError(t, err)

	serviceType := createTestServiceType(t)
	err = serviceTypeRepo.Create(ctx, serviceType)
	require.NoError(t, err)

	agentType := createTestAgentType(t)
	agentType.ServiceTypes = []domain.ServiceType{*serviceType}
	err = agentTypeRepo.Create(ctx, agentType)
	require.NoError(t, err)

	agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
	err = agentRepo.Create(ctx, agent)
	require.NoError(t, err)

	serviceGroup := createTestServiceGroup(t, consumer.ID)
	err = serviceGroupRepo.Create(ctx, serviceGroup)
	require.NoError(t, err)

	service := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID, provider.ID, consumer.ID)
	err = serviceRepo.Create(ctx, service)
	require.NoError(t, err)

	// Create different metric types for different entity types
	metricTypeService := createTestMetricTypeForEntity(t, domain.MetricEntityTypeService)
	err = metricTypeRepo.Create(ctx, metricTypeService)
	require.NoError(t, err)

	metricTypeAgent := createTestMetricTypeForEntity(t, domain.MetricEntityTypeAgent)
	err = metricTypeRepo.Create(ctx, metricTypeAgent)
	require.NoError(t, err)
	require.NoError(t, err)

	t.Run("create", func(t *testing.T) {
		metricEntry := &domain.MetricEntry{
			AgentID:    agent.ID,
			ServiceID:  service.ID,
			ResourceID: "test-resource",
			ProviderID: provider.ID,
			ConsumerID: consumer.ID,
			Value:      42.5,
			TypeID:     metricTypeService.ID,
		}

		err := repo.Create(context.Background(), metricEntry)
		require.NoError(t, err)
		assert.NotEmpty(t, metricEntry.ID)
		assert.NotZero(t, metricEntry.CreatedAt)
		assert.NotZero(t, metricEntry.UpdatedAt)

		// Use the utility function to create a metric entry
		metricEntryFromUtil := createTestMetricEntry(t, agent.ID, service.ID, metricTypeAgent.ID, provider.ID, consumer.ID)

		err = repo.Create(context.Background(), metricEntryFromUtil)
		require.NoError(t, err)
		assert.NotEmpty(t, metricEntryFromUtil.ID)
		assert.NotZero(t, metricEntryFromUtil.CreatedAt)
		assert.NotZero(t, metricEntryFromUtil.UpdatedAt)
		assert.Equal(t, metricTypeAgent.ID, metricEntryFromUtil.TypeID)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple metric entries
			entries := []struct {
				resourceID string
				value      float64
			}{
				{"resource-1", 10.0},
				{"resource-2", 20.0},
				{"resource-3", 30.0},
			}

			for _, e := range entries {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: e.resourceID,
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      e.value,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)

			// Verify relationships are loaded
			assert.NotNil(t, result.Items[0].Agent)
			assert.NotNil(t, result.Items[0].Service)
			assert.NotNil(t, result.Items[0].Type)
		})

		t.Run("success - list with agent filter", func(t *testing.T) {
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"agentId": {agent.ID.String()}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			for _, item := range result.Items {
				assert.Equal(t, agent.ID, item.AgentID)
			}
		})

		t.Run("success - list with resource filter", func(t *testing.T) {
			entry1 := createTestMetricEntry(t, agent.ID, service.ID, metricTypeAgent.ID, provider.ID, consumer.ID)
			err := repo.Create(context.Background(), entry1)
			require.NoError(t, err)

			entry2 := createTestMetricEntry(t, agent.ID, service.ID, metricTypeAgent.ID, provider.ID, consumer.ID)
			err = repo.Create(context.Background(), entry2)
			require.NoError(t, err)

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"resourceId": {entry1.ResourceID}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Equal(t, 1, len(result.Items))
			assert.Equal(t, entry1.ResourceID, result.Items[0].ResourceID)
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			// Create entries with different types
			entryService := createTestMetricEntry(t, agent.ID, service.ID, metricTypeService.ID, provider.ID, consumer.ID)
			err := repo.Create(context.Background(), entryService)
			require.NoError(t, err)

			entryAgent := createTestMetricEntry(t, agent.ID, service.ID, metricTypeAgent.ID, provider.ID, consumer.ID)
			err = repo.Create(context.Background(), entryAgent)
			require.NoError(t, err)

			// Filter by service metric type
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"typeId": {metricTypeService.ID.String()}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, metricTypeService.ID, item.TypeID)
				if item.Type != nil {
					assert.Equal(t, domain.MetricEntityTypeService, item.Type.EntityType)
				}
			}

			// One of these entries should have the metric type for agents
			assert.GreaterOrEqual(t, result.TotalItems, int64(len(result.Items)))
		})

		t.Run("success - list with datetime sorting", func(t *testing.T) {
			// Create entries with specific values for sorting
			entries := []float64{15.0, 25.0, 35.0}
			for _, value := range entries {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: "sort-test",
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      value,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "createdAt",
				SortAsc:  false, // Descending order
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].CreatedAt, result.Items[i].CreatedAt)
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			// Create multiple metric entries
			for i := 0; i < 5; i++ {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: "pagination-test",
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      float64(i * 10),
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
			}

			// First page
			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Second page
			page.Page = 2
			result, err = repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})

	t.Run("CountByMetricType", func(t *testing.T) {
		t.Run("success - counts entries for a specific metric type", func(t *testing.T) {
			// Clear existing entries for better test isolation
			testDB.DB.Exec("DELETE FROM metric_entries")

			// Create entries for service metric type
			const serviceEntryCount = 3
			for i := 0; i < serviceEntryCount; i++ {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: fmt.Sprintf("count-service-%d", i),
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      float64(i * 10),
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			// Create entries for agent metric type
			const agentEntryCount = 5
			for i := 0; i < agentEntryCount; i++ {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: fmt.Sprintf("count-agent-%d", i),
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      float64(i * 10),
					TypeID:     metricTypeAgent.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			// Count entries for service metric type
			serviceCount, err := repo.CountByMetricType(context.Background(), metricTypeService.ID)
			require.NoError(t, err)
			assert.Equal(t, int64(serviceEntryCount), serviceCount, "Should return correct count for service metric type")

			// Count entries for agent metric type
			agentCount, err := repo.CountByMetricType(context.Background(), metricTypeAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, int64(agentEntryCount), agentCount, "Should return correct count for agent metric type")

			// Count entries for non-existent metric type
			nonExistentTypeID := properties.NewUUID()
			nonExistentCount, err := repo.CountByMetricType(context.Background(), nonExistentTypeID)
			require.NoError(t, err)
			assert.Equal(t, int64(0), nonExistentCount, "Should return zero for non-existent metric type")
		})
	})

	t.Run("Aggregate", func(t *testing.T) {
		t.Run("success - aggregates values correctly", func(t *testing.T) {
			// Clear existing entries for better test isolation
			testDB.DB.Exec("DELETE FROM metric_entries")

			// Create test entries with known values and same resource ID
			testValues := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
			resourceID := "aggregate-test"

			for _, value := range testValues {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: resourceID,
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      value,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			// Set time range to cover all entries
			start := time.Now().Add(-1 * time.Hour)
			end := time.Now().Add(1 * time.Hour)

			// All entries are created in the same second, so with hour bucket they land in one bucket

			baseQuery := domain.AggregateQuery{
				ServiceID:  service.ID,
				ResourceID: resourceID,
				TypeID:     metricTypeService.ID,
				Bucket:     domain.AggregateBucketHour,
				Start:      start,
				End:        end,
			}

			// Test MAX aggregate
			maxQuery := baseQuery
			maxQuery.Aggregate = domain.AggregateMax
			maxResult, err := repo.Aggregate(context.Background(), maxQuery)
			require.NoError(t, err)
			require.Len(t, maxResult.Data, 1, "All entries should fall in one bucket")
			assert.Equal(t, 50.0, maxResult.Data[0][1], "MAX aggregate should return the maximum value")
			assert.Equal(t, domain.AggregateMax, maxResult.Aggregate)
			assert.Equal(t, domain.AggregateBucketHour, maxResult.Bucket)

			// Test SUM aggregate
			sumQuery := baseQuery
			sumQuery.Aggregate = domain.AggregateSum
			sumResult, err := repo.Aggregate(context.Background(), sumQuery)
			require.NoError(t, err)
			require.Len(t, sumResult.Data, 1)
			assert.Equal(t, 150.0, sumResult.Data[0][1], "SUM aggregate should return the sum of all values")

			// Test AVG aggregate
			avgQuery := baseQuery
			avgQuery.Aggregate = domain.AggregateAvg
			avgResult, err := repo.Aggregate(context.Background(), avgQuery)
			require.NoError(t, err)
			require.Len(t, avgResult.Data, 1)
			assert.Equal(t, 30.0, avgResult.Data[0][1], "AVG aggregate should return the average value")

			// Test MIN aggregate
			minQuery := baseQuery
			minQuery.Aggregate = domain.AggregateMin
			minResult, err := repo.Aggregate(context.Background(), minQuery)
			require.NoError(t, err)
			require.Len(t, minResult.Data, 1)
			assert.Equal(t, 10.0, minResult.Data[0][1], "MIN aggregate should return the minimum value")

			// Test DIFF aggregate (max - min = 50 - 10 = 40)
			diffQuery := baseQuery
			diffQuery.Aggregate = domain.AggregateDiffMaxMin
			diffResult, err := repo.Aggregate(context.Background(), diffQuery)
			require.NoError(t, err)
			require.Len(t, diffResult.Data, 1)
			assert.Equal(t, 40.0, diffResult.Data[0][1], "DIFF aggregate should return max - min")
		})

		t.Run("success - returns empty data for no matching entries", func(t *testing.T) {
			// Use a non-existent service ID
			nonExistentServiceID := properties.NewUUID()
			start := time.Now().Add(-1 * time.Hour)
			end := time.Now().Add(1 * time.Hour)

			result, err := repo.Aggregate(context.Background(), domain.AggregateQuery{
				ServiceID:  nonExistentServiceID,
				ResourceID: "no-match",
				TypeID:     metricTypeService.ID,
				Aggregate:  domain.AggregateMax,
				Bucket:     domain.AggregateBucketHour,
				Start:      start,
				End:        end,
			})
			require.NoError(t, err)
			assert.Empty(t, result.Data, "Should return empty data when no entries match")
			assert.Equal(t, domain.AggregateMax, result.Aggregate)
			assert.Equal(t, domain.AggregateBucketHour, result.Bucket)
		})
	})

	t.Run("ListResourceIDs", func(t *testing.T) {
		t.Run("success - returns distinct resource IDs", func(t *testing.T) {
			testDB.DB.Exec("DELETE FROM metric_entries")

			// Create entries with different resource IDs
			resourceIDs := []string{"res-a", "res-b", "res-c", "res-a"} // res-a duplicated
			for _, rid := range resourceIDs {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: rid,
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      1.0,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{Page: 1, PageSize: 10}
			result, err := repo.ListResourceIDs(
				context.Background(),
				&auth.IdentityScope{},
				page,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(3), result.TotalItems, "Should return 3 distinct resource IDs")
			assert.Len(t, result.Items, 3)
		})
	})

	t.Run("AggregateTotal", func(t *testing.T) {
		t.Run("success - scalar aggregation for each type", func(t *testing.T) {
			testDB.DB.Exec("DELETE FROM metric_entries")

			testValues := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
			for _, value := range testValues {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: "agg-total-test",
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      value,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			start := time.Now().Add(-1 * time.Hour)
			end := time.Now().Add(1 * time.Hour)

			// Test MIN
			minVal, err := repo.AggregateTotal(ctx, domain.AggregateMin, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 10.0, minVal)

			// Test MAX
			maxVal, err := repo.AggregateTotal(ctx, domain.AggregateMax, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 50.0, maxVal)

			// Test SUM
			sumVal, err := repo.AggregateTotal(ctx, domain.AggregateSum, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 150.0, sumVal)

			// Test AVG
			avgVal, err := repo.AggregateTotal(ctx, domain.AggregateAvg, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 30.0, avgVal)

			// Test DIFF (max - min = 50 - 10 = 40)
			diffVal, err := repo.AggregateTotal(ctx, domain.AggregateDiffMaxMin, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 40.0, diffVal)
		})

		t.Run("success - returns 0 for non-existent service", func(t *testing.T) {
			start := time.Now().Add(-1 * time.Hour)
			end := time.Now().Add(1 * time.Hour)

			result, err := repo.AggregateTotal(ctx, domain.AggregateSum, properties.NewUUID(), metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 0.0, result)
		})
	})

	t.Run("ListResourceIDs", func(t *testing.T) {
		t.Run("success - returns distinct resource IDs", func(t *testing.T) {
			testDB.DB.Exec("DELETE FROM metric_entries")

			// Create entries with different resource IDs
			resourceIDs := []string{"res-a", "res-b", "res-c", "res-a"} // res-a duplicated
			for _, rid := range resourceIDs {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: rid,
					ProviderID: provider.ID,
					ConsumerID: consumer.ID,
					Value:      1.0,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageReq{Page: 1, PageSize: 10}
			result, err := repo.ListResourceIDs(
				context.Background(),
				&auth.IdentityScope{},
				page,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(3), result.TotalItems, "Should return 3 distinct resource IDs")
			assert.Len(t, result.Items, 3)
		})

		t.Run("success - pagination", func(t *testing.T) {
			page1 := &domain.PageReq{Page: 1, PageSize: 2}
			result1, err := repo.ListResourceIDs(
				context.Background(),
				&auth.IdentityScope{},
				page1,
			)
			require.NoError(t, err)
			assert.Len(t, result1.Items, 2)
			assert.True(t, result1.HasNext)

			page2 := &domain.PageReq{Page: 2, PageSize: 2}
			result2, err := repo.ListResourceIDs(
				context.Background(),
				&auth.IdentityScope{},
				page2,
			)
			require.NoError(t, err)
			assert.Len(t, result2.Items, 1)
			assert.False(t, result2.HasNext)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns correct auth scope", func(t *testing.T) {
			// Create a metric entry with all scope fields set
			metricEntry := &domain.MetricEntry{
				AgentID:    agent.ID,
				ServiceID:  service.ID,
				ResourceID: "auth-scope-test",
				ProviderID: provider.ID,
				ConsumerID: consumer.ID,
				Value:      42.0,
				TypeID:     metricTypeService.ID,
			}
			err := repo.Create(context.Background(), metricEntry)
			require.NoError(t, err)

			// Get the auth scope
			scope, err := repo.AuthScope(context.Background(), metricEntry.ID)
			require.NoError(t, err)
			assert.NotNil(t, scope, "AuthScope should not return nil")

			// Check that the returned scope is a authz.DefaultObjectScope
			defaultScope, ok := scope.(*authz.DefaultObjectScope)
			require.True(t, ok, "AuthScope should return a authz.DefaultObjectScope")
			assert.Equal(t, provider.ID, *defaultScope.ProviderID, "Should return the correct provider ID")
			assert.Equal(t, consumer.ID, *defaultScope.ConsumerID, "Should return the correct consumer ID")
			assert.Equal(t, agent.ID, *defaultScope.AgentID, "Should return the correct agent ID")
		})
	})
}
