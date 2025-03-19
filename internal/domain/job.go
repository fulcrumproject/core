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

	Action   ServiceAction `gorm:"type:varchar(50);not null"`
	Priority int           `gorm:"not null;default:1"`

	// State management
	State        JobState   `gorm:"type:varchar(20);not null"`
	ErrorMessage string     `gorm:"type:text"`
	ClaimedAt    *time.Time `gorm:""`
	CompletedAt  *time.Time `gorm:""`

	// Relationships
	AgentID   UUID     `gorm:"type:uuid;not null"`
	Agent     *Agent   `gorm:"foreignKey:AgentID"`
	ServiceID UUID     `gorm:"type:uuid;not null"`
	Service   *Service `gorm:"foreignKey:ServiceID"`
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

// JobCommander defines the interface for job command operations
type JobCommander interface {
	// Claim claims a job for an agent
	Claim(ctx context.Context, agentID UUID, jobID UUID) error

	// Complete marks a job as completed
	Complete(ctx context.Context, agentID UUID, jobID UUID, resources *JSON, externalID *string) error

	// Fail marks a job as failed
	Fail(ctx context.Context, agentID UUID, jobID UUID, errorMessage string) error
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

func (s *jobCommander) Claim(ctx context.Context, agentID UUID, jobID UUID) error {
	// Get the agent to retrieve its providerID
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	job, err := s.store.JobRepo().FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	// Get the service and service group to get the broker ID
	svc, err := s.store.ServiceRepo().FindByID(ctx, job.ServiceID)
	if err != nil {
		return err
	}

	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, svc.GroupID)
	if err != nil {
		return err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &agentID, ProviderID: &agent.ProviderID, BrokerID: &sg.BrokerID}); err != nil {
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
	return s.store.JobRepo().Save(ctx, job)
}

func (s *jobCommander) Complete(ctx context.Context, agentID, jobID UUID, resources *JSON, externalID *string) error {
	// Get the agent to retrieve its providerID
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	job, err := s.store.JobRepo().FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	// Get the service and service group to get the broker ID
	svc, err := s.store.ServiceRepo().FindByID(ctx, job.ServiceID)
	if err != nil {
		return err
	}

	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, svc.GroupID)
	if err != nil {
		return err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &agentID, ProviderID: &agent.ProviderID, BrokerID: &sg.BrokerID}); err != nil {
		return err
	}

	if job.AgentID != agentID {
		return errors.New("cannot complete a job not assigned to the authenticated agent")
	}
	if job.State != JobProcessing {
		return errors.New("cannot complete a job not in processing state")
	}
	if svc.TargetState == nil {
		return errors.New("cannot complete a job on service that is not in transition")
	}
	return s.store.Atomic(ctx, func(store Store) error {
		job.State = JobCompleted
		now := time.Now()
		job.CompletedAt = &now

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

		if err := store.JobRepo().Save(ctx, job); err != nil {
			return err
		}

		return store.ServiceRepo().Save(ctx, svc)
	})
}

func (s *jobCommander) Fail(ctx context.Context, agentID UUID, jobID UUID, errorMessage string) error {
	// Get the agent to retrieve its providerID
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	job, err := s.store.JobRepo().FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	// Get the service and service group to get the broker ID
	svc, err := s.store.ServiceRepo().FindByID(ctx, job.ServiceID)
	if err != nil {
		return err
	}

	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, svc.GroupID)
	if err != nil {
		return err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &agentID, ProviderID: &agent.ProviderID, BrokerID: &sg.BrokerID}); err != nil {
		return err
	}

	if job.AgentID != agentID {
		return errors.New("cannot fail a job not assigned to the authenticated agent")
	}
	if job.State != JobProcessing {
		return errors.New("cannot fail a job not in processing state")
	}
	return s.store.Atomic(ctx, func(store Store) error {
		job.State = JobFailed
		job.ErrorMessage = errorMessage

		svc.ErrorMessage = &errorMessage
		svc.FailedAction = &job.Action

		if err := svc.Validate(); err != nil {
			return err
		}

		if err := store.JobRepo().Save(ctx, job); err != nil {
			return err
		}

		return store.ServiceRepo().Save(ctx, svc)
	})
}

type JobRepository interface {
	JobQuerier

	// Create creates a new job
	Create(ctx context.Context, job *Job) error

	// Save updates an existing job
	Save(ctx context.Context, job *Job) error

	// Delete removes a job by ID
	Delete(ctx context.Context, id UUID) error

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

	// GetTimeOutJobs retrieves jobs that have been processing for too long and returns them
	GetTimeOutJobs(ctx context.Context, olderThan time.Duration) ([]*Job, error)
}
