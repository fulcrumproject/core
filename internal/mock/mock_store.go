package mock

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
)

// MockStore provides a simple implementation of the MockStore interface for testing
// All repository methods return nil (no implementation)
type MockStore struct {
	agentTypeRepo    *MockAgentTypeRepo
	agentRepo        *MockAgentRepo
	brokerRepo       *MockBrokerRepo
	tokenRepo        *MockTokenRepo
	providerRepo     *MockProviderRepo
	serviceTypeRepo  *MockServiceTypeRepo
	serviceGroupRepo *MockServiceGroupRepo
	serviceRepo      *MockServiceRepo
	jobRepo          *MockJobRepo
	auditEntryRepo   *MockAuditEntryRepo
	metricTypeRepo   *MockMetricTypeRepo
	metricEntryRepo  *MockMetricEntryRepo
}

// Ensure MockStore implements Store
var _ domain.Store = (*MockStore)(nil)

func (m *MockStore) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return fn(m)
}

func (m *MockStore) BrokerRepo() domain.BrokerRepository {
	return m.brokerRepo
}

func (m *MockStore) TokenRepo() domain.TokenRepository {
	return m.tokenRepo
}

func (m *MockStore) AgentTypeRepo() domain.AgentTypeRepository {
	return m.agentTypeRepo
}

func (m *MockStore) AgentRepo() domain.AgentRepository {
	return m.agentRepo
}

func (m *MockStore) ProviderRepo() domain.ProviderRepository {
	return m.providerRepo
}

func (m *MockStore) ServiceTypeRepo() domain.ServiceTypeRepository {
	return m.serviceTypeRepo
}

func (m *MockStore) ServiceGroupRepo() domain.ServiceGroupRepository {
	return m.serviceGroupRepo
}

func (m *MockStore) ServiceRepo() domain.ServiceRepository {
	return m.serviceRepo
}

func (m *MockStore) JobRepo() domain.JobRepository {
	return m.jobRepo
}

func (m *MockStore) AuditEntryRepo() domain.AuditEntryRepository {
	return m.auditEntryRepo
}

func (m *MockStore) MetricTypeRepo() domain.MetricTypeRepository {
	return m.metricTypeRepo
}

func (m *MockStore) MetricEntryRepo() domain.MetricEntryRepository {
	return m.metricEntryRepo
}

// NewMockStore creates and initializes a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		agentTypeRepo:    &MockAgentTypeRepo{},
		agentRepo:        &MockAgentRepo{},
		brokerRepo:       &MockBrokerRepo{},
		tokenRepo:        &MockTokenRepo{},
		providerRepo:     &MockProviderRepo{},
		serviceTypeRepo:  &MockServiceTypeRepo{},
		serviceGroupRepo: &MockServiceGroupRepo{},
		serviceRepo:      &MockServiceRepo{},
		jobRepo:          &MockJobRepo{},
		auditEntryRepo:   &MockAuditEntryRepo{},
		metricTypeRepo:   &MockMetricTypeRepo{},
		metricEntryRepo:  &MockMetricEntryRepo{},
	}
}
