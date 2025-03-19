package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestServiceRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceRepository(testDB.DB)

	// Create broker first (needed for service group)
	brokerRepo := NewBrokerRepository(testDB.DB)
	broker := createTestBroker(t)
	require.NoError(t, brokerRepo.Create(context.Background(), broker))

	// Create provider first
	providerRepo := NewProviderRepository(testDB.DB)
	provider := &domain.Provider{
		Name:  "Test Provider",
		State: domain.ProviderEnabled,
	}
	require.NoError(t, providerRepo.Create(context.Background(), provider))

	// Create dependencies first
	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	agentType := &domain.AgentType{
		Name: "Test Agent Type",
	}
	require.NoError(t, agentTypeRepo.Create(context.Background(), agentType))

	agentRepo := NewAgentRepository(testDB.DB)
	agent := &domain.Agent{
		Name:        "Test Agent",
		State:       domain.AgentConnected,
		ProviderID:  provider.ID,
		AgentTypeID: agentType.ID,
	}
	require.NoError(t, agentRepo.Create(context.Background(), agent))

	serviceTypeRepo := NewServiceTypeRepository(testDB.DB)
	serviceType := &domain.ServiceType{
		Name: "Test Service Type",
	}
	require.NoError(t, serviceTypeRepo.Create(context.Background(), serviceType))

	serviceGroupRepo := NewServiceGroupRepository(testDB.DB)
	serviceGroup := createTestServiceGroup(t, broker.ID)
	require.NoError(t, serviceGroupRepo.Create(context.Background(), serviceGroup))

	t.Run("Create", func(t *testing.T) {
		service := &domain.Service{
			Name:              "Test Service",
			CurrentState:      domain.ServiceStarted,
			CurrentProperties: &(domain.JSON{"key": "value"}),
			Resources:         &(domain.JSON{"cpu": "1"}),
			AgentID:           agent.ID,
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup.ID,
		}

		err := repo.Create(context.Background(), service)
		require.NoError(t, err)
		assert.NotEmpty(t, service.ID)
		assert.NotZero(t, service.CreatedAt)
		assert.NotZero(t, service.UpdatedAt)
	})

	t.Run("FindByID", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:              "Test Service",
			CurrentState:      domain.ServiceStarted,
			CurrentProperties: &(domain.JSON{"key": "value"}),
			Attributes:        domain.Attributes{"key": []string{"value"}},
			Resources:         &(domain.JSON{"cpu": "1"}),
			AgentID:           agent.ID,
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Find the service
		found, err := repo.FindByID(context.Background(), service.ID)
		require.NoError(t, err)
		assert.Equal(t, service.ID, found.ID)
		assert.Equal(t, service.Name, found.Name)
		assert.Equal(t, service.CurrentState, found.CurrentState)
		assert.Equal(t, service.CurrentProperties, found.CurrentProperties)
		assert.Equal(t, &(domain.JSON{"cpu": "1"}), found.Resources)
		assert.Equal(t, service.AgentID, found.AgentID)
		assert.Equal(t, service.ServiceTypeID, found.ServiceTypeID)
		assert.Equal(t, service.GroupID, found.GroupID)

		// Check relationships are loaded
		assert.NotNil(t, found.Agent)
		assert.NotNil(t, found.ServiceType)
		assert.NotNil(t, found.Group)
	})

	t.Run("FindByID_NotFound", func(t *testing.T) {
		found, err := repo.FindByID(context.Background(), domain.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:              "Test Service",
			CurrentState:      domain.ServiceStarted,
			CurrentProperties: &(domain.JSON{"key": "value"}),
			Attributes:        domain.Attributes{"key": []string{"value"}},
			Resources:         &(domain.JSON{"cpu": "1"}),
			AgentID:           agent.ID,
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Update the service
		service.Name = "Updated Service"
		service.CurrentState = domain.ServiceStarted
		service.CurrentProperties = &(domain.JSON{"key": "value"})
		service.Resources = &(domain.JSON{"cpu": "2"})

		err = repo.Save(context.Background(), service)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.FindByID(context.Background(), service.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Service", found.Name)
		assert.Equal(t, domain.ServiceStarted, found.CurrentState)
		assert.Equal(t, &(domain.JSON{"key": "value"}), found.CurrentProperties)
		assert.Equal(t, &(domain.JSON{"cpu": "2"}), found.Resources)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:          "Test Service",
			CurrentState:  domain.ServiceStarted,
			AgentID:       agent.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Delete the service
		err = repo.Delete(context.Background(), service.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByID(context.Background(), service.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple services
			services := []*domain.Service{
				{Name: "Service A", CurrentState: domain.ServiceStarted, AgentID: agent.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
				{Name: "Service B", CurrentState: domain.ServiceStarted, AgentID: agent.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
				{Name: "Service C", CurrentState: domain.ServiceStarted, AgentID: agent.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
			}
			for _, service := range services {
				err := repo.Create(context.Background(), service)
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
			assert.NotNil(t, result.Items[0].ServiceType)
			assert.NotNil(t, result.Items[0].Group)
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"Service A"}},
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 1)
			assert.Equal(t, "Service A", result.Items[0].Name)
		})

		t.Run("success - list with state filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"currentState": {string(domain.ServiceStarted)}},
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, domain.ServiceStarted, item.CurrentState)
			}
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false, // Descending order
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			// Create multiple services
			for i := 0; i < 5; i++ {
				service := &domain.Service{
					Name:          "Paginated Service",
					CurrentState:  domain.ServiceStarted,
					AgentID:       agent.ID,
					ServiceTypeID: serviceType.ID,
					GroupID:       serviceGroup.ID,
				}
				err := repo.Create(context.Background(), service)
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

	t.Run("CountByGroup", func(t *testing.T) {
		// Create a service in the group
		service := &domain.Service{
			Name:          "Group Test Service",
			CurrentState:  domain.ServiceStarted,
			AgentID:       agent.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Test count by group
		count, err := repo.CountByGroup(context.Background(), serviceGroup.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))

		// Test count for non-existent group
		count, err = repo.CountByGroup(context.Background(), domain.NewUUID())
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
