package domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// ServiceState represents the possible states of a service
type ServiceState string

const (
	ServiceCreating     ServiceState = "Creating"
	ServiceCreated      ServiceState = "Created"
	ServiceStarting     ServiceState = "Starting"
	ServiceStarted      ServiceState = "Started"
	ServiceStopping     ServiceState = "Stopping"
	ServiceStopped      ServiceState = "Stopped"
	ServiceHotUpdating  ServiceState = "HotUpdating"
	ServiceColdUpdating ServiceState = "ColdUpdating"
	ServiceDeleting     ServiceState = "Deleting"
	ServiceDeleted      ServiceState = "Deleted"
)

// ParseServiceState parses a string into a ServiceState
func ParseServiceState(s string) (ServiceState, error) {
	state := ServiceState(s)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Validate checks if the service state is valid
func (s ServiceState) Validate() error {
	switch s {
	case
		ServiceCreating,
		ServiceCreated,
		ServiceStarting,
		ServiceStarted,
		ServiceStopping,
		ServiceStopped,
		ServiceHotUpdating,
		ServiceColdUpdating,
		ServiceDeleting,
		ServiceDeleted:
		return nil
	default:
		return fmt.Errorf("invalid service state: %s", s)
	}
}

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity

	Name       string     `gorm:"not null"`
	Attributes Attributes `gorm:"type:jsonb"`

	// State management
	CurrentState      ServiceState `gorm:"not null"`
	TargetState       *ServiceState
	ErrorMessage      *string
	FailedAction      *ServiceAction
	RetryCount        int
	CurrentProperties *JSON `gorm:"type:jsonb"`
	TargetProperties  *JSON `gorm:"type:jsonb"`

	// To store an external ID for the agent's use to facilitate metric reporting
	ExternalID *string `gorm:"uniqueIndex:service_external_id_uniq"`
	// Safe place for the Agent for store data
	Resources *JSON `gorm:"type:jsonb"`

	// Relationships
	GroupID       UUID          `gorm:"not null" json:"groupId"`
	Group         *ServiceGroup `gorm:"foreignKey:GroupID"`
	AgentID       UUID          `gorm:"not null"`
	Agent         *Agent        `gorm:"foreignKey:AgentID"`
	ServiceTypeID UUID          `gorm:"not null"`
	ServiceType   *ServiceType  `gorm:"foreignKey:ServiceTypeID"`
}

