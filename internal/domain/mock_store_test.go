package domain

import (
	"context"
	"time"
)

// Ensure interface compatibility
var _ Store = (*MockStore)(nil)

// MockStore implements the Store interface for testing
type MockStore struct {
	atomicFunc           func(context.Context, func(Store) error) error
	agentTypeRepoFunc    func() AgentTypeRepository
	agentRepoFunc        func() AgentRepository
	tokenRepoFunc        func() TokenRepository
	serviceTypeRepoFunc  func() ServiceTypeRepository
	serviceGroupRepoFunc func() ServiceGroupRepository
	serviceRepoFunc      func() ServiceRepository
	jobRepoFunc          func() JobRepository
	auditEntryRepoFunc   func() AuditEntryRepository
	metricTypeRepoFunc   func() MetricTypeRepository
	metricEntryRepoFunc  func() MetricEntryRepository
	participantRepoFunc  func() ParticipantRepository

	// Default repositories
	agentTypeRepo    AgentTypeRepository
	agentRepo        AgentRepository
	tokenRepo        TokenRepository
	serviceTypeRepo  ServiceTypeRepository
	serviceGroupRepo ServiceGroupRepository
	serviceRepo      ServiceRepository
	jobRepo          JobRepository
	auditEntryRepo   AuditEntryRepository
	metricTypeRepo   MetricTypeRepository
	metricEntryRepo  MetricEntryRepository
	participantRepo  ParticipantRepository
}

// NewMockStore creates a new MockStore with default mock repositories
func NewMockStore() *MockStore {
	store := &MockStore{
		agentTypeRepo:    &MockAgentTypeRepository{},
		agentRepo:        &MockAgentRepository{},
		tokenRepo:        &MockTokenRepository{},
		serviceTypeRepo:  &MockServiceTypeRepository{},
		serviceGroupRepo: &MockServiceGroupRepository{},
		serviceRepo:      &MockServiceRepository{},
		jobRepo:          &MockJobRepository{},
		auditEntryRepo:   &MockAuditEntryRepository{},
		metricTypeRepo:   &MockMetricTypeRepository{},
		metricEntryRepo:  &MockMetricEntryRepository{},
		participantRepo:  &MockParticipantRepository{},
	}
	return store
}

// Atomic executes function in a transaction
func (m *MockStore) Atomic(ctx context.Context, fn func(Store) error) error {
	if m.atomicFunc != nil {
		return m.atomicFunc(ctx, fn)
	}
	// Default implementation just runs the function with this store
	return fn(m)
}

// AgentTypeRepo returns the AgentTypeRepository
func (m *MockStore) AgentTypeRepo() AgentTypeRepository {
	if m.agentTypeRepoFunc != nil {
		return m.agentTypeRepoFunc()
	}
	return m.agentTypeRepo
}

// AgentRepo returns the AgentRepository
func (m *MockStore) AgentRepo() AgentRepository {
	if m.agentRepoFunc != nil {
		return m.agentRepoFunc()
	}
	return m.agentRepo
}

// TokenRepo returns the TokenRepository
func (m *MockStore) TokenRepo() TokenRepository {
	if m.tokenRepoFunc != nil {
		return m.tokenRepoFunc()
	}
	return m.tokenRepo
}

// ServiceTypeRepo returns the ServiceTypeRepository
func (m *MockStore) ServiceTypeRepo() ServiceTypeRepository {
	if m.serviceTypeRepoFunc != nil {
		return m.serviceTypeRepoFunc()
	}
	return m.serviceTypeRepo
}

// ServiceGroupRepo returns the ServiceGroupRepository
func (m *MockStore) ServiceGroupRepo() ServiceGroupRepository {
	if m.serviceGroupRepoFunc != nil {
		return m.serviceGroupRepoFunc()
	}
	return m.serviceGroupRepo
}

// ServiceRepo returns the ServiceRepository
func (m *MockStore) ServiceRepo() ServiceRepository {
	if m.serviceRepoFunc != nil {
		return m.serviceRepoFunc()
	}
	return m.serviceRepo
}

// JobRepo returns the JobRepository
func (m *MockStore) JobRepo() JobRepository {
	if m.jobRepoFunc != nil {
		return m.jobRepoFunc()
	}
	return m.jobRepo
}

