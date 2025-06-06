package domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// ServiceStatus represents the possible statuss of a service
type ServiceStatus string

const (
	ServiceCreating     ServiceStatus = "Creating"
	ServiceCreated      ServiceStatus = "Created"
	ServiceStarting     ServiceStatus = "Starting"
	ServiceStarted      ServiceStatus = "Started"
	ServiceStopping     ServiceStatus = "Stopping"
	ServiceStopped      ServiceStatus = "Stopped"
	ServiceHotUpdating  ServiceStatus = "HotUpdating"
	ServiceColdUpdating ServiceStatus = "ColdUpdating"
	ServiceDeleting     ServiceStatus = "Deleting"
	ServiceDeleted      ServiceStatus = "Deleted"
)

// ParseServiceStatus parses a string into a ServiceStatus
func ParseServiceStatus(s string) (ServiceStatus, error) {
	status := ServiceStatus(s)
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

// Validate checks if the service status is valid
func (s ServiceStatus) Validate() error {
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
		return fmt.Errorf("invalid service status: %s", s)
	}
}

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity

	Name       string     `json:"name" gorm:"not null"`
	Attributes Attributes `json:"attributes,omitempty" gorm:"type:jsonb"`

	// Status management
	CurrentStatus     ServiceStatus  `json:"currentStatus" gorm:"not null"`
	TargetStatus      *ServiceStatus `json:"targetStatus,omitempty"`
	ErrorMessage      *string        `json:"errorMessage,omitempty"`
	FailedAction      *ServiceAction `json:"failedAction,omitempty"`
	RetryCount        int            `json:"retryCount"`
	CurrentProperties *JSON          `json:"currentProperties,omitempty" gorm:"type:jsonb"`
	TargetProperties  *JSON          `json:"targetProperties,omitempty" gorm:"type:jsonb"`

	// To store an external ID for the agent's use to facilitate metric reporting
	ExternalID *string `json:"externalId,omitempty" gorm:"uniqueIndex:service_external_id_uniq"`
	// Safe place for the Agent for store data
	Resources *JSON `json:"resources,omitempty" gorm:"type:jsonb"`

	// Relationships
	ProviderID    UUID          `json:"providerId" gorm:"not null"`
	Provider      *Participant  `json:"-" gorm:"foreignKey:ProviderID"`
	ConsumerID    UUID          `json:"consumerId" gorm:"not null"`
	Consumer      *Participant  `json:"-" gorm:"foreignKey:ConsumerID"`
	GroupID       UUID          `gorm:"not null" json:"groupId"`
	Group         *ServiceGroup `json:"-" gorm:"foreignKey:GroupID"`
	AgentID       UUID          `json:"agentId" gorm:"not null"`
	Agent         *Agent        `json:"-" gorm:"foreignKey:AgentID"`
	ServiceTypeID UUID          `json:"serviceTypeId" gorm:"not null"`
	ServiceType   *ServiceType  `json:"-" gorm:"foreignKey:ServiceTypeID"`
}

// NewService creates a new Service without validation
func NewService(
	consumerID UUID,
	groupID UUID,
	providerID UUID,
	agentID UUID,
	serviceTypeID UUID,
	name string,
	attributes Attributes,
	properties *JSON,
) *Service {
	target := ServiceCreated
	return &Service{
		ConsumerID:       consumerID,
		GroupID:          groupID,
		ProviderID:       providerID,
		AgentID:          agentID,
		ServiceTypeID:    serviceTypeID,
		Name:             name,
		CurrentStatus:    ServiceCreating,
		TargetStatus:     &target,
		Attributes:       attributes,
		TargetProperties: properties,
	}
}

// Update updates the service
func (s *Service) Update(name *string, props *JSON) (bool, *ServiceAction, error) {
	var (
		updated bool
		action  *ServiceAction
	)
	if name != nil {
		updated = true
		s.Name = *name
	}

	if props != nil && !reflect.DeepEqual(&s.CurrentProperties, *props) {
		updated = true

		transStatus, targetAction, err := serviceUpdateNextStatusAndAction(s.CurrentStatus)
		if err != nil {
			return false, nil, err
		}

		// Create and validate - use entity method
		target := s.CurrentStatus
		s.TargetProperties = props
		s.CurrentStatus = transStatus
		s.TargetStatus = &target

		action = &targetAction
	}

	return updated, action, nil
}

// serviceUpdateNextStatusAndAction determines the transition status and action when updating a service's properties based on its current status
func serviceUpdateNextStatusAndAction(status ServiceStatus) (ServiceStatus, ServiceAction, error) {
	switch status {
	case ServiceStopped:
		return ServiceColdUpdating, ServiceActionColdUpdate, nil
	case ServiceStarted:
		return ServiceHotUpdating, ServiceActionHotUpdate, nil
	default:
		return "", "", NewInvalidInputErrorf("cannot update attributes on a service with status %v", status)
	}
}

// Transition sets the service statuss for a transition
func (s *Service) Transition(targetStatus ServiceStatus) (*ServiceAction, error) {
	transStatus, action, err := serviceNextStatusAndAction(s.CurrentStatus, targetStatus)
	if err != nil {
		return nil, InvalidInputError{Err: err}
	}

	s.CurrentStatus = transStatus
	s.TargetStatus = &targetStatus

	return &action, nil
}