// Validate a service
func (s Service) Validate() error {
	if s.Name == "" {
		return errors.New("service name cannot be empty")
	}
	if err := s.CurrentState.Validate(); err != nil {
		return err
	}
	if s.TargetState != nil {
		if err := s.TargetState.Validate(); err != nil {
			return err
		}
	}
	if s.GroupID == uuid.Nil {
		return errors.New("service group ID cannot be nil")
	}
	if s.AgentID == uuid.Nil {
		return errors.New("service agent ID cannot be nil")
	}
	if s.ServiceTypeID == uuid.Nil {
		return errors.New("service type ID cannot be nil")
	}
	if s.Attributes != nil {
		if err := s.Attributes.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// TableName returns the table name for the service
func (Service) TableName() string {
	return "services"
}

// NewService creates a new service with the right stuff
func NewService(agentID UUID,
	serviceTypeID UUID,
	groupID UUID,
	name string,
	attributes Attributes,
	properties JSON,
) *Service {
	target := ServiceCreated
	return &Service{
		GroupID:           groupID,
		AgentID:           agentID,
		ServiceTypeID:     serviceTypeID,
		Name:              name,
		CurrentState:      ServiceCreating,
		TargetState:       &target,
		Attributes:        attributes,
		CurrentProperties: nil,
		TargetProperties:  &properties,
	}
}

// ServiceCommander defines the interface for service command operations
type ServiceCommander interface {
	// Create handles service creation and creates a job for the agent
	Create(ctx context.Context, agentID UUID, serviceTypeID UUID, groupID UUID, name string, attributes Attributes, properties JSON) (*Service, error)

	// Update handles service updates and creates a job for the agent
	Update(ctx context.Context, id UUID, name *string, props *JSON) (*Service, error)

	// Transition transitions a service to a new state
	Transition(ctx context.Context, id UUID, target ServiceState) (*Service, error)

	// Retry retries a failed service operation
	Retry(ctx context.Context, id UUID) (*Service, error)

	// FailTimeoutServicesAndJobs fails services and jobs that have timed out
	FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error)
}

// serviceCommander is the concrete implementation of ServiceCommander
type serviceCommander struct {
	store Store
}

// NewServiceCommander creates a new commander for services
func NewServiceCommander(
	store Store,
) *serviceCommander {
	return &serviceCommander{
		// Using store interface to access repositories
		store: store,
	}
}

func (s *serviceCommander) Create(
	ctx context.Context,
	agentID UUID,
	serviceTypeID UUID,
	groupID UUID,
	name string,
	attributes Attributes,
	properties JSON,
) (*Service, error) {
	if err := s.validateServiceAuthScope(ctx, agentID, groupID); err != nil {
		return nil, UnauthorizedError{Err: err}
	}

	svc := NewService(agentID, serviceTypeID, groupID, name, attributes, properties)
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err := s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Create(ctx, svc); err != nil {
			return err
		}
		job := NewJob(svc.AgentID, svc.ID, ServiceActionCreate, 1)
		if err := job.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *serviceCommander) Update(ctx context.Context, id UUID, name *string, props *JSON) (*Service, error) {
	svc, err := s.store.ServiceRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.validateServiceAuthScope(ctx, svc.AgentID, svc.GroupID); err != nil {
		return nil, UnauthorizedError{Err: err}
	}

	updateSvc := false
	var job *Job

	// Name
	if name != nil && svc.Name != *name {
		updateSvc = true
		svc.Name = *name
	}

	// Properties
	if props != nil && !reflect.DeepEqual(&svc.CurrentProperties, *props) {
		updateSvc = true
		trans, action, err := serviceUpdateNextStateAndAction(svc.CurrentState)
		if err != nil {
			return nil, InvalidInputError{Err: err}
		}
		svc.TargetProperties = props
		target := svc.CurrentState
		svc.TargetState = &target
		svc.CurrentState = trans
		job = NewJob(svc.AgentID, svc.ID, action, 1)
		if err := job.Validate(); err != nil {
			return nil, InvalidInputError{Err: err}
		}
	}

	// Update Service
	if !updateSvc {
		return svc, nil
	}

	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if job == nil {
		// No job needed, just save the service
		if err := s.store.ServiceRepo().Save(ctx, svc); err != nil {
			return nil, err
		}
		return svc, nil
	}

	// Execute within a transaction when job is created
	var updatedService *Service = svc
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		return store.JobRepo().Create(ctx, job)
	})
	if err != nil {
		return nil, err
	}
	return updatedService, nil
}

// serviceUpdateNextStateAndAction determines the transition state and action when updating a service's properties based on its current state
func serviceUpdateNextStateAndAction(state ServiceState) (ServiceState, ServiceAction, error) {
	switch state {
	case ServiceStopped:
		return ServiceColdUpdating, ServiceActionColdUpdate, nil
	case ServiceStarted:
		return ServiceHotUpdating, ServiceActionHotUpdate, nil
	default:
		return "", "", NewInvalidInputErrorf("cannot update attributes on a service with state %v", state)
	}
}

func (s *serviceCommander) Transition(ctx context.Context, id UUID, target ServiceState) (*Service, error) {
	if err := target.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	svc, err := s.store.ServiceRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.validateServiceAuthScope(ctx, svc.AgentID, svc.GroupID); err != nil {
		return nil, UnauthorizedError{Err: err}
	}

	trans, action, err := serviceNextStateAndAction(svc.CurrentState, target)
	if err != nil {
		return nil, InvalidInputError{Err: err}
	}
	svc.CurrentState = trans
	svc.TargetState = &target
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Create job for transition
	job := NewJob(svc.AgentID, svc.ID, action, 1)
	if err := job.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Execute within a transaction
	var transitionedService *Service = svc
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		return store.JobRepo().Create(ctx, job)
	})
	if err != nil {
		return nil, err
	}
	return transitionedService, nil
}