// AuditEntryRepo returns the AuditEntryRepository
func (m *MockStore) AuditEntryRepo() AuditEntryRepository {
	if m.auditEntryRepoFunc != nil {
		return m.auditEntryRepoFunc()
	}
	return m.auditEntryRepo
}

// MetricTypeRepo returns the MetricTypeRepository
func (m *MockStore) MetricTypeRepo() MetricTypeRepository {
	if m.metricTypeRepoFunc != nil {
		return m.metricTypeRepoFunc()
	}
	return m.metricTypeRepo
}

// MetricEntryRepo returns the MetricEntryRepository
func (m *MockStore) MetricEntryRepo() MetricEntryRepository {
	if m.metricEntryRepoFunc != nil {
		return m.metricEntryRepoFunc()
	}
	return m.metricEntryRepo
}

// ParticipantRepo returns the ParticipantRepository
func (m *MockStore) ParticipantRepo() ParticipantRepository {
	if m.participantRepoFunc != nil {
		return m.participantRepoFunc()
	}
	return m.participantRepo
}

// WithAtomicFunc sets the atomic function and returns the store
func (m *MockStore) WithAtomicFunc(fn func(context.Context, func(Store) error) error) *MockStore {
	m.atomicFunc = fn
	return m
}

// WithAgentTypeRepo sets the agent type repository and returns the store
func (m *MockStore) WithAgentTypeRepo(repo AgentTypeRepository) *MockStore {
	m.agentTypeRepo = repo
	return m
}

// WithAgentRepo sets the agent repository and returns the store
func (m *MockStore) WithAgentRepo(repo AgentRepository) *MockStore {
	m.agentRepo = repo
	return m
}

// WithTokenRepo sets the token repository and returns the store
func (m *MockStore) WithTokenRepo(repo TokenRepository) *MockStore {
	m.tokenRepo = repo
	return m
}

// WithServiceTypeRepo sets the service type repository and returns the store
func (m *MockStore) WithServiceTypeRepo(repo ServiceTypeRepository) *MockStore {
	m.serviceTypeRepo = repo
	return m
}

// WithServiceGroupRepo sets the service group repository and returns the store
func (m *MockStore) WithServiceGroupRepo(repo ServiceGroupRepository) *MockStore {
	m.serviceGroupRepo = repo
	return m
}

// WithServiceRepo sets the service repository and returns the store
func (m *MockStore) WithServiceRepo(repo ServiceRepository) *MockStore {
	m.serviceRepo = repo
	return m
}

// WithJobRepo sets the job repository and returns the store
func (m *MockStore) WithJobRepo(repo JobRepository) *MockStore {
	m.jobRepo = repo
	return m
}

// WithAuditEntryRepo sets the audit entry repository and returns the store
func (m *MockStore) WithAuditEntryRepo(repo AuditEntryRepository) *MockStore {
	m.auditEntryRepo = repo
	return m
}

// WithMetricTypeRepo sets the metric type repository and returns the store
func (m *MockStore) WithMetricTypeRepo(repo MetricTypeRepository) *MockStore {
	m.metricTypeRepo = repo
	return m
}

// WithMetricEntryRepo sets the metric entry repository and returns the store
func (m *MockStore) WithMetricEntryRepo(repo MetricEntryRepository) *MockStore {
	m.metricEntryRepo = repo
	return m
}

// WithParticipantRepo sets the participant repository and returns the store
func (m *MockStore) WithParticipantRepo(repo ParticipantRepository) *MockStore {
	m.participantRepo = repo
	return m
}

// MockAgentTypeRepository implements the AgentTypeRepository interface for testing
type MockAgentTypeRepository struct {
	createFunc    func(ctx context.Context, agentType *AgentType) error
	updateFunc    func(ctx context.Context, agentType *AgentType) error
	deleteFunc    func(ctx context.Context, id UUID) error
	findByIDFunc  func(ctx context.Context, id UUID) (*AgentType, error)
	listFunc      func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[AgentType], error)
	existsFunc    func(ctx context.Context, id UUID) (bool, error)
	countFunc     func(ctx context.Context) (int64, error)
	authScopeFunc func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockAgentTypeRepository) Create(ctx context.Context, agentType *AgentType) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, agentType)
	}
	return nil
}

func (m *MockAgentTypeRepository) Update(ctx context.Context, agentType *AgentType) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, agentType)
	}
	return nil
}

