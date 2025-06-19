package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/pkg/domain"
)

func TestServiceRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceRepository(testDB.DB)

	// Create participants (for consumer and provider roles)
	participantRepo := NewParticipantRepository(testDB.DB)
	consumer := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(context.Background(), consumer))

	// Create provider participant
	provider := createTestParticipant(t, domain.ParticipantEnabled)
	provider.Name = "Test Provider"
	require.NoError(t, participantRepo.Create(context.Background(), provider))

	// Create dependencies first
	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	agentType := &domain.AgentType{
		Name: "Test Agent Type",
	}
	require.NoError(t, agentTypeRepo.Create(context.Background(), agentType))

	agentRepo := NewAgentRepository(testDB.DB)
	agent := &domain.Agent{
		Name:        "Test Agent",
		Status:      domain.AgentConnected,
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
	serviceGroup := createTestServiceGroup(t, consumer.ID)
	require.NoError(t, serviceGroupRepo.Create(context.Background(), serviceGroup))

	t.Run("Create", func(t *testing.T) {
		service := &domain.Service{
			Name:              "Test Service",
			CurrentStatus:     domain.ServiceStarted,
			CurrentProperties: &(properties.JSON{"key": "value"}),
			Resources:         &(properties.JSON{"cpu": "1"}),
			AgentID:           agent.ID,
			ProviderID:        provider.ID, // Set ProviderID to the created provider's ID
			ConsumerID:        consumer.ID, // Set ConsumerID to the created consumer's ID
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup.ID,
		}

		err := repo.Create(context.Background(), service)
		require.NoError(t, err)
		assert.NotEmpty(t, service.ID)
		assert.NotZero(t, service.CreatedAt)
		assert.NotZero(t, service.UpdatedAt)
	})

	t.Run("Get", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:              "Test Service",
			CurrentStatus:     domain.ServiceStarted,
			CurrentProperties: &(properties.JSON{"key": "value"}),
			Resources:         &(properties.JSON{"cpu": "1"}),
			AgentID:           agent.ID,
			ProviderID:        provider.ID, // Set ProviderID
			ConsumerID:        consumer.ID, // Set ConsumerID
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Find the service
		found, err := repo.Get(context.Background(), service.ID)
		require.NoError(t, err)
		assert.Equal(t, service.ID, found.ID)
		assert.Equal(t, service.Name, found.Name)
		assert.Equal(t, service.CurrentStatus, found.CurrentStatus)
		assert.Equal(t, service.CurrentProperties, found.CurrentProperties)
		assert.Equal(t, &(properties.JSON{"cpu": "1"}), found.Resources)
		assert.Equal(t, service.AgentID, found.AgentID)
		assert.Equal(t, service.ServiceTypeID, found.ServiceTypeID)
		assert.Equal(t, service.GroupID, found.GroupID)

		// Check relationships are loaded
		assert.NotNil(t, found.Agent)
		assert.NotNil(t, found.ServiceType)
		assert.NotNil(t, found.Group)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		found, err := repo.Get(context.Background(), properties.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:              "Test Service",
			CurrentStatus:     domain.ServiceStarted,
			CurrentProperties: &(properties.JSON{"key": "value"}),
			Resources:         &(properties.JSON{"cpu": "1"}),
			AgentID:           agent.ID,
			ProviderID:        provider.ID, // Set ProviderID
			ConsumerID:        consumer.ID, // Set ConsumerID
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Update the service
		service.Name = "Updated Service"
		service.CurrentStatus = domain.ServiceStarted
		service.CurrentProperties = &(properties.JSON{"key": "value"})
		service.Resources = &(properties.JSON{"cpu": "2"})

		err = repo.Save(context.Background(), service)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.Get(context.Background(), service.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Service", found.Name)
		assert.Equal(t, domain.ServiceStarted, found.CurrentStatus)
		assert.Equal(t, &(properties.JSON{"key": "value"}), found.CurrentProperties)
		assert.Equal(t, &(properties.JSON{"cpu": "2"}), found.Resources)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:          "Test Service",
			CurrentStatus: domain.ServiceStarted,
			AgentID:       agent.ID,
			ProviderID:    provider.ID, // Set ProviderID
			ConsumerID:    consumer.ID, // Set ConsumerID
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Delete the service
		err = repo.Delete(context.Background(), service.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.Get(context.Background(), service.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple services
			services := []*domain.Service{
				{Name: "Service A", CurrentStatus: domain.ServiceStarted, AgentID: agent.ID, ProviderID: provider.ID, ConsumerID: consumer.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
				{Name: "Service B", CurrentStatus: domain.ServiceStarted, AgentID: agent.ID, ProviderID: provider.ID, ConsumerID: consumer.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
				{Name: "Service C", CurrentStatus: domain.ServiceStarted, AgentID: agent.ID, ProviderID: provider.ID, ConsumerID: consumer.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
			}
			for _, service := range services {
				err := repo.Create(context.Background(), service)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
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

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 1)
			assert.Equal(t, "Service A", result.Items[0].Name)
		})

		t.Run("success - list with status filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"currentStatus": {string(domain.ServiceStarted)}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, domain.ServiceStarted, item.CurrentStatus)
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

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
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
					CurrentStatus: domain.ServiceStarted,
					ProviderID:    provider.ID, // Set ProviderID
					ConsumerID:    consumer.ID, // Set ConsumerID
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

	t.Run("CountByGroup", func(t *testing.T) {
		// Create a service in the group
		service := &domain.Service{
			Name:          "Group Test Service",
			CurrentStatus: domain.ServiceStarted,
			AgentID:       agent.ID,
			ProviderID:    provider.ID, // Set ProviderID
			ConsumerID:    consumer.ID, // Set ConsumerID
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
		count, err = repo.CountByGroup(context.Background(), properties.NewUUID())
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("CountByAgent", func(t *testing.T) {
		// Create a service for the agent
		service := &domain.Service{
			Name:          "Agent Test Service",
			CurrentStatus: domain.ServiceStarted,
			AgentID:       agent.ID,
			ProviderID:    provider.ID,
			ConsumerID:    consumer.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Test count by agent
		count, err := repo.CountByAgent(context.Background(), agent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1), "Should find at least one service for the agent")

		// Test count for non-existent agent
		nonExistentAgentID := properties.NewUUID()
		count, err = repo.CountByAgent(context.Background(), nonExistentAgentID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "Should return zero for non-existent agent")
	})

	t.Run("FindByExternalID", func(t *testing.T) {
		// Create a service with an external ID
		externalID := "ext-123456"
		service := &domain.Service{
			Name:          "External ID Test Service",
			CurrentStatus: domain.ServiceStarted,
			ExternalID:    &externalID,
			AgentID:       agent.ID,
			ProviderID:    provider.ID,
			ConsumerID:    consumer.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Test finding by external ID
		found, err := repo.FindByExternalID(context.Background(), agent.ID, externalID)
		require.NoError(t, err)
		assert.NotNil(t, found, "Should find service by external ID")
		assert.Equal(t, service.ID, found.ID, "IDs should match")
		assert.Equal(t, externalID, *found.ExternalID, "External IDs should match")
		assert.Equal(t, agent.ID, found.AgentID, "Agent IDs should match")

		// Test with valid agent ID but non-existent external ID
		nonExistentExternalID := "non-existent-id"
		found, err = repo.FindByExternalID(context.Background(), agent.ID, nonExistentExternalID)
		assert.Error(t, err, "Should return error for non-existent external ID")
		assert.IsType(t, domain.NotFoundError{}, err, "Error should be NotFoundError")
		assert.Nil(t, found, "Result should be nil")

		// Test with non-existent agent ID but valid external ID
		nonExistentAgentID := properties.NewUUID()
		found, err = repo.FindByExternalID(context.Background(), nonExistentAgentID, externalID)
		assert.Error(t, err, "Should return error for non-existent agent ID")
		assert.IsType(t, domain.NotFoundError{}, err, "Error should be NotFoundError")
		assert.Nil(t, found, "Result should be nil")
	})

	t.Run("AuthScope", func(t *testing.T) {
		service := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID, provider.ID, consumer.ID)
		require.NoError(t, repo.Create(context.Background(), service))

		scope, err := repo.AuthScope(context.Background(), service.ID)
		require.NoError(t, err)

		// Check that the returned scope is a auth.DefaultObjectScope
		defaultScope, ok := scope.(*auth.DefaultObjectScope)
		require.True(t, ok, "AuthScope should return a auth.DefaultObjectScope")
		assert.Equal(t, provider.ID, *defaultScope.ProviderID, "Provider ID should match")
		assert.Equal(t, consumer.ID, *defaultScope.ConsumerID, "Consumer ID should match")
		assert.Equal(t, agent.ID, *defaultScope.AgentID, "Agent ID should match")
	})
}
