package database

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fulcrumproject/core/pkg/domain"
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
	serviceGroup := &domain.ServiceGroup{
		Name:       "Test Service Group",
		ConsumerID: consumer.ID,
	}
	require.NoError(t, serviceGroupRepo.Create(context.Background(), serviceGroup))

	serviceRepo := NewServiceRepository(testDB.DB)
	service := &domain.Service{
		Name:              "Test Service",
		Status:            "Started",
		Properties:        &(properties.JSON{"key": "value"}),
		AgentInstanceData: &(properties.JSON{"cpu": 1}),
		AgentID:           agent.ID,
		ServiceTypeID:     serviceType.ID,
		GroupID:           serviceGroup.ID,
		ConsumerID:        consumer.ID,
		ProviderID:        provider.ID,
	}
	require.NoError(t, serviceRepo.Create(context.Background(), service))

	t.Run("create", func(t *testing.T) {
		job := domain.NewJob(service, "create", nil, 1)

		// Use the existing err variable
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)
		assert.NotEmpty(t, job.ID)
		assert.NotZero(t, job.CreatedAt)
		assert.NotZero(t, job.UpdatedAt)
	})

	t.Run("Get", func(t *testing.T) {
		// Create a job
		job := domain.NewJob(service, "create", nil, 1)
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Find the job
		found, err := repo.Get(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, job.ID, found.ID)
		assert.Equal(t, job.Action, found.Action)
		assert.Equal(t, job.Status, found.Status)
		assert.Equal(t, job.AgentID, found.AgentID)
		assert.Equal(t, job.ServiceID, found.ServiceID)
		assert.Equal(t, job.Priority, found.Priority)

		// Check relationships are loaded
		assert.NotNil(t, found.Agent)
		assert.NotNil(t, found.Service)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		found, err := repo.Get(context.Background(), properties.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a job
		job := domain.NewJob(service, "create", nil, 1)
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Update the job
		job.Status = domain.JobProcessing
		job.Priority = 2

		err = repo.Save(context.Background(), job)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.Get(context.Background(), job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, found.Status)
		assert.Equal(t, 2, found.Priority)
	})

	t.Run("delete", func(t *testing.T) {
		// Create a job
		job := domain.NewJob(service, "create", nil, 1)
		err := repo.Create(context.Background(), job)
		require.NoError(t, err)

		// Delete the job
		err = repo.Delete(context.Background(), job.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.Get(context.Background(), job.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple jobs
			job1 := domain.NewJob(service, "create", nil, 1)
			job2 := domain.NewJob(service, "update", nil, 2)
			job3 := domain.NewJob(service, "delete", nil, 3)

			jobs := []*domain.Job{job1, job2, job3}
			for _, job := range jobs {
				err := repo.Create(context.Background(), job)
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
		})

		t.Run("success - list with status filter", func(t *testing.T) {
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"status": {string(domain.JobPending)}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, domain.JobPending, item.Status)
			}
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"action": {string("create")}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, "create", item.Action)
			}
		})

		t.Run("success - list with sorting by priority", func(t *testing.T) {
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "priority",
				SortAsc:  false, // Descending order
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Priority, result.Items[i].Priority)
			}
		})
	})

	t.Run("GetPendingJobsForAgent", func(t *testing.T) {
		// Create a second service group
		serviceGroup2 := &domain.ServiceGroup{
			Name:       "Test Service Group 2",
			ConsumerID: consumer.ID,
		}
		require.NoError(t, serviceGroupRepo.Create(context.Background(), serviceGroup2))

		// Create a second service in the second service group
		service2 := &domain.Service{
			Name:              "Test Service 2",
			Status:            "Started",
			Properties:        &(properties.JSON{"key": "value2"}),
			AgentInstanceData: &(properties.JSON{"cpu": 2}),
			AgentID:           agent.ID,
			ServiceTypeID:     serviceType.ID,
			GroupID:           serviceGroup2.ID,
			ConsumerID:        consumer.ID,
			ProviderID:        provider.ID,
		}
		require.NoError(t, serviceRepo.Create(context.Background(), service2))

		// Create multiple pending jobs for the first service (same service group)
		job1 := domain.NewJob(service, "create", nil, 1)
		job2 := domain.NewJob(service, "update", nil, 2)
		job3 := domain.NewJob(service, "delete", nil, 3)

		// Create multiple pending jobs for the second service (different service group)
		job4 := domain.NewJob(service2, "start", nil, 1)
		job5 := domain.NewJob(service2, "stop", nil, 4)

		pendingJobs := []*domain.Job{job1, job2, job3, job4, job5}
		for _, job := range pendingJobs {
			err := repo.Create(context.Background(), job)
			require.NoError(t, err)
		}

		// Create a processing job for the first service group (should exclude this group from results)
		processingJob := domain.NewJob(service, "create", nil, 4)
		processingJob.Status = domain.JobProcessing
		err := repo.Create(context.Background(), processingJob)
		require.NoError(t, err)

		// Test fetching pending jobs - should return only jobs from service groups without processing jobs
		jobs, err := repo.GetPendingJobsForAgent(context.Background(), agent.ID, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jobs), "Should return exactly 1 job (only from service group 2, since group 1 has a processing job)")

		// Verify all returned jobs are pending
		for _, job := range jobs {
			assert.Equal(t, domain.JobPending, job.Status)
		}

		// Verify that we only got jobs from the second service group (the one without processing jobs)
		// job5 has priority 4 (highest in second group)
		assert.Equal(t, 4, jobs[0].Priority, "Should contain job with priority 4 from second service group")
		assert.Equal(t, service2.ID, jobs[0].ServiceID, "Should be from the second service")
		assert.Equal(t, job5.ID, jobs[0].ID, "Should be the expected jobID")

		// Test limit
		limitedJobs, err := repo.GetPendingJobsForAgent(context.Background(), agent.ID, 1)
		require.NoError(t, err)
		assert.Len(t, limitedJobs, 1, "Should respect the limit")

		// Now let's test the reverse scenario - create a processing job in the second service group
		processingJob2 := domain.NewJob(service2, "start", nil, 5)
		processingJob2.Status = domain.JobProcessing
		err = repo.Create(context.Background(), processingJob2)
		require.NoError(t, err)

		// Test fetching pending jobs again - should return no jobs since both groups have processing jobs
		jobs2, err := repo.GetPendingJobsForAgent(context.Background(), agent.ID, 10)
		require.NoError(t, err)
		assert.Equal(t, 0, len(jobs2), "Should return no jobs since both service groups have processing jobs")
	})

	t.Run("GetTimeOutJobs", func(t *testing.T) {
		// Create a job in processing status with an old created_at time
		now := time.Now()
		oldTime := now.Add(-2 * time.Hour) // 2 hours ago

		oldJob := domain.NewJob(service, "create", nil, 1)
		oldJob.Status = domain.JobProcessing
		// Set BaseEntity.CreatedAt directly since it's normally set during Insert
		oldJob.BaseEntity = domain.BaseEntity{
			CreatedAt: oldTime,
		}
		err := repo.Create(context.Background(), oldJob)
		require.NoError(t, err)

		// Create a job in processing status with a recent created_at time (will use current time)
		newJob := domain.NewJob(service, "start", nil, 2)
		require.NoError(t, err)
		newJob.Status = domain.JobProcessing
		newJob.ClaimedAt = &now // use current time for claimed time
		err = repo.Create(context.Background(), newJob)
		require.NoError(t, err)

		// Call GetTimeOutJobs with a 1 hour threshold
		timedOutJobs, err := repo.GetTimeOutJobs(context.Background(), 1*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(timedOutJobs)) // Only the old job should be returned
		assert.Equal(t, oldJob.ID, timedOutJobs[0].ID)

		// Verify the recent job was not returned as timed out
		updatedNewJob, err := repo.Get(context.Background(), newJob.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobProcessing, updatedNewJob.Status)
		assert.NotContains(t, timedOutJobs, newJob.ID)
	})

	t.Run("DeleteOldCompletedJobs", func(t *testing.T) {
		// Create completed jobs with varying completion times
		now := time.Now()

		// Create jobs with completion times at different intervals
		oldCompletedJob := domain.NewJob(service, "stop", nil, 1)
		oldCompletedJob.Status = domain.JobCompleted
		oldCompletedTime := now.Add(-48 * time.Hour) // 2 days ago
		oldCompletedJob.CompletedAt = &oldCompletedTime
		require.NoError(t, repo.Create(context.Background(), oldCompletedJob))

		oldFailedJob := domain.NewJob(service, "start", nil, 1)
		oldFailedJob.Status = domain.JobFailed
		oldFailedTime := now.Add(-36 * time.Hour) // 1.5 days ago
		oldFailedJob.CompletedAt = &oldFailedTime
		require.NoError(t, repo.Create(context.Background(), oldFailedJob))

		recentCompletedJob := domain.NewJob(service, "update", nil, 1)
		recentCompletedJob.Status = domain.JobCompleted
		recentCompletedTime := now.Add(-12 * time.Hour) // 12 hours ago
		recentCompletedJob.CompletedAt = &recentCompletedTime
		require.NoError(t, repo.Create(context.Background(), recentCompletedJob))

		pendingJob := domain.NewJob(service, "update", nil, 1)
		pendingJob.Status = domain.JobPending
		require.NoError(t, repo.Create(context.Background(), pendingJob))

		// Call DeleteOldCompletedJobs with a 24-hour threshold
		count, err := repo.DeleteOldCompletedJobs(context.Background(), 24*time.Hour)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, count, "Should delete exactly two old jobs")

		// Verify that old jobs were deleted
		_, err = repo.Get(context.Background(), oldCompletedJob.ID)
		assert.Error(t, err, "Old completed job should be deleted")
		assert.IsType(t, domain.NotFoundError{}, err)

		_, err = repo.Get(context.Background(), oldFailedJob.ID)
		assert.Error(t, err, "Old failed job should be deleted")
		assert.IsType(t, domain.NotFoundError{}, err)

		// Verify that recent and pending jobs still exist
		stillExists, err := repo.Get(context.Background(), recentCompletedJob.ID)
		assert.NoError(t, err, "Recent completed job should still exist")
		assert.Equal(t, recentCompletedJob.ID, stillExists.ID)

		stillExists, err = repo.Get(context.Background(), pendingJob.ID)
		assert.NoError(t, err, "Pending job should still exist")
		assert.Equal(t, pendingJob.ID, stillExists.ID)
	})

	t.Run("GetLastJobForService", func(t *testing.T) {
		t.Run("success - returns most recent job", func(t *testing.T) {
			// Create a fresh service for this test
			testService := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID, provider.ID, consumer.ID)
			require.NoError(t, serviceRepo.Create(context.Background(), testService))

			// Create multiple jobs sequentially (GORM will set CreatedAt automatically)
			firstJob := domain.NewJob(testService, "create", nil, 1)
			require.NoError(t, repo.Create(context.Background(), firstJob))

			// Small delay to ensure different timestamps
			time.Sleep(10 * time.Millisecond)

			secondJob := domain.NewJob(testService, "start", nil, 2)
			require.NoError(t, repo.Create(context.Background(), secondJob))

			// Get last job
			lastJob, err := repo.GetLastJobForService(context.Background(), testService.ID)
			require.NoError(t, err)
			assert.NotNil(t, lastJob)

			// Should return the most recent job (secondJob)
			assert.Equal(t, secondJob.ID, lastJob.ID)
			assert.Equal(t, testService.ID, lastJob.ServiceID)

			// Verify relationships are loaded
			assert.NotNil(t, lastJob.Agent)
			assert.NotNil(t, lastJob.Service)
		})

		t.Run("success - returns job regardless of status", func(t *testing.T) {
			// Create a fresh service for this test
			testService := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID, provider.ID, consumer.ID)
			require.NoError(t, serviceRepo.Create(context.Background(), testService))

			// Create jobs with different statuses
			pendingJob := domain.NewJob(testService, "create", nil, 1)
			pendingJob.Status = domain.JobPending

			completedJob := domain.NewJob(testService, "start", nil, 2)
			completedJob.Status = domain.JobCompleted

			require.NoError(t, repo.Create(context.Background(), pendingJob))
			require.NoError(t, repo.Create(context.Background(), completedJob))

			// Get last job - should return most recent regardless of status
			lastJob, err := repo.GetLastJobForService(context.Background(), testService.ID)
			require.NoError(t, err)
			assert.NotNil(t, lastJob)
			assert.Equal(t, testService.ID, lastJob.ServiceID)
		})

		t.Run("success - returns nil for non-existent service", func(t *testing.T) {
			nonExistentServiceID := properties.NewUUID()

			lastJob, err := repo.GetLastJobForService(context.Background(), nonExistentServiceID)
			require.NoError(t, err)
			assert.Nil(t, lastJob, "Should return nil for non-existent service")
		})

		t.Run("success - returns single job when only one exists", func(t *testing.T) {
			// Create a fresh service for this test
			testService := createTestService(t, serviceType.ID, serviceGroup.ID, agent.ID, provider.ID, consumer.ID)
			require.NoError(t, serviceRepo.Create(context.Background(), testService))

			// Create a single job
			singleJob := domain.NewJob(testService, "create", nil, 1)
			require.NoError(t, repo.Create(context.Background(), singleJob))

			// Get last job
			lastJob, err := repo.GetLastJobForService(context.Background(), testService.ID)
			require.NoError(t, err)
			assert.NotNil(t, lastJob)
			assert.Equal(t, singleJob.ID, lastJob.ID)
			assert.Equal(t, testService.ID, lastJob.ServiceID)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns correct auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a new job with known IDs
			job := domain.NewJob(service, "create", nil, 1)

			// The job should have provider, agent, and consumer IDs from the service
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

			// Check that the returned scope is a auth.DefaultObjectScope
			defaultScope, ok := scope.(*auth.DefaultObjectScope)
			require.True(t, ok, "AuthScope should return a auth.DefaultObjectScope")

			// Verify auth scope contains the correct IDs
			assert.NotNil(t, defaultScope.ProviderID, "ProviderID should not be nil")
			assert.Equal(t, service.ProviderID, *defaultScope.ProviderID, "Should return the correct provider ID")
			assert.NotNil(t, defaultScope.ConsumerID, "ConsumerID should not be nil")
			assert.Equal(t, service.ConsumerID, *defaultScope.ConsumerID, "Should return the correct consumer ID")
			assert.NotNil(t, defaultScope.AgentID, "AgentID should not be nil")
			assert.Equal(t, service.AgentID, *defaultScope.AgentID, "Should return the correct agent ID")
		})
	})
}
