package database

import (
	"context"
	"testing"
	"time"

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
		Name:          "Test Service",
		State:         domain.ServiceCreated,
		Attributes:    domain.Attributes{"key": []string{"value"}},
		Resources:     map[string]interface{}{"cpu": 1},
		AgentID:       agent.ID,
		ServiceTypeID: serviceType.ID,
		GroupID:       serviceGroup.ID,
	}
	require.NoError(t, serviceRepo.Create(context.Background(), service))

	t.Run("Create", func(t *testing.T) {
		job := &domain.Job{
			Type:        domain.JobServiceCreate,
			State:       domain.JobPending,
			AgentID:     agent.ID,
			ServiceID:   service.ID,
			Priority:    1,
			RequestData: domain.JSON{"serviceId": service.ID.String()},
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
			Type:        domain.JobServiceCreate,
			State:       domain.JobPending,
			AgentID:     agent.ID,
			ServiceID:   service.ID,
			Priority:    1,
			RequestData: domain.JSON{"serviceId": service.ID.String()},
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Find the job
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, job.ID, found.ID)
		assert.Equal(t, job.Type, found.Type)
		assert.Equal(t, job.State, found.State)
		assert.Equal(t, job.AgentID, found.AgentID)
		assert.Equal(t, job.ServiceID, found.ServiceID)
		assert.Equal(t, job.Priority, found.Priority)
		assert.Equal(t, job.RequestData, found.RequestData)

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
			Type:        domain.JobServiceCreate,
			State:       domain.JobPending,
			AgentID:     agent.ID,
			ServiceID:   service.ID,
			Priority:    1,
			RequestData: domain.JSON{"serviceId": service.ID.String()},
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Update the job
		job.State = domain.JobProcessing
		job.Priority = 2
		job.RequestData = domain.JSON{"serviceId": service.ID.String(), "updated": true}

		err = repo.Save(context.Background(), job)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, found.State)
		assert.Equal(t, 2, found.Priority)
		assert.Equal(t, domain.JSON{"serviceId": service.ID.String(), "updated": true}, found.RequestData)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a job
		job := &domain.Job{
			Type:        domain.JobServiceCreate,
			State:       domain.JobPending,
			AgentID:     agent.ID,
			ServiceID:   service.ID,
			Priority:    1,
			RequestData: domain.JSON{"serviceId": service.ID.String()},
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
				{Type: domain.JobServiceCreate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 1},
				{Type: domain.JobServiceUpdate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 2},
				{Type: domain.JobServiceDelete, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 3},
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
				Filters:  map[string][]string{"type": {string(domain.JobServiceCreate)}},
			}

			result, err := repo.List(context.Background(), page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, domain.JobServiceCreate, item.Type)
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
			{Type: domain.JobServiceCreate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 1},
			{Type: domain.JobServiceUpdate, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 2},
			{Type: domain.JobServiceDelete, State: domain.JobPending, AgentID: agent.ID, ServiceID: service.ID, Priority: 3},
		}
		for _, job := range pendingJobs {
			err := repo.Create(context.Background(), job)
			require.NoError(t, err)
		}

		// Create a processing job for the agent (shouldn't be returned)
		processingJob := &domain.Job{
			Type:      domain.JobServiceCreate,
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

	t.Run("ClaimJob", func(t *testing.T) {
		// Create a pending job
		job := &domain.Job{
			Type:      domain.JobServiceCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Claim the job
		err = repo.ClaimJob(context.Background(), job.ID, agent.ID)
		require.NoError(t, err)

		// Verify the job was claimed
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, found.State)
		assert.NotNil(t, found.ClaimedAt)

		// Try to claim a job that doesn't exist
		err = repo.ClaimJob(context.Background(), domain.NewUUID(), agent.ID)
		assert.Error(t, err)
		assert.IsType(t, domain.NotFoundError{}, err)

		// Try to claim a job that's already claimed
		err = repo.ClaimJob(context.Background(), job.ID, agent.ID)
		assert.Error(t, err)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("CompleteJob", func(t *testing.T) {
		// Create and claim a job
		job := &domain.Job{
			Type:      domain.JobServiceCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		err = repo.ClaimJob(context.Background(), job.ID, agent.ID)
		require.NoError(t, err)

		// Complete the job
		resultData := domain.JSON{"status": "success"}
		err = repo.CompleteJob(context.Background(), job.ID, resultData)
		require.NoError(t, err)

		// Verify the job was completed
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobCompleted, found.State)
		assert.Equal(t, resultData, found.ResultData)
		assert.NotNil(t, found.CompletedAt)

		// Try to complete a job that doesn't exist
		err = repo.CompleteJob(context.Background(), domain.NewUUID(), resultData)
		assert.Error(t, err)
		assert.IsType(t, domain.NotFoundError{}, err)

		// Try to complete a job that's already completed
		err = repo.CompleteJob(context.Background(), job.ID, resultData)
		assert.Error(t, err)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("FailJob", func(t *testing.T) {
		// Create and claim a job
		job := &domain.Job{
			Type:      domain.JobServiceCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		err = repo.ClaimJob(context.Background(), job.ID, agent.ID)
		require.NoError(t, err)

		// Fail the job
		errorMessage := "Resource not available"
		err = repo.FailJob(context.Background(), job.ID, errorMessage)
		require.NoError(t, err)

		// Verify the job was failed
		found, err := repo.FindByID(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobFailed, found.State)
		assert.Equal(t, errorMessage, found.ErrorMessage)
		assert.NotNil(t, found.CompletedAt)

		// Try to fail a job that doesn't exist
		err = repo.FailJob(context.Background(), domain.NewUUID(), errorMessage)
		assert.Error(t, err)
		assert.IsType(t, domain.NotFoundError{}, err)

		// Try to fail a job that's already failed
		err = repo.FailJob(context.Background(), job.ID, errorMessage)
		assert.Error(t, err)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("ReleaseStuckJobs", func(t *testing.T) {
		// Create a clean context for this test
		ctx := context.Background()

		// Create a recently claimed job (should not be released)
		recentJob := &domain.Job{
			Type:      domain.JobServiceCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
			Priority:  1,
		}
		err := repo.Create(ctx, recentJob)
		require.NoError(t, err)

		// Before trying to claim, verify job exists and has correct state
		foundJob, findErr := repo.FindByID(ctx, recentJob.ID)
		require.NoError(t, findErr, "Job should exist before claiming")
		require.Equal(t, domain.JobPending, foundJob.State, "Job state should be pending before claiming")

		// Now try to claim the job
		err = repo.ClaimJob(ctx, recentJob.ID, agent.ID)
		require.NoError(t, err, "Should be able to claim the recent job")

		// Create and claim multiple jobs in the past
		oldTime := time.Now().Add(-time.Hour)
		stuckJobs := []*domain.Job{
			{Type: domain.JobServiceCreate, State: domain.JobProcessing, AgentID: agent.ID, ServiceID: service.ID, ClaimedAt: &oldTime},
			{Type: domain.JobServiceUpdate, State: domain.JobProcessing, AgentID: agent.ID, ServiceID: service.ID, ClaimedAt: &oldTime},
		}
		for _, job := range stuckJobs {
			err := repo.Create(ctx, job)
			require.NoError(t, err)
		}

		// Release stuck jobs that have been processing for more than 30 minutes
		count, err := repo.ReleaseStuckJobs(ctx, 30)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 2)

		// Verify stuck jobs were reset to pending
		for _, job := range stuckJobs {
			found, err := repo.FindByID(ctx, job.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.JobPending, found.State)
			assert.Nil(t, found.ClaimedAt)
		}

		// Verify recent job is still processing
		found, err := repo.FindByID(ctx, recentJob.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, found.State)
		assert.NotNil(t, found.ClaimedAt)
	})

	t.Run("DeleteOldCompletedJobs", func(t *testing.T) {
		// Create completed old multiple jobs
		oldTime := time.Now().AddDate(0, 0, -10) // 10 days ago
		oldJobs := []*domain.Job{
			{Type: domain.JobServiceCreate, State: domain.JobCompleted, AgentID: agent.ID, ServiceID: service.ID, ClaimedAt: &oldTime, CompletedAt: &oldTime},
			{Type: domain.JobServiceUpdate, State: domain.JobCompleted, AgentID: agent.ID, ServiceID: service.ID, ClaimedAt: &oldTime, CompletedAt: &oldTime},
		}
		for _, job := range oldJobs {
			err := repo.Create(context.Background(), job)
			require.NoError(t, err)
		}

		// Create a recently completed job (should not be deleted)
		recentJob := &domain.Job{
			Type:      domain.JobServiceCreate,
			State:     domain.JobPending,
			AgentID:   agent.ID,
			ServiceID: service.ID,
		}
		err := repo.Create(context.Background(), recentJob)
		require.NoError(t, err)
		err = repo.ClaimJob(context.Background(), recentJob.ID, agent.ID)
		require.NoError(t, err)
		err = repo.CompleteJob(context.Background(), recentJob.ID, domain.JSON{"status": "success"})
		require.NoError(t, err)

		// Delete completed jobs older than 7 days
		count, err := repo.DeleteOldCompletedJobs(context.Background(), 7)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Verify old jobs were deleted
		for _, job := range oldJobs {
			_, err := repo.FindByID(context.Background(), job.ID)
			assert.Error(t, err)
			assert.IsType(t, domain.NotFoundError{}, err)
		}

		// Verify recent job still exists
		found, err := repo.FindByID(context.Background(), recentJob.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobCompleted, found.State)
	})
}
