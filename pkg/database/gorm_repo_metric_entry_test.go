package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fulcrumproject/core/pkg/domain"
)

func TestMetricEntryRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewMetricEntryRepository(testDB.DB)

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

	t.Run("Create", func(t *testing.T) {
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

			// Create test entries with known values
			testValues := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
			expectedSum := 150.0
			expectedMax := 50.0
			expectedDiffMaxMin := 40.0 // 50 - 10

			// Create entries
			for i, value := range testValues {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: fmt.Sprintf("aggregate-test-%d", i),
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

			// Test MAX aggregate
			maxResult, err := repo.Aggregate(context.Background(), domain.AggregateMax, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, expectedMax, maxResult, "MAX aggregate should return the maximum value")

			// Test SUM aggregate
			sumResult, err := repo.Aggregate(context.Background(), domain.AggregateSum, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, expectedSum, sumResult, "SUM aggregate should return the sum of all values")

			// Test DIFF_MAX_MIN aggregate
			diffResult, err := repo.Aggregate(context.Background(), domain.AggregateDiffMaxMin, service.ID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, expectedDiffMaxMin, diffResult, "DIFF_MAX_MIN aggregate should return the difference between max and min")
		})

		t.Run("success - returns zero for no matching entries", func(t *testing.T) {
			// Use a non-existent service ID
			nonExistentServiceID := properties.NewUUID()
			start := time.Now().Add(-1 * time.Hour)
			end := time.Now().Add(1 * time.Hour)

			// Test all aggregate types with non-existent service
			maxResult, err := repo.Aggregate(context.Background(), domain.AggregateMax, nonExistentServiceID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 0.0, maxResult, "Should return 0 when no entries match")

			sumResult, err := repo.Aggregate(context.Background(), domain.AggregateSum, nonExistentServiceID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 0.0, sumResult, "Should return 0 when no entries match")

			diffResult, err := repo.Aggregate(context.Background(), domain.AggregateDiffMaxMin, nonExistentServiceID, metricTypeService.ID, start, end)
			require.NoError(t, err)
			assert.Equal(t, 0.0, diffResult, "Should return 0 when no entries match")
		})

		t.Run("error - unsupported aggregate type", func(t *testing.T) {
			start := time.Now().Add(-1 * time.Hour)
			end := time.Now().Add(1 * time.Hour)

			// Test with invalid aggregate type
			result, err := repo.Aggregate(context.Background(), domain.AggregateType("invalid"), service.ID, metricTypeService.ID, start, end)
			require.Error(t, err)
			assert.Equal(t, 0.0, result)
			assert.Contains(t, err.Error(), "unsupported aggregate type")
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

			// Check that the returned scope is a auth.DefaultObjectScope
			defaultScope, ok := scope.(*auth.DefaultObjectScope)
			require.True(t, ok, "AuthScope should return a auth.DefaultObjectScope")
			assert.Equal(t, provider.ID, *defaultScope.ProviderID, "Should return the correct provider ID")
			assert.Equal(t, consumer.ID, *defaultScope.ConsumerID, "Should return the correct consumer ID")
			assert.Equal(t, agent.ID, *defaultScope.AgentID, "Should return the correct agent ID")
		})
	})
}
