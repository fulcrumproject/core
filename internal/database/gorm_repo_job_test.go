package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestJobRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewJobRepository(testDB.DB)

	// Create dependencies first
	providerRepo := NewProviderRepository(testDB.DB)
	provider := &domain.Provider{
		Name:  "Test Provider",
		State: domain.ProviderEnabled,
	}
	require.NoError(t, providerRepo.Create(context.Background(), provider))

	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	agentType := &domain.AgentType{
		Name: "Test Agent Type",
	}
	require.NoError(t, agentTypeRepo.Create(context.Background(), agentType))

	agentRepo := NewAgentRepository(testDB.DB)
	agent := &domain.Agent{
		Name:        "Test Agent",
		State:       domain.AgentConnected,
		TokenHash:   "test-token-hash",
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

	serviceRepo := NewServiceRepository(testDB.DB)
	service := &domain.Service{
		Name:              "Test Service",
		CurrentState:      domain.ServiceStarted,
		CurrentProperties: &(domain.JSON{"key": "value"}),
		Attributes:        domain.Attributes{"key": []string{"value"}},
		Resources:         map[string]interface{}{"cpu": 1},
		AgentID:           agent.ID,
		ServiceTypeID:     serviceType.ID,
		GroupID:           serviceGroup.ID,
	}
	require.NoError(t, serviceRepo.Create(context.Background(), service))

	t.Run("Create", func(t *testing.T) {
		job := &domain.Job{
			Action:    domain.ServiceActionCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}

		err := repo.Create(context.Background(), job)
		require.NoError(t, err)
		assert.NotEmpty(t, job.ID)
		assert.NotZero(t, job.CreatedAt)
		assert.NotZero(t, job.UpdatedAt)
	})

	t.Run("FindByID", func(t *testing.T) {
		// Create a job
		job := &domain.Job{
			Action:    domain.ServiceActionCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Find the job
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, job.ID, found.ID)
		assert.Equal(t, job.Action, found.Action)
		assert.Equal(t, job.State, found.State)
		assert.Equal(t, job.AgentID, found.AgentID)
		assert.Equal(t, job.ServiceID, found.ServiceID)
		assert.Equal(t, job.Priority, found.Priority)

		// Check relationships are loaded
		assert.NotNil(t, found.Agent)
		assert.NotNil(t, found.Service)
	})

	t.Run("FindByID_NotFound", func(t *testing.T) {
		found, err := repo.FindByID(context.Background(), domain.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a job
		job := &domain.Job{
			Action:    domain.ServiceActionCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Update the job
		job.State = domain.JobProcessing
		job.Priority = 2

		err = repo.Save(context.Background(), job)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, found.State)
		assert.Equal(t, 2, found.Priority)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a job
		job := &domain.Job{
			Action:    domain.ServiceActionCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Delete the job
		err = repo.Delete(context.Background(), job.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByID(context.Background(), job.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple jobs
			jobs := []*domain.Job{
				{Action: domain.ServiceActionCreate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 1},
				{Action: domain.ServiceActionColdUpdate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 2},
				{Action: domain.ServiceActionDelete, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 3},
			}
			for _, job := range jobs {
				err := repo.Create(context.Background(), job)
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
		})

		t.Run("success - list with state filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"state": {string(domain.JobPending)}},
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, domain.JobPending, item.State)
			}
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"action": {string(domain.ServiceActionCreate)}},
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, domain.ServiceActionCreate, item.Action)
			}
		})

		t.Run("success - list with sorting by priority", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "priority",
				SortAsc:  false, // Descending order
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Priority, result.Items[i].Priority)
			}
		})
	})

	t.Run("GetPendingJobsForAgent", func(t *testing.T) {
		// Create multiple pending jobs for the agent
		pendingJobs := []*domain.Job{
			{Action: domain.ServiceActionCreate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 1},
			{Action: domain.ServiceActionHotUpdate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 2},
			{Action: domain.ServiceActionDelete, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 3},
		}
		for _, job := range pendingJobs {
			err := repo.Create(context.Background(), job)
			require.NoError(t, err)
		}

		// Create a processing job for the agent (shouldn't be returned)
		processingJob := &domain.Job{
			Action:    domain.ServiceActionCreate,
			State:     domain.JobProcessing,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  4,
		}
		err := repo.Create(context.Background(), processingJob)
		require.NoError(t, err)

		// Test fetching pending jobs
		jobs, err := repo.GetPendingJobsForAgent(context.Background(), agent.ID, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 3)

		// Verify all returned jobs are pending
		for _, job := range jobs {
			assert.Equal(t, domain.JobPending, job.State)
		}

		// Test limit
		limitedJobs, err := repo.GetPendingJobsForAgent(context.Background(), agent.ID, 2)
		require.NoError(t, err)
		assert.Len(t, limitedJobs, 2)

		// Verify priority ordering (highest first)
		assert.GreaterOrEqual(t, limitedJobs[0].Priority, limitedJobs[1].Priority)
	})
}