func (m *MockAgentTypeRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockAgentTypeRepository) FindByID(ctx context.Context, id UUID) (*AgentType, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("agent type not found")
}

func (m *MockAgentTypeRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[AgentType], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[AgentType]{
		Items:       []AgentType{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockAgentTypeRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockAgentTypeRepository) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

func (m *MockAgentTypeRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

func (m *MockAgentTypeRepository) Save(ctx context.Context, agentType *AgentType) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, agentType)
	}
	return nil
}

// MockAgentRepository implements the AgentRepository interface for testing
type MockAgentRepository struct {
	createFunc                           func(ctx context.Context, agent *Agent) error
	updateFunc                           func(ctx context.Context, agent *Agent) error
	deleteFunc                           func(ctx context.Context, id UUID) error
	findByIDFunc                         func(ctx context.Context, id UUID) (*Agent, error)
	listFunc                             func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Agent], error)
	existsFunc                           func(ctx context.Context, id UUID) (bool, error)
	countByProviderFunc                  func(ctx context.Context, providerID UUID) (int64, error)
	authScopeFunc                        func(ctx context.Context, id UUID) (*AuthTargetScope, error)
	markInactiveAgentsAsDisconnectedFunc func(ctx context.Context, inactiveThreshold time.Duration) (int64, error)
}

func (m *MockAgentRepository) Create(ctx context.Context, agent *Agent) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, agent)
	}
	return nil
}

func (m *MockAgentRepository) Update(ctx context.Context, agent *Agent) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, agent)
	}
	return nil
}

func (m *MockAgentRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockAgentRepository) FindByID(ctx context.Context, id UUID) (*Agent, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("agent not found")
}

func (m *MockAgentRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Agent], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[Agent]{
		Items:       []Agent{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockAgentRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockAgentRepository) CountByProvider(ctx context.Context, providerID UUID) (int64, error) {
	if m.countByProviderFunc != nil {
		return m.countByProviderFunc(ctx, providerID)
	}
	return 0, nil
}

func (m *MockAgentRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// MarkInactiveAgentsAsDisconnected marks inactive agents as disconnected
func (m *MockAgentRepository) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveThreshold time.Duration) (int64, error) {
	if m.markInactiveAgentsAsDisconnectedFunc != nil {
		return m.markInactiveAgentsAsDisconnectedFunc(ctx, inactiveThreshold)
	}
	return 0, nil
}

// Save updates an existing agent
func (m *MockAgentRepository) Save(ctx context.Context, agent *Agent) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, agent)
	}
	return nil
}

// MockParticipantRepository implements the ParticipantRepository interface for testing
type MockParticipantRepository struct {
	createFunc    func(ctx context.Context, participant *Participant) error
	saveFunc      func(ctx context.Context, participant *Participant) error
	deleteFunc    func(ctx context.Context, id UUID) error
	findByIDFunc  func(ctx context.Context, id UUID) (*Participant, error)
	listFunc      func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Participant], error)
	existsFunc    func(ctx context.Context, id UUID) (bool, error)
	authScopeFunc func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockParticipantRepository) Create(ctx context.Context, participant *Participant) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, participant)
	}
	return nil
}

func (m *MockParticipantRepository) Save(ctx context.Context, participant *Participant) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, participant)
	}
	return nil
}

func (m *MockParticipantRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockParticipantRepository) FindByID(ctx context.Context, id UUID) (*Participant, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("participant not found")
}

func (m *MockParticipantRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Participant], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[Participant]{
		Items:       []Participant{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockParticipantRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil // Default to true for mock existence checks
}

func (m *MockParticipantRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil // Default to empty scope
}

// MockTokenRepository implements the TokenRepository interface for testing
type MockTokenRepository struct {
	createFunc                func(ctx context.Context, token *Token) error
	updateFunc                func(ctx context.Context, token *Token) error
	deleteFunc                func(ctx context.Context, id UUID) error
	findByIDFunc              func(ctx context.Context, id UUID) (*Token, error)
	listFunc                  func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Token], error)
	existsFunc                func(ctx context.Context, id UUID) (bool, error)
	authScopeFunc             func(ctx context.Context, id UUID) (*AuthTargetScope, error)
	findByHashedValueFunc     func(ctx context.Context, hashedValue string) (*Token, error)
	findByValueFunc           func(ctx context.Context, value string) (*Token, error)
	deleteByAgentIDFunc       func(ctx context.Context, agentID UUID) error
	deleteByParticipantIDFunc func(ctx context.Context, participantID UUID) error
}

