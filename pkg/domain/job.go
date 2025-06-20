package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
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

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobPending    JobStatus = "Pending"
	JobProcessing JobStatus = "Processing"
	JobCompleted  JobStatus = "Completed"
	JobFailed     JobStatus = "Failed"
)

// Validate checks if the service status is valid
func (s JobStatus) Validate() error {
	switch s {
	case
		JobPending,
		JobProcessing,
		JobCompleted,
		JobFailed:
		return nil
	default:
		return fmt.Errorf("invalid job status: %s", s)
	}
}

// ParseJobStatus parses a string into a JobStatus
func ParseJobStatus(s string) (JobStatus, error) {
	status := JobStatus(s)
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

// Job represents a task to be executed by an agent
type Job struct {
	BaseEntity

	Action   ServiceAction `gorm:"type:varchar(50);not null"`
	Priority int           `gorm:"not null;default:1"`

	// Status management
	Status       JobStatus  `gorm:"type:varchar(20);not null"`
	ErrorMessage string     `gorm:"type:text"`
	ClaimedAt    *time.Time `gorm:""`
	CompletedAt  *time.Time `gorm:""`

	// Relationships
	AgentID    properties.UUID `gorm:"not null"`
	Agent      *Agent          `gorm:"foreignKey:AgentID"`
	ServiceID  properties.UUID `gorm:"not null"`
	Service    *Service        `gorm:"foreignKey:ServiceID"`
	ProviderID properties.UUID `gorm:"not null"`
	Provider   *Participant    `gorm:"foreignKey:ProviderID"`
	ConsumerID properties.UUID `gorm:"not null"`
	Consumer   *Participant    `gorm:"foreignKey:ConsumerID"`
}

// TableName returns the table name for the job
func (Job) TableName() string {
	return "jobs"
}

// Validate ensures all Job fields are valid
func (j *Job) Validate() error {
	if err := j.Action.Validate(); err != nil {
		return fmt.Errorf("invalid action: %w", err)
	}
	if err := j.Status.Validate(); err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}
	if j.Priority < 1 {
		return errors.New("priority must be greater than 0")
	}
	if j.AgentID == uuid.Nil {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if j.ServiceID == uuid.Nil {
		return fmt.Errorf("service ID cannot be empty")
	}
	return nil
}

// NewJob creates a new job instance with the provided parameters
func NewJob(svc *Service, action ServiceAction, priority int) *Job {
	return &Job{
		ConsumerID: svc.ConsumerID,
		ProviderID: svc.ProviderID,
		AgentID:    svc.AgentID,
		ServiceID:  svc.ID,
		Status:     JobPending,
		Action:     action,
		Priority:   priority,
	}
}

// Claim marks a job as claimed by an agent
func (j *Job) Claim() error {
	if j.Status != JobPending {
		return fmt.Errorf("cannot claim a job not in pending status")
	}
	j.Status = JobProcessing
	now := time.Now()
	j.ClaimedAt = &now
	return nil
}

// Complete marks a job as successfully completed
func (j *Job) Complete() error {
	if j.Status != JobProcessing {
		return fmt.Errorf("cannot complete a job not in processing status")
	}
	j.Status = JobCompleted
	now := time.Now()
	j.CompletedAt = &now
	return nil
}

// Fail records job failure with error details
func (j *Job) Fail(errorMessage string) error {
	if j.Status != JobProcessing {
		return fmt.Errorf("cannot fail a job not in processing status")
	}
	j.Status = JobFailed
	j.ErrorMessage = errorMessage
	return nil
}

// Retry increments retry count and updates status
func (j *Job) Retry() error {
	if j.Status != JobFailed {
		return fmt.Errorf("cannot retry a job not in failed status")
	}
	j.Status = JobPending
	j.ErrorMessage = ""
	return nil
}

// JobCommander defines the interface for job command operations
type JobCommander interface {
	// Claim claims a job for an agent
	Claim(ctx context.Context, jobID properties.UUID) error

	// Complete marks a job as completed
	Complete(ctx context.Context, jobID properties.UUID, resources *properties.JSON, externalID *string) error

	// Fail marks a job as failed
	Fail(ctx context.Context, jobID properties.UUID, errorMessage string) error
}

// jobCommander is the concrete implementation of JobCommander
type jobCommander struct {
	store Store
}

// NewJobCommander creates a new command executor
func NewJobCommander(
	store Store,
) *jobCommander {
	return &jobCommander{
		store: store,
	}
}

func (s *jobCommander) Claim(ctx context.Context, jobID properties.UUID) error {
	job, err := s.store.JobRepo().Get(ctx, jobID)
	if err != nil {
		return err
	}

	if err := job.Claim(); err != nil {
		return InvalidInputError{Err: err}
	}

	return s.store.JobRepo().Save(ctx, job)
}

func (s *jobCommander) Complete(ctx context.Context, jobID properties.UUID, resources *properties.JSON, externalID *string) error {
	job, err := s.store.JobRepo().Get(ctx, jobID)
	if err != nil {
		return err
	}

	svc, err := s.store.ServiceRepo().Get(ctx, job.ServiceID)
	if err != nil {
		return err
	}

	if svc.TargetStatus == nil {
		return InvalidInputError{Err: errors.New("cannot complete a job on service that is not in transition")}
	}

	originalSvc := *svc

	return s.store.Atomic(ctx, func(store Store) error {
		if err := job.Complete(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.JobRepo().Save(ctx, job); err != nil {
			return err
		}

		// Coordinate with service
		if err := svc.HandleJobComplete(resources, externalID); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := svc.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeServiceTransitioned, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
}

func (s *jobCommander) Fail(ctx context.Context, jobID properties.UUID, errorMessage string) error {
	job, err := s.store.JobRepo().Get(ctx, jobID)
	if err != nil {
		return err
	}

	svc, err := s.store.ServiceRepo().Get(ctx, job.ServiceID)
	if err != nil {
		return err
	}

	originalSvc := *svc

	return s.store.Atomic(ctx, func(store Store) error {
		if err := job.Fail(errorMessage); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.JobRepo().Save(ctx, job); err != nil {
			return err
		}

		// Coordinate with service
		svc.HandleJobFailure(errorMessage, job.Action)
		if err := svc.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeServiceTransitioned, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
}

type JobRepository interface {
	JobQuerier
	BaseEntityRepository[Job]

	// DeleteOldCompletedJobs removes completed or failed jobs older than the specified interval
	DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error)
}

type JobQuerier interface {
	BaseEntityQuerier[Job]

	// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
	GetPendingJobsForAgent(ctx context.Context, agentID properties.UUID, limit int) ([]*Job, error)

	// GetTimeOutJobs retrieves jobs that have been processing for too long and returns them
	GetTimeOutJobs(ctx context.Context, olderThan time.Duration) ([]*Job, error)
}
