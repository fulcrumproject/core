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

	// Create participants (needed for service group and provider role)
	participantRepo := NewParticipantRepository(testDB.DB)
	consumer := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(context.Background(), consumer))

	// Create dependencies first
	provider := createTestParticipant(t, domain.ParticipantEnabled)
	provider.Name = "Test Provider"
	require.NoError(t, participantRepo.Create(context.Background(), provider))

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
		Name:          "Test Service Group",
		ParticipantID: consumer.ID,
	}
	require.NoError(t, serviceGroupRepo.Create(context.Background(), serviceGroup))

	serviceRepo := NewServiceRepository(testDB.DB)
	service := &domain.Service{
		Name:              "Test Service",
		CurrentState:      domain.ServiceStarted,
		CurrentProperties: &(domain.JSON{"key": "value"}),
		Attributes:        domain.Attributes{"key": []string{"value"}},
		Resources:         &(domain.JSON{"cpu": 1}),
		AgentID:           agent.ID,
		ServiceTypeID:     serviceType.ID,
		GroupID:           serviceGroup.ID,
		ConsumerID:        consumer.ID,
		ProviderID:        provider.ID,
	}
	require.NoError(t, serviceRepo.Create(context.Background(), service))

	t.Run("Create", func(t *testing.T) {
		job := domain.NewJob(service, domain.ServiceActionCreate, 1)

		// Use the existing err variable
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)
		assert.NotEmpty(t, job.ID)
		assert.NotZero(t, job.CreatedAt)
		assert.NotZero(t, job.UpdatedAt)
	})

	t.Run("FindByID", func(t *testing.T) {
		// Create a job
		job := domain.NewJob(service, domain.ServiceActionCreate, 1)
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
		job := domain.NewJob(service, domain.ServiceActionCreate, 1)
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
		job := domain.NewJob(service, domain.ServiceActionCreate, 1)
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
			job1 := domain.NewJob(service, domain.ServiceActionCreate, 1)
			job2 := domain.NewJob(service, domain.ServiceActionColdUpdate, 2)
			job3 := domain.NewJob(service, domain.ServiceActionDelete, 3)

			jobs := []*domain.Job{job1, job2, job3}
			for _, job := range jobs {
				err := repo.Create(context.Background(), job)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), &domain.EmptyAuthIdentityScope, page)
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

			result, err := repo.List(context.Background(), &domain.EmptyAuthIdentityScope, page)
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

			result, err := repo.List(context.Background(), &domain.EmptyAuthIdentityScope, page)
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

			result, err := repo.List(context.Background(), &domain.EmptyAuthIdentityScope, page)
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
		job1 := domain.NewJob(service, domain.ServiceActionCreate, 1)
		job2 := domain.NewJob(service, domain.ServiceActionHotUpdate, 2)
		job3 := domain.NewJob(service, domain.ServiceActionDelete, 3)

		pendingJobs := []*domain.Job{job1, job2, job3}
		for _, job := range pendingJobs {
			err := repo.Create(context.Background(), job)
			require.NoError(t, err)
		}

		// Create a processing job for the agent (shouldn't be returned)
		processingJob := domain.NewJob(service, domain.ServiceActionCreate, 4)
		processingJob.State = domain.JobProcessing
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

	t.Run("GetTimeOutJobs", func(t *testing.T) {
		// Create a job in processing state with an old created_at time
		now := time.Now()
		oldTime := now.Add(-2 * time.Hour) // 2 hours ago

		oldJob := domain.NewJob(service, domain.ServiceActionCreate, 1)
		oldJob.State = domain.JobProcessing
		// Set BaseEntity.CreatedAt directly since it's normally set during Insert
		oldJob.BaseEntity = domain.BaseEntity{
			CreatedAt: oldTime,
		}
		err := repo.Create(context.Background(), oldJob)
		require.NoError(t, err)

		// Create a job in processing state with a recent created_at time (will use current time)
		newJob := domain.NewJob(service, domain.ServiceActionStart, 2)
		require.NoError(t, err)
		newJob.State = domain.JobProcessing
		newJob.ClaimedAt = &now // use current time for claimed time
		err = repo.Create(context.Background(), newJob)
		require.NoError(t, err)

		// Call GetTimeOutJobs with a 1 hour threshold
		timedOutJobs, err := repo.GetTimeOutJobs(context.Background(), 1*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(timedOutJobs)) // Only the old job should be returned
		assert.Equal(t, oldJob.ID, timedOutJobs[0].ID)

		// Verify the recent job was not returned as timed out
		updatedNewJob, err := repo.FindByID(context.Background(), newJob.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, updatedNewJob.State)
		assert.NotContains(t, timedOutJobs, newJob.ID)
	})

	t.Run("DeleteOldCompletedJobs", func(t *testing.T) {
		// Create completed jobs with varying completion times
		now := time.Now()

		// Create jobs with completion times at different intervals
		oldCompletedJob := domain.NewJob(service, domain.ServiceActionStop, 1)
		oldCompletedJob.State = domain.JobCompleted
		oldCompletedTime := now.Add(-48 * time.Hour) // 2 days ago
		oldCompletedJob.CompletedAt = &oldCompletedTime
		require.NoError(t, repo.Create(context.Background(), oldCompletedJob))

		oldFailedJob := domain.NewJob(service, domain.ServiceActionStart, 1)
		oldFailedJob.State = domain.JobFailed
		oldFailedTime := now.Add(-36 * time.Hour) // 1.5 days ago
		oldFailedJob.CompletedAt = &oldFailedTime
		require.NoError(t, repo.Create(context.Background(), oldFailedJob))

		recentCompletedJob := domain.NewJob(service, domain.ServiceActionHotUpdate, 1)
		recentCompletedJob.State = domain.JobCompleted
		recentCompletedTime := now.Add(-12 * time.Hour) // 12 hours ago
		recentCompletedJob.CompletedAt = &recentCompletedTime
		require.NoError(t, repo.Create(context.Background(), recentCompletedJob))

		pendingJob := domain.NewJob(service, domain.ServiceActionHotUpdate, 1)
		pendingJob.State = domain.JobPending
		require.NoError(t, repo.Create(context.Background(), pendingJob))

		// Call DeleteOldCompletedJobs with a 24-hour threshold
		count, err := repo.DeleteOldCompletedJobs(context.Background(), 24*time.Hour)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, count, "Should delete exactly two old jobs")

		// Verify that old jobs were deleted
		_, err = repo.FindByID(context.Background(), oldCompletedJob.ID)
		assert.Error(t, err, "Old completed job should be deleted")
		assert.IsType(t, domain.NotFoundError{}, err)

		_, err = repo.FindByID(context.Background(), oldFailedJob.ID)
		assert.Error(t, err, "Old failed job should be deleted")
		assert.IsType(t, domain.NotFoundError{}, err)

		// Verify that recent and pending jobs still exist
		stillExists, err := repo.FindByID(context.Background(), recentCompletedJob.ID)
		assert.NoError(t, err, "Recent completed job should still exist")
		assert.Equal(t, recentCompletedJob.ID, stillExists.ID)

		stillExists, err = repo.FindByID(context.Background(), pendingJob.ID)
		assert.NoError(t, err, "Pending job should still exist")
		assert.Equal(t, pendingJob.ID, stillExists.ID)
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns correct auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a new job with known IDs
			job := domain.NewJob(service, domain.ServiceActionCreate, 1)

			// The job should have provider, agent, and broker IDs from the service
			require.NotNil(t, service.ProviderID)
			require.NotNil(t, service.AgentID)
			require.NotNil(t, service.ConsumerID)

			// Save job to database
			require.NoError(t, repo.Create(ctx, job))

			// Get job's auth scope
			scope, err := repo.AuthScope(ctx, job.ID)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, scope, "AuthScope should not return nil")

			// Verify auth scope contains the correct IDs
			assert.NotNil(t, scope.ProviderID, "ProviderID should not be nil")
			assert.Equal(t, service.ProviderID, *scope.ProviderID, "Should return the correct provider ID")
			assert.NotNil(t, scope.ConsumerID, "ConsumerID should not be nil")
			assert.Equal(t, service.ConsumerID, *scope.ConsumerID, "Should return the correct consumer ID")
			assert.NotNil(t, scope.AgentID, "AgentID should not be nil")
			assert.Equal(t, service.AgentID, *scope.AgentID, "Should return the correct agent ID")
		})
	})
}
