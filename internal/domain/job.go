package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ServiceAction represents the type of operation a job performs
type ServiceAction string

const (
	ServiceActionCreate     ServiceAction = "ServiceCreate"
	ServiceActionStart      ServiceAction = "ServiceStart"
	ServiceActionStop       ServiceAction = "ServiceStop"
	ServiceActionHotUpdate  ServiceAction = "ServiceHotUpdate"
	ServiceActionColdUpdate ServiceAction = "ServiceColdUpdate"
	ServiceActionDelete     ServiceAction = "ServiceDelete"
)

// ParseServiceAction parses a string into a JobType
func ParseServiceAction(s string) (ServiceAction, error) {
	jobType := ServiceAction(s)
	if err := jobType.Validate(); err != nil {
		return "", err
	}
	return jobType, nil
}

// Validate checks if the job type is valid
func (t ServiceAction) Validate() error {
	switch t {
	case
		ServiceActionCreate,
		ServiceActionStart,
		ServiceActionStop,
		ServiceActionHotUpdate,
		ServiceActionColdUpdate,
		ServiceActionDelete:
		return nil
	}
	return fmt.Errorf("invalid job type: %s", t)
}

// JobState represents the current state of a job
type JobState string

const (
	JobPending    JobState = "Pending"
	JobProcessing JobState = "Processing"
	JobCompleted  JobState = "Completed"
	JobFailed     JobState = "Failed"
)

// Validate checks if the service state is valid
func (s JobState) Validate() error {
	switch s {
	case
		JobPending,
		JobProcessing,
		JobCompleted,
		JobFailed:
		return nil
	default:
		return fmt.Errorf("invalid job state: %s", s)
	}
}

// ParseJobState parses a string into a JobState
func ParseJobState(s string) (JobState, error) {
	state := JobState(s)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Job represents a task to be executed by an agent
type Job struct {
	BaseEntity
	// Immutables
	AgentID   UUID          `gorm:"type:uuid;not null"`
	ServiceID UUID          `gorm:"type:uuid;not null"`
	State     JobState      `gorm:"type:varchar(20);not null"`
	Action    ServiceAction `gorm:"type:varchar(50);not null"`
	Priority  int           `gorm:"not null;default:1"`
	// For state management
	ErrorMessage string     `gorm:"type:text"`
	ClaimedAt    *time.Time `gorm:""`
	CompletedAt  *time.Time `gorm:""`

	// Relationships
	Agent   *Agent   `gorm:"foreignKey:AgentID"`
	Service *Service `gorm:"foreignKey:ServiceID"`
}

// TableName returns the table name for the job
func (*Job) TableName() string {
	return "jobs"
}

// Validate ensures all Job fields are valid
func (j *Job) Validate() error {
	if err := j.Action.Validate(); err != nil {
		return fmt.Errorf("invalid action: %w", err)
	}
	if err := j.State.Validate(); err != nil {
		return fmt.Errorf("invalid state: %w", err)
	}
	if j.Priority < 1 {
		return errors.New("priority must be greater than 0")
	}
	return nil
}

// NewJob creates a new job instance with the provided parameters
func NewJob(agentID, serviceID UUID, action ServiceAction, priority int) *Job {
	return &Job{
		AgentID:   agentID,
		ServiceID: serviceID,
		State:     JobPending,
		Action:    action,
		Priority:  priority,
	}
}

// JobCommander handles job operations
type JobCommander struct {
	repo        JobRepository
	serviceRepo ServiceRepository
}

// NewJobCommander creates a new command executor
func NewJobCommander(
	jobRepo JobRepository,
	serviceRepo ServiceRepository,
) *JobCommander {
	return &JobCommander{
		repo:        jobRepo,
		serviceRepo: serviceRepo,
	}
}

// Claim claims a job for an agent
func (s *JobCommander) Claim(ctx context.Context, agentID UUID, jobID UUID) error {
	job, err := s.repo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.AgentID != agentID {
		return errors.New("cannot claim a job not assigned to the authenticated agent")
	}
	if job.State != JobPending {
		return errors.New("cannot claim a job not in pending state")
	}
	job.State = JobProcessing
	now := time.Now()
	job.ClaimedAt = &now
	err = s.repo.Save(ctx, job)
	if err != nil {
		return err
	}
	return nil
}

// Complete marks a job as completed
func (s *JobCommander) Complete(ctx context.Context, agentID, jobID UUID, resources *JSON, externalID *string) error {
	job, err := s.repo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.AgentID != agentID {
		return errors.New("cannot complete a job not assigned to the authenticated agent")
	}
	if job.State != JobProcessing {
		return errors.New("cannot complete a job not in processing state")
	}
	// Update Job
	job.State = JobCompleted
	now := time.Now()
	job.CompletedAt = &now
	err = s.repo.Save(ctx, job)
	if err != nil {
		return err
	}
	// Update Service
	svc, err := s.serviceRepo.FindByID(ctx, job.ServiceID)
	if err != nil {
		return err
	}
	if svc.TargetState == nil {
		return errors.New("cannot complete a job on service that is not in transition")
	}
	svc.CurrentState = *svc.TargetState
	svc.TargetState = nil
	svc.FailedAction = nil
	svc.ErrorMessage = nil
	svc.RetryCount = 0
	if resources != nil {
		svc.Resources = resources
	}
	if externalID != nil {
		svc.ExternalID = externalID
	}
	if svc.TargetProperties != nil {
		svc.CurrentProperties = svc.TargetProperties
		svc.TargetProperties = nil
	}
	if err := svc.Validate(); err != nil {
		return err
	}
	if err := s.serviceRepo.Save(ctx, svc); err != nil {
		return err
	}
	return nil
}

// Fail marks a job as failed
func (s *JobCommander) Fail(ctx context.Context, agentID UUID, jobID UUID, errorMessage string) error {
	job, err := s.repo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.AgentID != agentID {
		return errors.New("cannot fail a job not assigned to the authenticated agent")
	}
	if job.State != JobProcessing {
		return errors.New("cannot fail a job not in processing state")
	}
	// Update Job
	job.State = JobFailed
	job.ErrorMessage = errorMessage
	err = s.repo.Save(ctx, job)
	if err != nil {
		return err
	}
	// Update Service
	svc, err := s.serviceRepo.FindByID(ctx, job.ServiceID)
	if err != nil {
		return err
	}
	svc.ErrorMessage = &errorMessage
	svc.FailedAction = &job.Action
	if err := svc.Validate(); err != nil {
		return err
	}
	if err := s.serviceRepo.Save(ctx, svc); err != nil {
		return err
	}
	return nil
}

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

	// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
	GetPendingJobsForAgent(ctx context.Context, agentID UUID, limit int) ([]*Job, error)

	// Maintenance operations

	// ReleaseStuckJobs resets jobs that have been processing for too long
	ReleaseStuckJobs(ctx context.Context, olderThan time.Duration) (int, error)

	// DeleteOldCompletedJobs removes completed or failed jobs older than the specified interval
	DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error)
}

type JobQuerier interface {
	// FindByID retrieves a job by ID
	FindByID(ctx context.Context, id UUID) (*Job, error)

	// List retrieves a list of jobs based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Job], error)

	// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
	GetPendingJobsForAgent(ctx context.Context, agentID UUID, limit int) ([]*Job, error)
}
