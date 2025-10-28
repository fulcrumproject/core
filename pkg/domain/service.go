package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

// Event types
const (
	EventTypeServiceCreated      EventType = "service.created"
	EventTypeServiceUpdated      EventType = "service.updated"
	EventTypeServiceTransitioned EventType = "service.transitioned"
	EventTypeServiceRetried      EventType = "service.retried"
)

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity

	Name       string           `json:"name" gorm:"not null"`
	Status     string           `json:"status" gorm:"not null"`
	Properties *properties.JSON `json:"properties,omitempty" gorm:"type:jsonb"`

	// Agent's native instance identifier for this service in their infrastructure system
	AgentInstanceID *string `json:"agentInstanceId,omitempty" gorm:"uniqueIndex:service_agent_instance_id_uniq"`
	// Safe place for the Agent to store data
	AgentInstanceData *properties.JSON `json:"agentInstanceData,omitempty" gorm:"type:jsonb"`

	// Relationships
	ProviderID    properties.UUID `json:"providerId" gorm:"not null"`
	Provider      *Participant    `json:"-" gorm:"foreignKey:ProviderID"`
	ConsumerID    properties.UUID `json:"consumerId" gorm:"not null"`
	Consumer      *Participant    `json:"-" gorm:"foreignKey:ConsumerID"`
	GroupID       properties.UUID `gorm:"not null" json:"groupId"`
	Group         *ServiceGroup   `json:"-" gorm:"foreignKey:GroupID"`
	AgentID       properties.UUID `json:"agentId" gorm:"not null"`
	Agent         *Agent          `json:"-" gorm:"foreignKey:AgentID"`
	ServiceTypeID properties.UUID `json:"serviceTypeId" gorm:"not null"`
	ServiceType   *ServiceType    `json:"-" gorm:"foreignKey:ServiceTypeID"`
}

// NewService creates a new Service without validation
func NewService(
	agent *Agent,
	group *ServiceGroup,
	params CreateServiceParams,
	initialStatus string,
) *Service {
	return &Service{
		ConsumerID:    group.ConsumerID,
		GroupID:       group.ID,
		ProviderID:    agent.ProviderID,
		AgentID:       agent.ID,
		ServiceTypeID: params.ServiceTypeID,
		Name:          params.Name,
		Status:        initialStatus,
		Properties:    &params.Properties,
	}
}

// HandleJobComplete handles the completion of a job
func (s *Service) HandleJobComplete(lifecycle LifecycleSchema, action string, errorCode *string, params *properties.JSON, agentInstanceData *properties.JSON, agentInstanceID *string) error {
	// Update status using lifecycle schema
	nextStatus, err := lifecycle.ResolveNextState(s.Status, action, errorCode)
	if err != nil {
		return err
	}
	s.Status = nextStatus

	// Update agent data and agent instance ID if provided
	if agentInstanceData != nil {
		s.AgentInstanceData = agentInstanceData
	}
	if agentInstanceID != nil {
		s.AgentInstanceID = agentInstanceID
	}

	// Update properties if the action is an update
	if action == "update" {
		s.Properties = params
	}

	return nil
}

// Update updates the service
func (s *Service) Update(name *string, properties *properties.JSON) (update bool, action bool, err error) {
	if name != nil {
		s.Name = *name
		update = true
	}

	if properties != nil {
		action = true
	}

	return update, action, nil
}

// ApplyAgentPropertyUpdates applies property updates from an agent using the schema engine
func ApplyAgentPropertyUpdates(
	ctx context.Context,
	store Store,
	engine *schema.Engine[ServicePropertyContext],
	svc *Service,
	serviceType *ServiceType,
	updates map[string]any,
) error {
	if len(updates) == 0 {
		return nil
	}

	// Ensure properties map exists
	if svc.Properties == nil {
		props := make(properties.JSON)
		svc.Properties = &props
	}

	// Create context for agent property updates
	schemaCtx := ServicePropertyContext{
		Actor:         ActorAgent,
		Store:         store,
		ProviderID:    svc.ProviderID,
		ConsumerID:    svc.ConsumerID,
		GroupID:       svc.GroupID,
		ServiceID:     &svc.ID,
		ServiceStatus: svc.Status,
	}
	// Set pool set ID if agent is loaded (needed for pool generators)
	if svc.Agent != nil {
		schemaCtx.ServicePoolSetID = svc.Agent.ServicePoolSetID
	}

	// Use engine to validate and process the updates
	oldProperties := map[string]any(*svc.Properties)
	validatedProperties, err := engine.ApplyUpdate(ctx, schemaCtx, serviceType.PropertySchema, oldProperties, updates)
	if err != nil {
		return err
	}

	// Merge validated properties
	for k, v := range validatedProperties {
		(*svc.Properties)[k] = v
	}

	return nil
}

