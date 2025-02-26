package domain

import "context"

// JobRepository defines the interface for the Job repository
type JobRepository interface {
	// Create creates a new job
	Create(ctx context.Context, job *Job) error

	// Save updates an existing job
	Save(ctx context.Context, job *Job) error

	// Delete removes a job by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves a job by ID
	FindByID(ctx context.Context, id UUID) (*Job, error)

	// List retrieves a list of jobs based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Job], error)

	// Queue specific operations

	// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
	GetPendingJobsForAgent(ctx context.Context, agentID UUID, limit int) ([]*Job, error)

	// ClaimJob marks a job as being processed by an agent
	ClaimJob(ctx context.Context, jobID UUID, agentID UUID) error

	// CompleteJob marks a job as completed with result data
	CompleteJob(ctx context.Context, jobID UUID, resultData JSON) error

	// FailJob marks a job as failed with an error message
	FailJob(ctx context.Context, jobID UUID, errorMessage string) error

	// Maintenance operations

	// ReleaseStuckJobs resets jobs that have been processing for too long
	ReleaseStuckJobs(ctx context.Context, olderThanMinutes int) (int, error)

	// DeleteOldCompletedJobs removes completed or failed jobs older than the specified days
	DeleteOldCompletedJobs(ctx context.Context, olderThanDays int) (int, error)
}
