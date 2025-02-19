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
	serviceGroup := &domain.ServiceGroup{
		Name: "Test Service Group",
	}
	require.NoError(t, serviceGroupRepo.Create(context.Background(), serviceGroup))

	t.Run("Create", func(t *testing.T) {
		service := &domain.Service{
			Name:          "Test Service",
			State:         domain.ServiceCreated,
			Attributes:    domain.Attributes{"key": []string{"value"}},
			Resources:     map[string]interface{}{"cpu": 1},
			AgentID:       agent.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
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
			Name:          "Test Service",
			State:         domain.ServiceCreated,
			Attributes:    domain.Attributes{"key": []string{"value"}},
			Resources:     map[string]interface{}{"cpu": "1"},
			AgentID:       agent.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Find the service
		found, err := repo.FindByID(context.Background(), service.ID)
		require.NoError(t, err)
		assert.Equal(t, service.ID, found.ID)
		assert.Equal(t, service.Name, found.Name)
		assert.Equal(t, service.State, found.State)
		assert.Equal(t, service.Attributes, found.Attributes)
		assert.Equal(t, domain.JSON{"cpu": "1"}, found.Resources)
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
			Name:          "Test Service",
			State:         domain.ServiceCreated,
			Attributes:    domain.Attributes{"key": []string{"value"}},
			Resources:     map[string]interface{}{"cpu": "1"},
			AgentID:       agent.ID,
			ServiceTypeID: serviceType.ID,
			GroupID:       serviceGroup.ID,
		}
		err := repo.Create(context.Background(), service)
		require.NoError(t, err)

		// Update the service
		service.Name = "Updated Service"
		service.State = domain.ServiceUpdated
		service.Attributes = domain.Attributes{"key": []string{"updated"}}
		service.Resources = map[string]interface{}{"cpu": "2"}

		err = repo.Save(context.Background(), service)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.FindByID(context.Background(), service.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Service", found.Name)
		assert.Equal(t, domain.ServiceUpdated, found.State)
		assert.Equal(t, domain.Attributes{"key": []string{"updated"}}, found.Attributes)
		assert.Equal(t, domain.JSON{"cpu": "2"}, found.Resources)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a service
		service := &domain.Service{
			Name:          "Test Service",
			State:         domain.ServiceCreated,
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
		// Create multiple services
		services := []*domain.Service{
			{Name: "Service A", State: domain.ServiceCreated, AgentID: agent.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
			{Name: "Service B", State: domain.ServiceUpdated, AgentID: agent.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
			{Name: "Service C", State: domain.ServiceCreated, AgentID: agent.ID, ServiceTypeID: serviceType.ID, GroupID: serviceGroup.ID},
		}
		for _, service := range services {
			err := repo.Create(context.Background(), service)
			require.NoError(t, err)
		}

		// Test listing with pagination
		result, err := repo.List(context.Background(), nil, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 3)

		// Test listing with name filter
		result, err = repo.List(context.Background(), &domain.SimpleFilter{Field: "name", Value: "Service A"}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "Service A", result.Items[0].Name)

		// Test listing with state filter
		result, err = repo.List(context.Background(), &domain.SimpleFilter{Field: "state", Value: string(domain.ServiceUpdated)}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 1)
		assert.Equal(t, domain.ServiceUpdated, result.Items[0].State)
	})

	t.Run("CountByGroup", func(t *testing.T) {
		// Create a service in the group
		service := &domain.Service{
			Name:          "Group Test Service",
			State:         domain.ServiceCreated,
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