// serviceNextStatusAndAction determines the intermediate status and action for a service transition
func serviceNextStatusAndAction(curr, target ServiceStatus) (ServiceStatus, ServiceAction, error) {
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

// RetryFailedAction prepares a service for retry and returns a job for the failed action
func (s *Service) RetryFailedAction() *ServiceAction {
	if s.FailedAction == nil {
		return nil
	}

	s.RetryCount += 1

	return s.FailedAction
}

// Validate a service
func (s Service) Validate() error {
	if s.Name == "" {
		return errors.New("service name cannot be empty")
	}
	if err := s.CurrentStatus.Validate(); err != nil {
		return err
	}
	if s.TargetStatus != nil {
		if err := s.TargetStatus.Validate(); err != nil {
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

// ServiceCommander defines the interface for service command operations
type ServiceCommander interface {
	// Create handles service creation and creates a job for the agent
	Create(ctx context.Context, agentID UUID, serviceTypeID UUID, groupID UUID, name string, attributes Attributes, properties JSON) (*Service, error)

	// Update handles service updates and creates a job for the agent
	Update(ctx context.Context, id UUID, name *string, props *JSON) (*Service, error)

	// Transition transitions a service to a new status
	Transition(ctx context.Context, id UUID, target ServiceStatus) (*Service, error)

	// Retry retries a failed service operation
	Retry(ctx context.Context, id UUID) (*Service, error)

	// FailTimeoutServicesAndJobs fails services and jobs that have timed out
	FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error)
}

// serviceCommander is the concrete implementation of ServiceCommander
type serviceCommander struct {
	store          Store
	auditCommander AuditEntryCommander
}

// NewServiceCommander creates a new commander for services
func NewServiceCommander(
	store Store,
	auditCommander AuditEntryCommander,
) *serviceCommander {
	return &serviceCommander{
		store:          store,
		auditCommander: auditCommander,
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
	// Find and check dependencies
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	group, err := s.store.ServiceGroupRepo().FindByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Create and validate
	svc := NewService(
		group.ConsumerID,
		groupID,
		agent.ProviderID,
		agentID,
		serviceTypeID,
		name,
		attributes,
		&properties,
	)
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, audit and create job
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Create(ctx, svc); err != nil {
			return err
		}
		job := NewJob(svc, ServiceActionCreate, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeServiceCreated,
			JSON{"status": svc},
			&svc.ID,
			&svc.ProviderID,
			&svc.AgentID,
			&svc.ConsumerID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *serviceCommander) Update(ctx context.Context, id UUID, name *string, props *JSON) (*Service, error) {
	// Find it
	svc, err := s.store.ServiceRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Audit copy
	originalSvc := *svc

	// Update
	updateSvc, action, err := svc.Update(name, props)
	if err != nil {
		return nil, err
	}
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, audit and create job
	err = s.store.Atomic(ctx, func(store Store) error {
		if updateSvc {
			if err := store.ServiceRepo().Save(ctx, svc); err != nil {
				return err
			}
			if _, err := s.auditCommander.CreateCtxWithDiff(ctx, EventTypeServiceUpdated,
				&id, &svc.ProviderID, &svc.AgentID, &svc.ConsumerID,
				&originalSvc, svc); err != nil {
				return err
			}
		}
		if action != nil {
			job := NewJob(svc, *action, 1)
			if err := job.Validate(); err != nil {
				return err
			}
			if err := store.JobRepo().Create(ctx, job); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *serviceCommander) Transition(ctx context.Context, id UUID, target ServiceStatus) (*Service, error) {
	// Find it
	svc, err := s.store.ServiceRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Audit copy
	originalSvc := *svc

	// Transition
	action, err := svc.Transition(target)
	if err != nil {
		return nil, err
	}
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, audit and create job if needed
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		job := NewJob(svc, *action, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtxWithDiff(ctx, EventTypeServiceTransitioned,
			&id, &svc.ProviderID, &svc.AgentID, &svc.ConsumerID,
			&originalSvc, svc)
		return err
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *serviceCommander) Retry(ctx context.Context, id UUID) (*Service, error) {
	// Find it
	svc, err := s.store.ServiceRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Audit copy
	originalSvc := *svc

	// Retry
	action := svc.RetryFailedAction()
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if action == nil {
		return svc, nil // Nothing to retry
	}

	// Save, audit and create job if needed
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		job := NewJob(svc, *action, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtxWithDiff(ctx, EventTypeServiceRetried,
			&id, &svc.ProviderID, &svc.AgentID, &svc.ConsumerID, &originalSvc, svc)
		return err
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
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
			job.Status = JobFailed
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

// HandleJobComplete updates the service status when a job completes successfully
func (s *Service) HandleJobComplete(resources *JSON, externalID *string) error {
	if s.TargetStatus == nil {
		return InvalidInputError{Err: errors.New("cannot complete a job on service that is not in transition")}
	}

	s.CurrentStatus = *s.TargetStatus
	s.TargetStatus = nil
	s.FailedAction = nil
	s.ErrorMessage = nil
	s.RetryCount = 0

	if resources != nil {
		s.Resources = resources
	}
	if externalID != nil {
		s.ExternalID = externalID
	}
	if s.TargetProperties != nil {
		s.CurrentProperties = s.TargetProperties
		s.TargetProperties = nil
	}

	return nil
}

// HandleJobFailure updates the service status when a job fails
func (s *Service) HandleJobFailure(errorMessage string, action ServiceAction) {
	s.ErrorMessage = &errorMessage
	s.FailedAction = &action
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
	List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Service], error)

	// CountByGroup returns the number of services in a specific group
	CountByGroup(ctx context.Context, groupID UUID) (int64, error)

	// CountByAgent returns the number of services handled by a specific agent
	CountByAgent(ctx context.Context, agentID UUID) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}
