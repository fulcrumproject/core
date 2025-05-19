package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestMetricEntryRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewMetricEntryRepository(testDB.DB)

	// Create repository instances
	providerRepo := NewProviderRepository(testDB.DB)
	serviceTypeRepo := NewServiceTypeRepository(testDB.DB)
	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	agentRepo := NewAgentRepository(testDB.DB)
	serviceGroupRepo := NewServiceGroupRepository(testDB.DB)
	serviceRepo := NewServiceRepository(testDB.DB)
	metricTypeRepo := NewMetricTypeRepository(testDB.DB)
	brokerRepo := NewBrokerRepository(testDB.DB)

	ctx := context.Background()

	// Create broker first (needed for service group)
	broker := createTestBroker(t)
	err := brokerRepo.Create(ctx, broker)
	require.NoError(t, err)

	// Create test dependencies
	provider := createTestProvider(t, domain.ProviderEnabled)
	err = providerRepo.Create(ctx, provider)
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

	serviceGroup := createTestServiceGroup(t, broker.ID)
	err = serviceGroupRepo.Create(ctx, serviceGroup)
	require.NoError(t, err)

	service := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID, provider.ID, broker.ID)
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
			ConsumerID: broker.ID,
			Value:      42.5,
			TypeID:     metricTypeService.ID,
		}

		err := repo.Create(context.Background(), metricEntry)
		require.NoError(t, err)
		assert.NotEmpty(t, metricEntry.ID)
		assert.NotZero(t, metricEntry.CreatedAt)
		assert.NotZero(t, metricEntry.UpdatedAt)

		// Use the utility function to create a metric entry
		metricEntryFromUtil := createTestMetricEntry(t, agent.ID, service.ID, metricTypeAgent.ID, provider.ID, broker.ID)

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
					ConsumerID: broker.ID,
					Value:      e.value,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), &domain.EmptyAuthScope, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)

			// Verify relationships are loaded
			assert.NotNil(t, result.Items[0].Agent)
			assert.NotNil(t, result.Items[0].Service)
			assert.NotNil(t, result.Items[0].Type)
		})

		t.Run("success - list with agent filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"agentId": {agent.ID.String()}},
			}

			result, err := repo.List(context.Background(), &domain.EmptyAuthScope, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			for _, item := range result.Items {
				assert.Equal(t, agent.ID, item.AgentID)
			}
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			// Create entries with different types
			entryService := createTestMetricEntry(t, agent.ID, service.ID, metricTypeService.ID, provider.ID, broker.ID)
			err := repo.Create(context.Background(), entryService)
			require.NoError(t, err)

			entryAgent := createTestMetricEntry(t, agent.ID, service.ID, metricTypeAgent.ID, provider.ID, broker.ID)
			err = repo.Create(context.Background(), entryAgent)
			require.NoError(t, err)

			// Filter by service metric type
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"typeId": {metricTypeService.ID.String()}},
			}

			result, err := repo.List(context.Background(), &domain.EmptyAuthScope, page)
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
					ConsumerID: broker.ID,
					Value:      value,
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "createdAt",
				SortAsc:  false, // Descending order
			}

			result, err := repo.List(context.Background(), &domain.EmptyAuthScope, page)
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
					ConsumerID: broker.ID,
					Value:      float64(i * 10),
					TypeID:     metricTypeService.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// First page
			result, err := repo.List(context.Background(), &domain.EmptyAuthScope, page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Second page
			page.Page = 2
			result, err = repo.List(context.Background(), &domain.EmptyAuthScope, page)
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
					ConsumerID: broker.ID,
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
					ConsumerID: broker.ID,
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
			nonExistentTypeID := domain.NewUUID()
			nonExistentCount, err := repo.CountByMetricType(context.Background(), nonExistentTypeID)
			require.NoError(t, err)
			assert.Equal(t, int64(0), nonExistentCount, "Should return zero for non-existent metric type")
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
				ConsumerID: broker.ID,
				Value:      42.0,
				TypeID:     metricTypeService.ID,
			}
			err := repo.Create(context.Background(), metricEntry)
			require.NoError(t, err)

			// Get the auth scope
			scope, err := repo.AuthScope(context.Background(), metricEntry.ID)
			require.NoError(t, err)
			assert.NotNil(t, scope, "AuthScope should not return nil")
			assert.Equal(t, provider.ID, *scope.ParticipantID, "Should return the correct provider ID")
			assert.Equal(t, agent.ID, *scope.AgentID, "Should return the correct agent ID")
			assert.Equal(t, broker.ID, *scope.BrokerID, "Should return the correct broker ID")
		})
	})
}
