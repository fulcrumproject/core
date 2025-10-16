package api

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
)

// mockEventSubscriptionCommander is a custom mock for EventSubscriptionCommander
type mockEventSubscriptionCommander struct {
	updateProgressFunc    func(ctx context.Context, params domain.UpdateProgressParams) (*domain.EventSubscription, error)
	acquireLeaseFunc      func(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error)
	renewLeaseFunc        func(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error)
	releaseLeaseFunc      func(ctx context.Context, params domain.ReleaseLeaseParams) (*domain.EventSubscription, error)
	acknowledgeEventsFunc func(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error)
	setActiveFunc         func(ctx context.Context, params domain.SetActiveParams) (*domain.EventSubscription, error)
	deleteFunc            func(ctx context.Context, subscriberID string) error
}

func (m *mockEventSubscriptionCommander) UpdateProgress(ctx context.Context, params domain.UpdateProgressParams) (*domain.EventSubscription, error) {
	if m.updateProgressFunc != nil {
		return m.updateProgressFunc(ctx, params)
	}
	return nil, fmt.Errorf("updateProgress not mocked")
}

func (m *mockEventSubscriptionCommander) AcquireLease(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error) {
	if m.acquireLeaseFunc != nil {
		return m.acquireLeaseFunc(ctx, params)
	}
	return nil, fmt.Errorf("acquireLease not mocked")
}

func (m *mockEventSubscriptionCommander) RenewLease(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error) {
	if m.renewLeaseFunc != nil {
		return m.renewLeaseFunc(ctx, params)
	}
	return nil, fmt.Errorf("renewLease not mocked")
}

func (m *mockEventSubscriptionCommander) ReleaseLease(ctx context.Context, params domain.ReleaseLeaseParams) (*domain.EventSubscription, error) {
	if m.releaseLeaseFunc != nil {
		return m.releaseLeaseFunc(ctx, params)
	}
	return nil, fmt.Errorf("releaseLease not mocked")
}

func (m *mockEventSubscriptionCommander) AcknowledgeEvents(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error) {
	if m.acknowledgeEventsFunc != nil {
		return m.acknowledgeEventsFunc(ctx, params)
	}
	return nil, fmt.Errorf("acknowledgeEvents not mocked")
}

func (m *mockEventSubscriptionCommander) SetActive(ctx context.Context, params domain.SetActiveParams) (*domain.EventSubscription, error) {
	if m.setActiveFunc != nil {
		return m.setActiveFunc(ctx, params)
	}
	return nil, fmt.Errorf("setActive not mocked")
}

func (m *mockEventSubscriptionCommander) Delete(ctx context.Context, subscriberID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, subscriberID)
	}
	return fmt.Errorf("delete not mocked")
}

// mockAgentCommander is a custom mock for AgentCommander
type mockAgentCommander struct {
	createFunc       func(ctx context.Context, params domain.CreateAgentParams) (*domain.Agent, error)
	updateFunc       func(ctx context.Context, params domain.UpdateAgentParams) (*domain.Agent, error)
	deleteFunc       func(ctx context.Context, id properties.UUID) error
	updateStatusFunc func(ctx context.Context, params domain.UpdateAgentStatusParams) (*domain.Agent, error)
}

func (m *mockAgentCommander) Create(ctx context.Context, params domain.CreateAgentParams) (*domain.Agent, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAgentCommander) Update(ctx context.Context, params domain.UpdateAgentParams) (*domain.Agent, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockAgentCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockAgentCommander) UpdateStatus(ctx context.Context, params domain.UpdateAgentStatusParams) (*domain.Agent, error) {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, params)
	}
	return nil, fmt.Errorf("update status not mocked")
}