func (m *MockTokenRepository) Create(ctx context.Context, token *Token) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, token)
	}
	return nil
}

func (m *MockTokenRepository) Update(ctx context.Context, token *Token) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, token)
	}
	return nil
}

func (m *MockTokenRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockTokenRepository) FindByID(ctx context.Context, id UUID) (*Token, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("token not found")
}

func (m *MockTokenRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Token], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[Token]{
		Items:       []Token{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockTokenRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockTokenRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

func (m *MockTokenRepository) FindByValue(ctx context.Context, value string) (*Token, error) {
	if m.findByValueFunc != nil {
		return m.findByValueFunc(ctx, value)
	}
	return nil, NewNotFoundErrorf("token not found")
}

// FindByHashedValue retrieves a token by its hashed value
func (m *MockTokenRepository) FindByHashedValue(ctx context.Context, hashedValue string) (*Token, error) {
	if m.findByHashedValueFunc != nil {
		return m.findByHashedValueFunc(ctx, hashedValue)
	}
	return nil, NewNotFoundErrorf("token not found")
}

// Save updates an existing token
func (m *MockTokenRepository) Save(ctx context.Context, token *Token) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, token)
	}
	return nil
}

// DeleteByAgentID deletes tokens by agent ID
func (m *MockTokenRepository) DeleteByAgentID(ctx context.Context, agentID UUID) error {
	if m.deleteByAgentIDFunc != nil {
		return m.deleteByAgentIDFunc(ctx, agentID)
	}
	return nil
}

// DeleteByParticipantID deletes tokens by participant ID
func (m *MockTokenRepository) DeleteByParticipantID(ctx context.Context, participantID UUID) error {
	if m.deleteByParticipantIDFunc != nil {
		return m.deleteByParticipantIDFunc(ctx, participantID)
	}
	return nil
}

// MockServiceTypeRepository implements the ServiceTypeRepository interface for testing
type MockServiceTypeRepository struct {
	createFunc    func(ctx context.Context, serviceType *ServiceType) error
	updateFunc    func(ctx context.Context, serviceType *ServiceType) error
	deleteFunc    func(ctx context.Context, id UUID) error
	findByIDFunc  func(ctx context.Context, id UUID) (*ServiceType, error)
	listFunc      func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[ServiceType], error)
	existsFunc    func(ctx context.Context, id UUID) (bool, error)
	countFunc     func(ctx context.Context) (int64, error)
	authScopeFunc func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockServiceTypeRepository) Create(ctx context.Context, serviceType *ServiceType) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, serviceType)
	}
	return nil
}

func (m *MockServiceTypeRepository) Update(ctx context.Context, serviceType *ServiceType) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, serviceType)
	}
	return nil
}

func (m *MockServiceTypeRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockServiceTypeRepository) FindByID(ctx context.Context, id UUID) (*ServiceType, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("service type not found")
}

func (m *MockServiceTypeRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[ServiceType], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[ServiceType]{
		Items:       []ServiceType{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockServiceTypeRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockServiceTypeRepository) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

func (m *MockServiceTypeRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// Save updates an existing service type
func (m *MockServiceTypeRepository) Save(ctx context.Context, serviceType *ServiceType) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, serviceType)
	}
	return nil
}

// MockServiceGroupRepository implements the ServiceGroupRepository interface for testing
type MockServiceGroupRepository struct {
	createFunc    func(ctx context.Context, serviceGroup *ServiceGroup) error
	updateFunc    func(ctx context.Context, serviceGroup *ServiceGroup) error
	deleteFunc    func(ctx context.Context, id UUID) error
	findByIDFunc  func(ctx context.Context, id UUID) (*ServiceGroup, error)
	listFunc      func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[ServiceGroup], error)
	existsFunc    func(ctx context.Context, id UUID) (bool, error)
	countFunc     func(ctx context.Context) (int64, error)
	authScopeFunc func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockServiceGroupRepository) Create(ctx context.Context, serviceGroup *ServiceGroup) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, serviceGroup)
	}
	return nil
}

func (m *MockServiceGroupRepository) Update(ctx context.Context, serviceGroup *ServiceGroup) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, serviceGroup)
	}
	return nil
}

