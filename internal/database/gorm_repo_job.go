package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormJobRepository struct {
	*GormRepository[domain.Job]
}

var applyJobFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"action":    parserInFilterFieldApplier("action", domain.ParseServiceAction),
	"state":     parserInFilterFieldApplier("state", domain.ParseJobState),
	"agentId":   parserInFilterFieldApplier("agent_id", domain.ParseUUID),
	"serviceId": parserInFilterFieldApplier("service_id", domain.ParseUUID),
})

var applyJobSort = mapSortApplier(map[string]string{
	"priority":    "priority",
	"createdAt":   "created_at",
	"claimedAt":   "claimed_at",
	"completedAt": "completed_at",
})

// NewJobRepository creates a new instance of JobRepository
func NewJobRepository(db *gorm.DB) domain.JobRepository {
	repo := &gormJobRepository{
		GormRepository: NewGormRepository[domain.Job](
			db,
			applyJobFilter,
			applyJobSort,
			[]string{"Agent", "Service"}, // Find preload paths
			[]string{"Agent", "Service"}, // List preload paths - empty for performance reasons
		),
	}
	return repo
}

// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
func (r *gormJobRepository) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
	var jobs []*domain.Job
	err := r.db.WithContext(ctx).
		Preload("Service").
		Where("agent_id = ? AND state = ?", agentID, domain.JobPending).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// ClaimJob marks a job as being processed by an agent
func (r *gormJobRepository) ClaimJob(ctx context.Context, jobID domain.UUID, agentID domain.UUID) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&domain.Job{}).
		Where("id = ? AND agent_id = ? AND state = ?", jobID, agentID, domain.JobPending).
		Updates(map[string]interface{}{
			"state":      domain.JobProcessing,
			"claimed_at": now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: fmt.Errorf("job not found or already claimed")}
	}

	return nil
}

// CompleteJob marks a job as completed with result data and service external ID
func (r *gormJobRepository) CompleteJob(ctx context.Context, jobID domain.UUID, resultData domain.JSON, externalID string) error {
	// Validate required externalID
	if externalID == "" {
		return fmt.Errorf("external ID is required")
	}

	// First, find the job to get its type and serviceID
	var job domain.Job
	if err := r.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return err
	}

	// Check if job is in processing state
	if job.State != domain.JobProcessing {
		return domain.NotFoundError{Err: fmt.Errorf("job not found or not in processing state")}
	}

	// Update the job first
	now := time.Now()
	jobResult := r.db.WithContext(ctx).
		Model(&domain.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"state":        domain.JobCompleted,
			"result_data":  resultData,
			"completed_at": now,
		})

	if jobResult.Error != nil {
		return jobResult.Error
	}

	if jobResult.RowsAffected == 0 {
		return fmt.Errorf("failed to update job")
	}

	// If this is a service create job, update the service with external ID
	if job.Action == domain.ServiceActionCreate {
		// Find the service
		var service domain.Service
		if err := r.db.WithContext(ctx).First(&service, "id = ?", job.ServiceID).Error; err != nil {
			return err
		}

		// Update service with external ID and change state
		serviceResult := r.db.WithContext(ctx).
			Model(&domain.Service{}).
			Where("id = ?", job.ServiceID).
			Updates(map[string]interface{}{
				"external_id": externalID,
				"state":       domain.ServiceStarted,
			})

		if serviceResult.Error != nil {
			return serviceResult.Error
		}

		if serviceResult.RowsAffected == 0 {
			return fmt.Errorf("failed to update service with external ID")
		}
	}

	return nil
}

// FailJob marks a job as failed with an error message
func (r *gormJobRepository) FailJob(ctx context.Context, jobID domain.UUID, errorMessage string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&domain.Job{}).
		Where("id = ? AND state = ?", jobID, domain.JobProcessing).
		Updates(map[string]interface{}{
			"state":         domain.JobFailed,
			"error_message": errorMessage,
			"completed_at":  now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: fmt.Errorf("job not found or not in processing state")}
	}

	return nil
}

// ReleaseStuckJobs resets jobs that have been processing for too long
func (r *gormJobRepository) ReleaseStuckJobs(ctx context.Context, olderThanMinutes int) (int, error) {
	cutoffTime := time.Now().Add(-time.Duration(olderThanMinutes) * time.Minute)
	// Using Exec for direct SQL execution with NULL
	result := r.db.WithContext(ctx).Exec(
		"UPDATE jobs SET state = ?, claimed_at = NULL WHERE state = ? AND claimed_at < ?",
		domain.JobPending, domain.JobProcessing, cutoffTime,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// DeleteOldCompletedJobs removes completed or failed jobs older than the specified days
func (r *gormJobRepository) DeleteOldCompletedJobs(ctx context.Context, olderThanDays int) (int, error) {
	cutoffTime := time.Now().AddDate(0, 0, -olderThanDays)
	// Using Exec for direct SQL execution
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM jobs WHERE (state = ? OR state = ?) AND completed_at < ?",
		domain.JobCompleted, domain.JobFailed, cutoffTime,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}