type mockParticipantCommander struct {
	createFunc func(ctx context.Context, params domain.CreateParticipantParams) (*domain.Participant, error)
	updateFunc func(ctx context.Context, params domain.UpdateParticipantParams) (*domain.Participant, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockParticipantCommander) Create(ctx context.Context, params domain.CreateParticipantParams) (*domain.Participant, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Update(ctx context.Context, params domain.UpdateParticipantParams) (*domain.Participant, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// mockJobCommander is a custom mock for JobCommander
type mockJobCommander struct {
	claimFunc    func(ctx context.Context, jobID properties.UUID) error
	completeFunc func(ctx context.Context, params domain.CompleteJobParams) error
	failFunc     func(ctx context.Context, params domain.FailJobParams) error
}

func (m *mockJobCommander) Claim(ctx context.Context, jobID properties.UUID) error {
	if m.claimFunc != nil {
		return m.claimFunc(ctx, jobID)
	}
	return nil
}

func (m *mockJobCommander) Complete(ctx context.Context, params domain.CompleteJobParams) error {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, params)
	}
	return nil
}

func (m *mockJobCommander) Fail(ctx context.Context, params domain.FailJobParams) error {
	if m.failFunc != nil {
		return m.failFunc(ctx, params)
	}
	return nil
}

// MockMetricEntryCommander mocks the MetricEntryCommander interface
type mockMetricEntryCommander struct {
	createFunc                    func(ctx context.Context, params domain.CreateMetricEntryParams) (*domain.MetricEntry, error)
	createWithAgentInstanceIDFunc func(ctx context.Context, params domain.CreateMetricEntryWithAgentInstanceIDParams) (*domain.MetricEntry, error)
}

func (m *mockMetricEntryCommander) Create(ctx context.Context, params domain.CreateMetricEntryParams) (*domain.MetricEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricEntryCommander) CreateWithAgentInstanceID(ctx context.Context, params domain.CreateMetricEntryWithAgentInstanceIDParams) (*domain.MetricEntry, error) {
	if m.createWithAgentInstanceIDFunc != nil {
		return m.createWithAgentInstanceIDFunc(ctx, params)
	}
	return nil, fmt.Errorf("createWithAgentInstanceID not mocked")
}

// mockMetricTypeCommander is a custom mock for MetricTypeCommander
type mockMetricTypeCommander struct {
	createFunc func(ctx context.Context, params domain.CreateMetricTypeParams) (*domain.MetricType, error)
	updateFunc func(ctx context.Context, params domain.UpdateMetricTypeParams) (*domain.MetricType, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockMetricTypeCommander) Create(ctx context.Context, params domain.CreateMetricTypeParams) (*domain.MetricType, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricTypeCommander) Update(ctx context.Context, params domain.UpdateMetricTypeParams) (*domain.MetricType, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockMetricTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockServiceGroupCommander is a custom mock for ServiceGroupCommander
type mockServiceGroupCommander struct {
	createFunc func(ctx context.Context, params domain.CreateServiceGroupParams) (*domain.ServiceGroup, error)
	updateFunc func(ctx context.Context, params domain.UpdateServiceGroupParams) (*domain.ServiceGroup, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockServiceGroupCommander) Create(ctx context.Context, params domain.CreateServiceGroupParams) (*domain.ServiceGroup, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceGroupCommander) Update(ctx context.Context, params domain.UpdateServiceGroupParams) (*domain.ServiceGroup, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceGroupCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockServiceCommander is a custom mock for ServiceCommander
type mockServiceCommander struct {
	createFunc                     func(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error)
	createWithTagsFunc             func(ctx context.Context, params domain.CreateServiceWithTagsParams) (*domain.Service, error)
	updateFunc                     func(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error)
	doActionFunc                   func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error)
	retryFunc                      func(ctx context.Context, id properties.UUID) (*domain.Service, error)
	failTimeoutServicesAndJobsFunc func(ctx context.Context, timeout time.Duration) (int, error)
}

func (m *mockServiceCommander) Create(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceCommander) CreateWithTags(ctx context.Context, params domain.CreateServiceWithTagsParams) (*domain.Service, error) {
	if m.createWithTagsFunc != nil {
		return m.createWithTagsFunc(ctx, params)
	}
	return nil, fmt.Errorf("createWithTags not mocked")
}

func (m *mockServiceCommander) Update(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceCommander) DoAction(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
	if m.doActionFunc != nil {
		return m.doActionFunc(ctx, params)
	}
	return nil, fmt.Errorf("doAction not mocked")
}

func (m *mockServiceCommander) Retry(ctx context.Context, id properties.UUID) (*domain.Service, error) {
	if m.retryFunc != nil {
		return m.retryFunc(ctx, id)
	}
	return nil, fmt.Errorf("retry not mocked")
}

func (m *mockServiceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	if m.failTimeoutServicesAndJobsFunc != nil {
		return m.failTimeoutServicesAndJobsFunc(ctx, timeout)
	}
	return 0, fmt.Errorf("failTimeoutServicesAndJobs not mocked")
}

// mockTokenCommander is a custom mock for TokenCommander
type mockTokenCommander struct {
	createFunc     func(ctx context.Context, params domain.CreateTokenParams) (*domain.Token, error)
	updateFunc     func(ctx context.Context, params domain.UpdateTokenParams) (*domain.Token, error)
	deleteFunc     func(ctx context.Context, id properties.UUID) error
	regenerateFunc func(ctx context.Context, id properties.UUID) (*domain.Token, error)
}

func (m *mockTokenCommander) Create(ctx context.Context, params domain.CreateTokenParams) (*domain.Token, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockTokenCommander) Update(ctx context.Context, params domain.UpdateTokenParams) (*domain.Token, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockTokenCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockTokenCommander) Regenerate(ctx context.Context, id properties.UUID) (*domain.Token, error) {
	if m.regenerateFunc != nil {
		return m.regenerateFunc(ctx, id)
	}
	return nil, fmt.Errorf("regenerate not mocked")
}

// createMockServiceTypeCommander creates a ServiceTypeCommander with a mock store
func createMockServiceTypeCommander() domain.ServiceTypeCommander {
	// Create a simple mock store for testing
	mockStore := newMockStoreForServiceType()
	return domain.NewServiceTypeCommander(mockStore)
}

// mockStoreForServiceType is a minimal mock store for ServiceTypeCommander
type mockStoreForServiceType struct {
	serviceTypeRepo domain.ServiceTypeRepository
	serviceRepo     domain.ServiceRepository
	eventRepo       domain.EventRepository
}

func newMockStoreForServiceType() *mockStoreForServiceType {
	return &mockStoreForServiceType{
		serviceTypeRepo: &mockServiceTypeRepository{},
		serviceRepo:     &mockServiceRepository{},
		eventRepo:       &mockEventRepository{},
	}
}

func (m *mockStoreForServiceType) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return fn(m)
}

// Required Store interface methods (minimal implementation)
func (m *mockStoreForServiceType) AgentTypeRepo() domain.AgentTypeRepository { return nil }
func (m *mockStoreForServiceType) AgentRepo() domain.AgentRepository         { return nil }
func (m *mockStoreForServiceType) TokenRepo() domain.TokenRepository         { return nil }
func (m *mockStoreForServiceType) ServiceTypeRepo() domain.ServiceTypeRepository {
	return m.serviceTypeRepo
}
func (m *mockStoreForServiceType) ServiceGroupRepo() domain.ServiceGroupRepository { return nil }
func (m *mockStoreForServiceType) ServiceRepo() domain.ServiceRepository           { return m.serviceRepo }
func (m *mockStoreForServiceType) JobRepo() domain.JobRepository                   { return nil }
func (m *mockStoreForServiceType) EventRepo() domain.EventRepository               { return m.eventRepo }
func (m *mockStoreForServiceType) EventSubscriptionRepo() domain.EventSubscriptionRepository {
	return nil
}
func (m *mockStoreForServiceType) MetricTypeRepo() domain.MetricTypeRepository   { return nil }
func (m *mockStoreForServiceType) ParticipantRepo() domain.ParticipantRepository { return nil }
func (m *mockStoreForServiceType) MetricEntryRepo() domain.MetricEntryRepository { return nil }

// Mock repository implementations for ServiceTypeCommander testing
type mockServiceTypeRepository struct{}

func (m *mockServiceTypeRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServiceType, error) {
	return &domain.ServiceType{Name: "Test Service Type"}, nil
}
func (m *mockServiceTypeRepository) Create(ctx context.Context, entity *domain.ServiceType) error {
	return nil
}
func (m *mockServiceTypeRepository) Save(ctx context.Context, entity *domain.ServiceType) error {
	return nil
}
func (m *mockServiceTypeRepository) Delete(ctx context.Context, id properties.UUID) error {
	return nil
}
func (m *mockServiceTypeRepository) List(ctx context.Context, scope *auth.IdentityScope, req *domain.PageReq) (*domain.PageRes[domain.ServiceType], error) {
	return nil, nil
}
func (m *mockServiceTypeRepository) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockServiceTypeRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockServiceTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.AllwaysMatchObjectScope{}, nil
}

type mockServiceRepository struct{}

func (m *mockServiceRepository) CountByServiceType(ctx context.Context, serviceTypeID properties.UUID) (int64, error) {
	return 0, nil
}
func (m *mockServiceRepository) Get(ctx context.Context, id properties.UUID) (*domain.Service, error) {
	return nil, nil
}
func (m *mockServiceRepository) Create(ctx context.Context, entity *domain.Service) error {
	return nil
}
func (m *mockServiceRepository) Save(ctx context.Context, entity *domain.Service) error {
	return nil
}
func (m *mockServiceRepository) Delete(ctx context.Context, id properties.UUID) error {
	return nil
}
func (m *mockServiceRepository) List(ctx context.Context, scope *auth.IdentityScope, req *domain.PageReq) (*domain.PageRes[domain.Service], error) {
	return nil, nil
}
func (m *mockServiceRepository) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockServiceRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockServiceRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.AllwaysMatchObjectScope{}, nil
}
func (m *mockServiceRepository) FindByAgentInstanceID(ctx context.Context, agentID properties.UUID, agentInstanceID string) (*domain.Service, error) {
	return nil, nil
}
func (m *mockServiceRepository) CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error) {
	return 0, nil
}
func (m *mockServiceRepository) CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error) {
	return 0, nil
}