func (m *MockServiceGroupRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockServiceGroupRepository) FindByID(ctx context.Context, id UUID) (*ServiceGroup, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("service group not found")
}

func (m *MockServiceGroupRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[ServiceGroup], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[ServiceGroup]{
		Items:       []ServiceGroup{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockServiceGroupRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockServiceGroupRepository) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

func (m *MockServiceGroupRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// Save updates an existing service group
func (m *MockServiceGroupRepository) Save(ctx context.Context, serviceGroup *ServiceGroup) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, serviceGroup)
	}
	return nil
}

// MockServiceRepository implements the ServiceRepository interface for testing
type MockServiceRepository struct {
	createFunc           func(ctx context.Context, service *Service) error
	updateFunc           func(ctx context.Context, service *Service) error
	saveFunc             func(ctx context.Context, service *Service) error
	deleteFunc           func(ctx context.Context, id UUID) error
	findByIDFunc         func(ctx context.Context, id UUID) (*Service, error)
	findByExternalIDFunc func(ctx context.Context, agentID UUID, externalID string) (*Service, error)
	listFunc             func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Service], error)
	existsFunc           func(ctx context.Context, id UUID) (bool, error)
	countByGroupFunc     func(ctx context.Context, groupID UUID) (int64, error)
	countByAgentFunc     func(ctx context.Context, agentID UUID) (int64, error)
	authScopeFunc        func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockServiceRepository) Create(ctx context.Context, service *Service) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, service)
	}
	service.ID = NewUUID()
	return nil
}

func (m *MockServiceRepository) Update(ctx context.Context, service *Service) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, service)
	}
	return nil
}

func (m *MockServiceRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockServiceRepository) FindByID(ctx context.Context, id UUID) (*Service, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("service not found")
}

func (m *MockServiceRepository) FindByExternalID(ctx context.Context, agentID UUID, externalID string) (*Service, error) {
	if m.findByExternalIDFunc != nil {
		return m.findByExternalIDFunc(ctx, agentID, externalID)
	}
	return nil, NewNotFoundErrorf("service not found")
}

func (m *MockServiceRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Service], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[Service]{
		Items:       []Service{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockServiceRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockServiceRepository) CountByGroup(ctx context.Context, groupID UUID) (int64, error) {
	if m.countByGroupFunc != nil {
		return m.countByGroupFunc(ctx, groupID)
	}
	return 0, nil
}

func (m *MockServiceRepository) CountByAgent(ctx context.Context, agentID UUID) (int64, error) {
	if m.countByAgentFunc != nil {
		return m.countByAgentFunc(ctx, agentID)
	}
	return 0, nil
}

func (m *MockServiceRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// Save updates an existing service
func (m *MockServiceRepository) Save(ctx context.Context, service *Service) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, service)
	}
	return nil
}

// MockJobRepository implements the JobRepository interface for testing
type MockJobRepository struct {
	createFunc                 func(ctx context.Context, job *Job) error
	updateFunc                 func(ctx context.Context, job *Job) error
	saveFunc                   func(ctx context.Context, job *Job) error
	deleteFunc                 func(ctx context.Context, id UUID) error
	findByIDFunc               func(ctx context.Context, id UUID) (*Job, error)
	listFunc                   func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Job], error)
	existsFunc                 func(ctx context.Context, id UUID) (bool, error)
	getPendingJobsForAgentFunc func(ctx context.Context, agentID UUID, limit int) ([]*Job, error)
	getTimeOutJobsFunc         func(ctx context.Context, timeout time.Duration) ([]*Job, error)
	deleteOldCompletedJobsFunc func(ctx context.Context, olderThan time.Duration) (int, error)
	authScopeFunc              func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockJobRepository) Create(ctx context.Context, job *Job) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, job)
	}
	return nil
}

func (m *MockJobRepository) Update(ctx context.Context, job *Job) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, job)
	}
	return nil
}

func (m *MockJobRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockJobRepository) FindByID(ctx context.Context, id UUID) (*Job, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("job not found")
}