// Validate a service
func (s *Service) Validate() error {
	if s.Name == "" {
		return errors.New("service name cannot be empty")
	}
	if s.Status == "" {
		return errors.New("service status cannot be empty")
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
	return nil
}

// TableName returns the table name for the service
func (Service) TableName() string {
	return "services"
}

// ServiceCommander defines the interface for service command operations
type ServiceCommander interface {
	// Create handles service creation and creates a job for the agent
	Create(ctx context.Context, params CreateServiceParams) (*Service, error)

	// CreateWithTags handles service creation using agent discovery by tags
	CreateWithTags(ctx context.Context, params CreateServiceWithTagsParams) (*Service, error)

	// Update handles service updates and creates a job for the agent
	Update(ctx context.Context, params UpdateServiceParams) (*Service, error)

	// DoAction handles service actions
	DoAction(ctx context.Context, params DoServiceActionParams) (*Service, error)

	// FailTimeoutServicesAndJobs fails services and jobs that have timed out
	FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error)
}

// serviceCommander is the concrete implementation of ServiceCommander
type serviceCommander struct {
	store  Store
	engine *schema.Engine[ServicePropertyContext]
}

// NewServiceCommander creates a new commander for services
func NewServiceCommander(
	store Store,
	engine *schema.Engine[ServicePropertyContext],
) *serviceCommander {
	return &serviceCommander{
		store:  store,
		engine: engine,
	}
}

type CreateServiceParams struct {
	AgentID       properties.UUID `json:"agentId"`
	ServiceTypeID properties.UUID `json:"serviceTypeId"`
	GroupID       properties.UUID `json:"groupId"`
	Name          string          `json:"name"`
	Properties    properties.JSON `json:"targetProperties"`
}

type CreateServiceWithTagsParams struct {
	CreateServiceParams
	ServiceTags []string `json:"agentTags,omitempty"`
}

type UpdateServiceParams struct {
	ID         properties.UUID  `json:"id"`
	Name       *string          `json:"name,omitempty"`
	Properties *properties.JSON `json:"properties,omitempty"`
}

type DoServiceActionParams struct {
	ID     properties.UUID `json:"id"`
	Action string          `json:"action"`
}

func (s *serviceCommander) Create(
	ctx context.Context,
	params CreateServiceParams,
) (*Service, error) {
	agent, err := s.store.AgentRepo().Get(ctx, params.AgentID)
	if err != nil {
		return nil, NewInvalidInputErrorf("agent with ID %s does not exist", params.AgentID)
	}

	return CreateServiceWithAgent(ctx, s.store, s.engine, agent, params)
}

func (s *serviceCommander) CreateWithTags(
	ctx context.Context,
	params CreateServiceWithTagsParams,
) (*Service, error) {
	return CreateServiceWithTags(ctx, s.store, s.engine, params)
}

func CreateServiceWithTags(
	ctx context.Context,
	store Store,
	engine *schema.Engine[ServicePropertyContext],
	params CreateServiceWithTagsParams,
) (*Service, error) {
	agents, err := store.AgentRepo().FindByServiceTypeAndTags(ctx, params.ServiceTypeID, params.ServiceTags)
	if err != nil {
		return nil, err
	}

	if len(agents) == 0 {
		return nil, NewInvalidInputErrorf("no agent found for service type %s with tags %v", params.ServiceTypeID, params.ServiceTags)
	}

	agent := agents[0]
	return CreateServiceWithAgent(ctx, store, engine, agent, params.CreateServiceParams)
}