type mockEventRepository struct{}

func (m *mockEventRepository) Create(ctx context.Context, entry *domain.Event) error {
	return nil
}
func (m *mockEventRepository) Get(ctx context.Context, id properties.UUID) (*domain.Event, error) {
	return nil, nil
}
func (m *mockEventRepository) Save(ctx context.Context, entity *domain.Event) error {
	return nil
}
func (m *mockEventRepository) Delete(ctx context.Context, id properties.UUID) error {
	return nil
}
func (m *mockEventRepository) List(ctx context.Context, scope *auth.IdentityScope, req *domain.PageReq) (*domain.PageRes[domain.Event], error) {
	return nil, nil
}
func (m *mockEventRepository) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockEventRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockEventRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.AllwaysMatchObjectScope{}, nil
}
func (m *mockEventRepository) ListFromSequence(ctx context.Context, fromSequenceNumber int64, limit int) ([]*domain.Event, error) {
	return nil, nil
}
func (m *mockEventRepository) ServiceUptime(ctx context.Context, serviceID properties.UUID, start time.Time, end time.Time) (uint64, uint64, error) {
	return 0, 0, nil
}

// createMockAgentTypeCommander creates an AgentTypeCommander with a mock store
func createMockAgentTypeCommander() domain.AgentTypeCommander {
	// Create a simple mock store for testing
	mockStore := newMockStoreForAgentType()
	return domain.NewAgentTypeCommander(mockStore)
}