func (m *MockJobRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Job], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[Job]{
		Items:       []Job{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockJobRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockJobRepository) GetPendingJobsForAgent(ctx context.Context, agentID UUID, limit int) ([]*Job, error) {
	if m.getPendingJobsForAgentFunc != nil {
		return m.getPendingJobsForAgentFunc(ctx, agentID, limit)
	}
	return []*Job{}, nil
}

func (m *MockJobRepository) GetTimeOutJobs(ctx context.Context, timeout time.Duration) ([]*Job, error) {
	if m.getTimeOutJobsFunc != nil {
		return m.getTimeOutJobsFunc(ctx, timeout)
	}
	return []*Job{}, nil
}

// DeleteOldCompletedJobs removes completed or failed jobs older than the specified interval
func (m *MockJobRepository) DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	if m.deleteOldCompletedJobsFunc != nil {
		return m.deleteOldCompletedJobsFunc(ctx, olderThan)
	}
	return 0, nil
}

func (m *MockJobRepository) Save(ctx context.Context, job *Job) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, job)
	}
	return nil
}

func (m *MockJobRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// MockAuditEntryRepository implements the AuditEntryRepository interface for testing
type MockAuditEntryRepository struct {
	createFunc    func(ctx context.Context, auditEntry *AuditEntry) error
	listFunc      func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[AuditEntry], error)
	authScopeFunc func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockAuditEntryRepository) Create(ctx context.Context, auditEntry *AuditEntry) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, auditEntry)
	}
	return nil
}

func (m *MockAuditEntryRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[AuditEntry], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[AuditEntry]{
		Items:       []AuditEntry{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockAuditEntryRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// MockMetricTypeRepository implements the MetricTypeRepository interface for testing
type MockMetricTypeRepository struct {
	createFunc     func(ctx context.Context, metricType *MetricType) error
	updateFunc     func(ctx context.Context, metricType *MetricType) error
	deleteFunc     func(ctx context.Context, id UUID) error
	findByIDFunc   func(ctx context.Context, id UUID) (*MetricType, error)
	findByNameFunc func(ctx context.Context, name string) (*MetricType, error)
	listFunc       func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[MetricType], error)
	existsFunc     func(ctx context.Context, id UUID) (bool, error)
	countFunc      func(ctx context.Context) (int64, error)
	authScopeFunc  func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockMetricTypeRepository) Create(ctx context.Context, metricType *MetricType) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, metricType)
	}
	return nil
}

func (m *MockMetricTypeRepository) Update(ctx context.Context, metricType *MetricType) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, metricType)
	}
	return nil
}

func (m *MockMetricTypeRepository) Delete(ctx context.Context, id UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockMetricTypeRepository) FindByID(ctx context.Context, id UUID) (*MetricType, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, NewNotFoundErrorf("metric type not found")
}

func (m *MockMetricTypeRepository) FindByName(ctx context.Context, name string) (*MetricType, error) {
	if m.findByNameFunc != nil {
		return m.findByNameFunc(ctx, name)
	}
	return nil, NewNotFoundErrorf("metric type not found")
}

func (m *MockMetricTypeRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[MetricType], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[MetricType]{
		Items:       []MetricType{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockMetricTypeRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockMetricTypeRepository) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

func (m *MockMetricTypeRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}

// Save updates an existing metric type
func (m *MockMetricTypeRepository) Save(ctx context.Context, metricType *MetricType) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, metricType)
	}
	return nil
}

// MockMetricEntryRepository implements the MetricEntryRepository interface for testing
type MockMetricEntryRepository struct {
	createFunc            func(ctx context.Context, metricEntry *MetricEntry) error
	listFunc              func(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[MetricEntry], error)
	existsFunc            func(ctx context.Context, id UUID) (bool, error)
	countByMetricTypeFunc func(ctx context.Context, typeID UUID) (int64, error)
	authScopeFunc         func(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func (m *MockMetricEntryRepository) Create(ctx context.Context, metricEntry *MetricEntry) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, metricEntry)
	}
	return nil
}

func (m *MockMetricEntryRepository) List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[MetricEntry], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authIdentityScope, req)
	}
	return &PageResponse[MetricEntry]{
		Items:       []MetricEntry{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *MockMetricEntryRepository) Exists(ctx context.Context, id UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockMetricEntryRepository) CountByMetricType(ctx context.Context, typeID UUID) (int64, error) {
	if m.countByMetricTypeFunc != nil {
		return m.countByMetricTypeFunc(ctx, typeID)
	}
	return 0, nil
}

func (m *MockMetricEntryRepository) AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &EmptyAuthTargetScope, nil
}
