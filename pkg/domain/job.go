package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

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

	Action   string           `gorm:"type:varchar(50);not null"`
	Params   *properties.JSON `gorm:"type:jsonb"`
	Priority int              `gorm:"not null;default:1"`

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
	if j.Action == "" {
		return fmt.Errorf("action cannot be empty")
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
func NewJob(svc *Service, action string, params *properties.JSON, priority int) *Job {
	return &Job{
		ConsumerID: svc.ConsumerID,
		ProviderID: svc.ProviderID,
		AgentID:    svc.AgentID,
		ServiceID:  svc.ID,
		Status:     JobPending,
		Action:     action,
		Params:     params,
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

// IsActive checks if the job is active (blocks new job attempts for the same service)
func (j *Job) IsActive() bool {
	return j.Status == JobProcessing || j.Status == JobPending
}

// JobCommander defines the interface for job command operations
type JobCommander interface {
	// Claim claims a job for an agent
	Claim(ctx context.Context, jobID properties.UUID) error

	// Complete marks a job as completed
	Complete(ctx context.Context, params CompleteJobParams) error

	// Fail marks a job as failed
	Fail(ctx context.Context, params FailJobParams) error
}

type CompleteJobParams struct {
	JobID             properties.UUID  `json:"jobId"`
	AgentInstanceData *properties.JSON `json:"agentInstanceData"`
	AgentInstanceID   *string          `json:"agentInstanceId"`
	Properties        map[string]any   `json:"properties,omitempty"`
}

type FailJobParams struct {
	JobID        properties.UUID `json:"jobId"`
	ErrorMessage string          `json:"errorMessage"`
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

func (s *jobCommander) Complete(ctx context.Context, params CompleteJobParams) error {
	job, err := s.store.JobRepo().Get(ctx, params.JobID)
	if err != nil {
		return err
	}
	svc, err := s.store.ServiceRepo().Get(ctx, job.ServiceID)
	if err != nil {
		return err
	}
	originalSvc := *svc

	// Load ServiceType for property validation
	serviceType, err := s.store.ServiceTypeRepo().Get(ctx, svc.ServiceTypeID)
	if err != nil {
		return err
	}

	return s.store.Atomic(ctx, func(store Store) error {
		// Validate lifecycle schema exists
		if serviceType.LifecycleSchema == nil {
			return NewInvalidInputErrorf("service type %s does not have a lifecycle schema", serviceType.Name)
		}

		// Update job
		if err := job.Complete(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.JobRepo().Save(ctx, job); err != nil {
			return err
		}

		// Apply agent property updates if provided
		if len(params.Properties) > 0 {
			if err := svc.ApplyAgentPropertyUpdates(serviceType, params.Properties); err != nil {
				return InvalidInputError{Err: err}
			}
		}

		// Update service
		if err := svc.HandleJobComplete(serviceType.LifecycleSchema, job.Action, nil, job.Params, params.AgentInstanceData, params.AgentInstanceID); err != nil {
			return InvalidInputError{Err: err}
		}

		// Clear agent instance ID if service reached a terminal state to allow infrastructure ID reuse (e.g., Proxmox VM IDs)
		if serviceType.LifecycleSchema.IsTerminalState(svc.Status) {
			svc.AgentInstanceID = nil
		}

		if err := svc.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}

		// Release pool allocations if service reached a terminal state
		if serviceType.LifecycleSchema.IsTerminalState(svc.Status) {
			if err := ReleaseServicePoolAllocations(ctx, store, svc.ID); err != nil {
				return fmt.Errorf("failed to release pool allocations: %w", err)
			}
		}

		// Create event for the updated service
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

func (s *jobCommander) Fail(ctx context.Context, params FailJobParams) error {
	job, err := s.store.JobRepo().Get(ctx, params.JobID)
	if err != nil {
		return err
	}
	svc, err := s.store.ServiceRepo().Get(ctx, job.ServiceID)
	if err != nil {
		return err
	}
	originalSvc := *svc

	// Load ServiceType to get lifecycle schema
	serviceType, err := s.store.ServiceTypeRepo().Get(ctx, svc.ServiceTypeID)
	if err != nil {
		return err
	}

	return s.store.Atomic(ctx, func(store Store) error {
		// Validate lifecycle schema exists
		if serviceType.LifecycleSchema == nil {
			return NewInvalidInputErrorf("service type %s does not have a lifecycle schema", serviceType.Name)
		}

		// Update job
		if err := job.Fail(params.ErrorMessage); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.JobRepo().Save(ctx, job); err != nil {
			return err
		}

		// Update service state using error message for transition logic (regexp matching)
		errorCode := &params.ErrorMessage
		if err := svc.HandleJobComplete(serviceType.LifecycleSchema, job.Action, errorCode, job.Params, nil, nil); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := svc.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}

		// Create event for the updated service
		eventEntry, err := NewEvent(EventTypeServiceTransitioned, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return nil
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

	// GetLastJobForService retrieves the last job for a specific service
	GetLastJobForService(ctx context.Context, serviceID properties.UUID) (*Job, error)

	// GetTimeOutJobs retrieves jobs that have been processing for too long and returns them
	GetTimeOutJobs(ctx context.Context, olderThan time.Duration) ([]*Job, error)
}