// mockStoreForAgentType is a minimal mock store for AgentTypeCommander
type mockStoreForAgentType struct {
	agentTypeRepo domain.AgentTypeRepository
	agentRepo     domain.AgentRepository
	eventRepo     domain.EventRepository
}

func newMockStoreForAgentType() *mockStoreForAgentType {
	return &mockStoreForAgentType{
		agentTypeRepo: &mockAgentTypeRepository{},
		agentRepo:     &mockAgentRepository{},
		eventRepo:     &mockEventRepository{},
	}
}

func (m *mockStoreForAgentType) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return fn(m)
}

// Required Store interface methods (minimal implementation)
func (m *mockStoreForAgentType) AgentTypeRepo() domain.AgentTypeRepository       { return m.agentTypeRepo }
func (m *mockStoreForAgentType) AgentRepo() domain.AgentRepository               { return m.agentRepo }
func (m *mockStoreForAgentType) TokenRepo() domain.TokenRepository               { return nil }
func (m *mockStoreForAgentType) ServiceTypeRepo() domain.ServiceTypeRepository   { return nil }
func (m *mockStoreForAgentType) ServiceGroupRepo() domain.ServiceGroupRepository { return nil }
func (m *mockStoreForAgentType) ServiceRepo() domain.ServiceRepository           { return nil }
func (m *mockStoreForAgentType) JobRepo() domain.JobRepository                   { return nil }
func (m *mockStoreForAgentType) EventRepo() domain.EventRepository               { return m.eventRepo }
func (m *mockStoreForAgentType) EventSubscriptionRepo() domain.EventSubscriptionRepository {
	return nil
}
func (m *mockStoreForAgentType) MetricTypeRepo() domain.MetricTypeRepository   { return nil }
func (m *mockStoreForAgentType) ParticipantRepo() domain.ParticipantRepository { return nil }
func (m *mockStoreForAgentType) MetricEntryRepo() domain.MetricEntryRepository { return nil }

