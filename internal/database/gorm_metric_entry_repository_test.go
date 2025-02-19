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

		// Test listing with pagination
		result, err := repo.List(context.Background(), nil, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 3)

		// Test listing with agentId filter
		result, err = repo.List(context.Background(), &domain.SimpleFilter{Field: "agentId", Value: agent.ID.String()}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 3)
		assert.Equal(t, agent.ID, result.Items[0].AgentID)

		// Verify relationships are loaded
		assert.NotNil(t, result.Items[0].Agent)
		assert.NotNil(t, result.Items[0].Service)
		assert.NotNil(t, result.Items[0].Type)
	})
}
