package database

import (
	"context"
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

	ctx := context.Background()

	// Create test dependencies
	provider := createTestProvider(t, domain.ProviderEnabled)
	err := providerRepo.Create(ctx, provider)
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

	serviceGroup := createTestServiceGroup(t)
	err = serviceGroupRepo.Create(ctx, serviceGroup)
	require.NoError(t, err)

	service := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID)
	err = serviceRepo.Create(ctx, service)
	require.NoError(t, err)

	metricType := createTestMetricType(t)
	err = metricTypeRepo.Create(ctx, metricType)
	require.NoError(t, err)

	t.Run("Create", func(t *testing.T) {
		metricEntry := &domain.MetricEntry{
			AgentID:    agent.ID,
			ServiceID:  service.ID,
			ResourceID: "test-resource",
			Value:      42.5,
			TypeID:     metricType.ID,
		}

		err := repo.Create(context.Background(), metricEntry)
		require.NoError(t, err)
		assert.NotEmpty(t, metricEntry.ID)
		assert.NotZero(t, metricEntry.CreatedAt)
		assert.NotZero(t, metricEntry.UpdatedAt)
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
					Value:      e.value,
					TypeID:     metricType.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), page)
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

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			for _, item := range result.Items {
				assert.Equal(t, agent.ID, item.AgentID)
			}
		})

		t.Run("success - list with datetime sorting", func(t *testing.T) {
			// Create entries with specific values for sorting
			entries := []float64{15.0, 25.0, 35.0}
			for _, value := range entries {
				entry := &domain.MetricEntry{
					AgentID:    agent.ID,
					ServiceID:  service.ID,
					ResourceID: "sort-test",
					Value:      value,
					TypeID:     metricType.ID,
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

			result, err := repo.List(context.Background(), page)
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
					Value:      float64(i * 10),
					TypeID:     metricType.ID,
				}
				err := repo.Create(context.Background(), entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// First page
			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Second page
			page.Page = 2
			result, err = repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})
}
