package mock

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
)

// Store provides a simple implementation of the Store interface for testing
// All repository methods return nil (no implementation)
type Store struct {
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
var _ domain.Store = (*Store)(nil)

func (m *Store) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return fn(m)
}

func (m *Store) BrokerRepo() domain.BrokerRepository {
	return m.brokerRepo
}

func (m *Store) TokenRepo() domain.TokenRepository {
	return m.tokenRepo
}

func (m *Store) AgentTypeRepo() domain.AgentTypeRepository {
	return m.agentTypeRepo
}

func (m *Store) AgentRepo() domain.AgentRepository {
	return m.agentRepo
}

func (m *Store) ProviderRepo() domain.ProviderRepository {
	return m.providerRepo
}

func (m *Store) ServiceTypeRepo() domain.ServiceTypeRepository {
	return m.serviceTypeRepo
}

func (m *Store) ServiceGroupRepo() domain.ServiceGroupRepository {
	return m.serviceGroupRepo
}

func (m *Store) ServiceRepo() domain.ServiceRepository {
	return m.serviceRepo
}

func (m *Store) JobRepo() domain.JobRepository {
	return m.jobRepo
}

func (m *Store) AuditEntryRepo() domain.AuditEntryRepository {
	return m.auditEntryRepo
}

func (m *Store) MetricTypeRepo() domain.MetricTypeRepository {
	return m.metricTypeRepo
}

func (m *Store) MetricEntryRepo() domain.MetricEntryRepository {
	return m.metricEntryRepo
}

// NewMockStore creates and initializes a new mock store
func NewMockStore() *Store {
	return &Store{
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