// Mock repository implementations for AgentTypeCommander testing
type mockAgentTypeRepository struct{}

func (m *mockAgentTypeRepository) Get(ctx context.Context, id properties.UUID) (*domain.AgentType, error) {
	return &domain.AgentType{Name: "Test Agent Type"}, nil
}
func (m *mockAgentTypeRepository) Create(ctx context.Context, entity *domain.AgentType) error {
	return nil
}
func (m *mockAgentTypeRepository) Save(ctx context.Context, entity *domain.AgentType) error {
	return nil
}
func (m *mockAgentTypeRepository) Delete(ctx context.Context, id properties.UUID) error {
	return nil
}
func (m *mockAgentTypeRepository) List(ctx context.Context, scope *auth.IdentityScope, req *domain.PageReq) (*domain.PageRes[domain.AgentType], error) {
	return nil, nil
}
func (m *mockAgentTypeRepository) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockAgentTypeRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockAgentTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.AllwaysMatchObjectScope{}, nil
}

type mockAgentRepository struct{}

func (m *mockAgentRepository) CountByAgentType(ctx context.Context, agentTypeID properties.UUID) (int64, error) {
	return 0, nil
}
func (m *mockAgentRepository) Get(ctx context.Context, id properties.UUID) (*domain.Agent, error) {
	return nil, nil
}
func (m *mockAgentRepository) Create(ctx context.Context, entity *domain.Agent) error {
	return nil
}
func (m *mockAgentRepository) Save(ctx context.Context, entity *domain.Agent) error {
	return nil
}
func (m *mockAgentRepository) Delete(ctx context.Context, id properties.UUID) error {
	return nil
}
func (m *mockAgentRepository) List(ctx context.Context, scope *auth.IdentityScope, req *domain.PageReq) (*domain.PageRes[domain.Agent], error) {
	return nil, nil
}
func (m *mockAgentRepository) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockAgentRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockAgentRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.AllwaysMatchObjectScope{}, nil
}
func (m *mockAgentRepository) CountByProvider(ctx context.Context, providerID properties.UUID) (int64, error) {
	return 0, nil
}
func (m *mockAgentRepository) FindByServiceTypeAndTags(ctx context.Context, serviceTypeID properties.UUID, tags []string) ([]*domain.Agent, error) {
	return nil, nil
}
func (m *mockAgentRepository) UpdateStatus(ctx context.Context, agentID properties.UUID, status domain.AgentStatus) error {
	return nil
}
func (m *mockAgentRepository) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	return 0, nil
}