func CreateServiceWithAgent(
	ctx context.Context,
	store Store,
	engine *schema.Engine[ServicePropertyContext],
	agent *Agent,
	params CreateServiceParams,
) (*Service, error) {
	group, err := store.ServiceGroupRepo().Get(ctx, params.GroupID)
	if err != nil {
		return nil, err
	}

	// Load ServiceType to get property schema
	serviceType, err := store.ServiceTypeRepo().Get(ctx, params.ServiceTypeID)
	if err != nil {
		return nil, err
	}

	// Extract actor from auth context
	identity := auth.MustGetIdentity(ctx)
	actor := ActorTypeFromAuthRole(identity.Role)

	// Generate service ID upfront so pool generators can use it for allocation tracking
	serviceID := properties.UUID(uuid.New())

	// Check if the agent's type supports the requested service type
	supported := false
	for _, agentServiceType := range agent.AgentType.ServiceTypes {
		if agentServiceType.ID == params.ServiceTypeID {
			supported = true
			break
		}
	}
	if !supported {
		return nil, NewInvalidInputErrorf("agent type %s does not support service type %s", agent.AgentType.Name, params.ServiceTypeID)
	}

	// Get initial state from lifecycle schema (always present)
	initialState := serviceType.LifecycleSchema.InitialState

	svc := NewService(
		agent,
		group,
		params,
		initialState,
	)
	// Set the pre-generated ID
	svc.ID = serviceID

	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = store.Atomic(ctx, func(txStore Store) error {
		// Validate and process properties using schema engine WITHIN transaction
		// This ensures pool allocations happen within the same transaction
		schemaCtx := ServicePropertyContext{
			Actor:            actor,
			Store:            txStore, // Use transactional store
			ProviderID:       agent.ProviderID,
			ConsumerID:       svc.ConsumerID,
			GroupID:          svc.GroupID,
			ServicePoolSetID: agent.ServicePoolSetID,
			ServiceID:        &serviceID,
			ServiceStatus:    "", // empty during create
		}

		validatedProperties, err := engine.ApplyCreate(ctx, schemaCtx, serviceType.PropertySchema, params.Properties)
		if err != nil {
			return err
		}
		params.Properties = validatedProperties

		// Update service with validated/generated properties
		svc.Properties = &params.Properties

		// Create service with pre-generated ID
		if err := txStore.ServiceRepo().Create(ctx, svc); err != nil {
			return err
		}

		// Create job with final properties (including allocated pool values)
		finalProps := params.Properties
		if svc.Properties != nil {
			finalProps = *svc.Properties
		}
		job := NewJob(svc, "create", &finalProps, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := txStore.JobRepo().Create(ctx, job); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceCreated, WithInitiatorCtx(ctx), WithService(svc))
		if err != nil {
			return err
		}
		if err := txStore.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *serviceCommander) Update(ctx context.Context, params UpdateServiceParams) (*Service, error) {
	return UpdateService(ctx, s.store, s.engine, params)
}

func UpdateService(ctx context.Context, store Store, engine *schema.Engine[ServicePropertyContext], params UpdateServiceParams) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Load ServiceType to get property schema and lifecycle
	serviceType, err := store.ServiceTypeRepo().Get(ctx, svc.ServiceTypeID)
	if err != nil {
		return nil, err
	}

	// Load agent to get pool set (needed for context, even if not updating properties)
	agent, err := store.AgentRepo().Get(ctx, svc.AgentID)
	if err != nil {
		return nil, err
	}

	// Extract actor from auth context (needed for context)
	identity := auth.MustGetIdentity(ctx)
	actor := ActorTypeFromAuthRole(identity.Role)

	// Update, if needed
	originalSvc := *svc
	update, action, err := svc.Update(params.Name, params.Properties)
	if err != nil {
		return nil, err
	}
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, event and create job
	err = store.Atomic(ctx, func(txStore Store) error {
		// Validate and process properties if provided WITHIN transaction
		if params.Properties != nil {
			// Build schema context with transactional store
			schemaCtx := ServicePropertyContext{
				Actor:            actor,
				Store:            txStore, // Use transactional store
				ProviderID:       svc.ProviderID,
				ConsumerID:       svc.ConsumerID,
				GroupID:          svc.GroupID,
				ServicePoolSetID: agent.ServicePoolSetID,
				ServiceID:        &svc.ID,
				ServiceStatus:    svc.Status,
			}

			// Convert existing properties to map
			oldProperties := map[string]any(*svc.Properties)

			// Engine handles merging: takes old properties and partial new properties
			validatedProperties, err := engine.ApplyUpdate(ctx, schemaCtx, serviceType.PropertySchema, oldProperties, *params.Properties)
			if err != nil {
				return err
			}
			convertedProperties := properties.JSON(validatedProperties)
			params.Properties = &convertedProperties

			// Update service with validated properties
			svc.Properties = params.Properties
		}
		if update {
			if err := txStore.ServiceRepo().Save(ctx, svc); err != nil {
				return err
			}
			eventEntry, err := NewEvent(EventTypeServiceUpdated, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
			if err != nil {
				return err
			}
			if err := txStore.EventRepo().Create(ctx, eventEntry); err != nil {
				return err
			}
		}
		if action {
			// Check if service is in a terminal state (lifecycle always present)
			if serviceType.LifecycleSchema.IsTerminalState(svc.Status) {
				return NewInvalidInputErrorf("cannot perform action on service in terminal state: %s", svc.Status)
			}

			// Check if the service is in a valid state to be updated with a job
			if err := serviceType.LifecycleSchema.ValidateActionAllowed(svc.Status, "update"); err != nil {
				return InvalidInputError{Err: err}
			}

			// If pending job exists, fail it
			err = checkHasNotActiveJob(ctx, txStore, svc)
			if err != nil {
				return err
			}

			// Create new job
			job := NewJob(svc, "update", params.Properties, 1)
			if err := job.Validate(); err != nil {
				return err
			}
			if err := txStore.JobRepo().Create(ctx, job); err != nil {
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

func (s *serviceCommander) DoAction(ctx context.Context, params DoServiceActionParams) (*Service, error) {
	return DoServiceAction(ctx, s.store, params)
}

func DoServiceAction(ctx context.Context, store Store, params DoServiceActionParams) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Load ServiceType to get lifecycle schema
	serviceType, err := store.ServiceTypeRepo().Get(ctx, svc.ServiceTypeID)
	if err != nil {
		return nil, err
	}

	// Check if service is in a terminal state (lifecycle always present)
	if serviceType.LifecycleSchema.IsTerminalState(svc.Status) {
		return nil, NewInvalidInputErrorf("cannot perform action on service in terminal state: %s", svc.Status)
	}

	// Check if the service is in a valid state to perform this action
	if err := serviceType.LifecycleSchema.ValidateActionAllowed(svc.Status, params.Action); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// If pending job exists, fail it
	err = checkHasNotActiveJob(ctx, store, svc)
	if err != nil {
		return nil, err
	}

	// Create the new job
	err = store.Atomic(ctx, func(store Store) error {
		job := NewJob(svc, params.Action, nil, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func checkHasNotActiveJob(ctx context.Context, store Store, svc *Service) error {
	job, err := store.JobRepo().GetLastJobForService(ctx, svc.ID)
	if err != nil {
		return err
	}
	if job != nil && job.IsActive() {
		return NewInvalidInputErrorf("cannot update service %s while there is an active job %s", svc.ID, job.ID)
	}
	return nil
}

func (s *serviceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	timedOutJobs, err := s.store.JobRepo().GetTimeOutJobs(ctx, timeout)
	if err != nil {
		return 0, fmt.Errorf("failed to retrive timeout jobs: %v", err)
	}

	counter := 0
	errorMsg := "Job marked as failed due to exceeding maximum processing time"
	for _, job := range timedOutJobs {
		// Update job to failed
		job.Status = JobFailed
		job.ErrorMessage = errorMsg
		now := time.Now()
		job.CompletedAt = &now
		if err := s.store.JobRepo().Save(ctx, job); err != nil {
			return counter, err
		}
		counter++
	}

	return counter, nil
}

// ServiceRepository defines the interface for the Service repository
type ServiceRepository interface {
	ServiceQuerier
	BaseEntityRepository[Service]
}

// ServiceQuerier defines the interface for the Service read-only queries
type ServiceQuerier interface {
	BaseEntityQuerier[Service]

	// FindByAgentInstanceID retrieves a service by its agent instance ID and agent ID
	FindByAgentInstanceID(ctx context.Context, agentID properties.UUID, agentInstanceID string) (*Service, error)

	// CountByGroup returns the number of services in a specific group
	CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error)

	// CountByAgent returns the number of services handled by a specific agent
	CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error)

	// CountByServiceType returns the number of services of a specific type
	CountByServiceType(ctx context.Context, serviceTypeID properties.UUID) (int64, error)
}