func (s *serviceCommander) Retry(ctx context.Context, id UUID) (*Service, error) {
	svc, err := s.store.ServiceRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.validateServiceAuthScope(ctx, svc.AgentID, svc.GroupID); err != nil {
		return nil, UnauthorizedError{Err: err}
	}

	if svc.FailedAction == nil {
		// Nothing to retry
		return svc, nil
	}
	svc.RetryCount += 1
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Create job for retry
	job := NewJob(svc.AgentID, svc.ID, *svc.FailedAction, 1)
	if err := job.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Execute within a transaction
	var retryService *Service = svc
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		return store.JobRepo().Create(ctx, job)
	})
	if err != nil {
		return nil, err
	}
	return retryService, nil
}

// serviceNextStateAndAction determines the intermediate state and action for a service transition
func serviceNextStateAndAction(curr, target ServiceState) (ServiceState, ServiceAction, error) {
	switch curr {
	case ServiceCreated:
		if target == ServiceStarted {
			return ServiceStarting, ServiceActionStart, nil
		}
	case ServiceStarted:
		if target == ServiceStopped {
			return ServiceStopping, ServiceActionStop, nil
		}
	case ServiceStopped:
		if target == ServiceStarted {
			return ServiceStarting, ServiceActionStart, nil
		}
		if target == ServiceDeleted {
			return ServiceDeleting, ServiceActionDelete, nil
		}
	}
	return "", "", fmt.Errorf("invalid transition from %s to %s", curr, target)
}

func (s *serviceCommander) validateServiceAuthScope(ctx context.Context, agentID, groupID UUID) error {
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return err
	}
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, groupID)
	if err != nil {
		return err
	}
	return ValidateAuthScope(ctx, &AuthScope{AgentID: &agentID, ProviderID: &agent.ProviderID, BrokerID: &sg.BrokerID})
}

func (s *serviceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	timedOutJobs, err := s.store.JobRepo().GetTimeOutJobs(ctx, timeout)
	if err != nil {
		return 0, fmt.Errorf("failed to retrive timeout jobs: %v", err)
	}

	counter := 0
	errorMsg := "Job marked as failed due to exceeding maximum processing time"
	for _, job := range timedOutJobs {
		err := s.store.Atomic(ctx, func(s Store) error {
			// Update job
			job.State = JobFailed
			job.ErrorMessage = errorMsg
			now := time.Now()
			job.CompletedAt = &now

			// Update associated service status
			svc, err := s.ServiceRepo().FindByID(ctx, job.ServiceID)
			if err != nil {
				return fmt.Errorf("failed to find service %s for timed out job: %w", job.ServiceID, err)
			}

			svc.ErrorMessage = &errorMsg
			svc.FailedAction = &job.Action

			if err := s.JobRepo().Save(ctx, job); err != nil {
				return err
			}
			return s.ServiceRepo().Save(ctx, svc)
		})
		counter++
		if err != nil {
			return 0, fmt.Errorf("Error marking timed out job %s as failed: %v", job.ID, err)
		}
	}

	return counter, nil
}

// ServiceRepository defines the interface for the Service repository
type ServiceRepository interface {
	ServiceQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *Service) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Service) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error
}

// ServiceQuerier defines the interface for the Service read-only queries
type ServiceQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Service, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// FindByExternalID retrieves a service by its external ID and agent ID
	FindByExternalID(ctx context.Context, agentID UUID, externalID string) (*Service, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Service], error)

	// CountByGroup returns the number of services in a specific group
	CountByGroup(ctx context.Context, groupID UUID) (int64, error)

	// CountByAgent returns the number of services handled by a specific agent
	CountByAgent(ctx context.Context, agentID UUID) (int64, error)
}
