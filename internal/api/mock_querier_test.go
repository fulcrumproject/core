package api

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

// MockAgentTypeQuerier mock
type MockAgentTypeQuerier struct{}

var _ domain.AgentTypeQuerier = (*MockAgentTypeQuerier)(nil)

func (m *MockAgentTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.AgentType, error) {
	panic("not implemented")
}

func (m *MockAgentTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockAgentTypeQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
	panic("not implemented")
}

func (m *MockAgentTypeQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockAgentTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockAgentQuerier mock
type MockAgentQuerier struct{}

var _ domain.AgentQuerier = (*MockAgentQuerier)(nil)

func (m *MockAgentQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockAgentQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error) {
	panic("not implemented")
}

func (m *MockAgentQuerier) CountByProvider(ctx context.Context, providerID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockAgentQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockBrokerQuerier mock
type MockBrokerQuerier struct{}

var _ domain.BrokerQuerier = (*MockBrokerQuerier)(nil)

func (m *MockBrokerQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Broker, error) {
	panic("not implemented")
}

func (m *MockBrokerQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockBrokerQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Broker], error) {
	panic("not implemented")
}

func (m *MockBrokerQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockBrokerQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockJobQuerier mock
type MockJobQuerier struct{}

var _ domain.JobQuerier = (*MockJobQuerier)(nil)

func (m *MockJobQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockJobQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
	panic("not implemented")
}

func (m *MockJobQuerier) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobQuerier) GetTimeOutJobs(ctx context.Context, timeout time.Duration) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockAuditEntryQuerier mock
type MockAuditEntryQuerier struct{}

var _ domain.AuditEntryQuerier = (*MockAuditEntryQuerier)(nil)

func (m *MockAuditEntryQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
	panic("not implemented")
}

func (m *MockAuditEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockMetricEntryQuerier mock
type MockMetricEntryQuerier struct{}

var _ domain.MetricEntryQuerier = (*MockMetricEntryQuerier)(nil)

func (m *MockMetricEntryQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockMetricEntryQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
	panic("not implemented")
}

func (m *MockMetricEntryQuerier) CountByMetricType(ctx context.Context, metricTypeID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockMetricEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockServiceQuerier mock
type MockServiceQuerier struct{}

var _ domain.ServiceQuerier = (*MockServiceQuerier)(nil)

func (m *MockServiceQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceQuerier) FindByExternalID(ctx context.Context, agentID domain.UUID, externalID string) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockServiceQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
	panic("not implemented")
}

func (m *MockServiceQuerier) CountByAgent(ctx context.Context, agentID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceQuerier) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockProviderQuerier mock
type MockProviderQuerier struct{}

var _ domain.ProviderQuerier = (*MockProviderQuerier)(nil)

func (m *MockProviderQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Provider, error) {
	panic("not implemented")
}

func (m *MockProviderQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockProviderQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Provider], error) {
	panic("not implemented")
}

func (m *MockProviderQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockTokenQuerier mock
type MockTokenQuerier struct{}

var _ domain.TokenQuerier = (*MockTokenQuerier)(nil)

func (m *MockTokenQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenQuerier) FindByValue(ctx context.Context, value string) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenQuerier) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
	panic("not implemented")
}

func (m *MockTokenQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockMetricTypeQuerier mock
type MockMetricTypeQuerier struct{}

var _ domain.MetricTypeQuerier = (*MockMetricTypeQuerier)(nil)

func (m *MockMetricTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MockMetricTypeQuerier) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MockMetricTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockMetricTypeQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockMetricTypeQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error) {
	panic("not implemented")
}

func (m *MockMetricTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockServiceGroupQuerier mock
type MockServiceGroupQuerier struct{}

var _ domain.ServiceGroupQuerier = (*MockServiceGroupQuerier)(nil)

func (m *MockServiceGroupQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *MockServiceGroupQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceGroupQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error) {
	panic("not implemented")
}

func (m *MockServiceGroupQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MockServiceTypeQuerier mock
type MockServiceTypeQuerier struct{}

var _ domain.ServiceTypeQuerier = (*MockServiceTypeQuerier)(nil)

func (m *MockServiceTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
	panic("not implemented")
}

func (m *MockServiceTypeQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceTypeQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
	panic("not implemented")
}

func (m *MockServiceTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}
