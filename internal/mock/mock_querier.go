package mock

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

// AgentTypeQuerier mock
type AgentTypeQuerier struct{}

var _ domain.AgentTypeQuerier = (*AgentTypeQuerier)(nil)

func (m *AgentTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.AgentType, error) {
	panic("not implemented")
}

func (m *AgentTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *AgentTypeQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
	panic("not implemented")
}

func (m *AgentTypeQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *AgentTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// AgentQuerier mock
type AgentQuerier struct{}

var _ domain.AgentQuerier = (*AgentQuerier)(nil)

func (m *AgentQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *AgentQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *AgentQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error) {
	panic("not implemented")
}

func (m *AgentQuerier) CountByProvider(ctx context.Context, providerID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *AgentQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// BrokerQuerier mock
type BrokerQuerier struct{}

var _ domain.BrokerQuerier = (*BrokerQuerier)(nil)

func (m *BrokerQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Broker, error) {
	panic("not implemented")
}

func (m *BrokerQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *BrokerQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Broker], error) {
	panic("not implemented")
}

func (m *BrokerQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *BrokerQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// JobQuerier mock
type JobQuerier struct{}

var _ domain.JobQuerier = (*JobQuerier)(nil)

func (m *JobQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Job, error) {
	panic("not implemented")
}

func (m *JobQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *JobQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
	panic("not implemented")
}

func (m *JobQuerier) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *JobQuerier) GetTimeOutJobs(ctx context.Context, timeout time.Duration) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *JobQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// AuditEntryQuerier mock
type AuditEntryQuerier struct{}

var _ domain.AuditEntryQuerier = (*AuditEntryQuerier)(nil)

func (m *AuditEntryQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
	panic("not implemented")
}

func (m *AuditEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MetricEntryQuerier mock
type MetricEntryQuerier struct{}

var _ domain.MetricEntryQuerier = (*MetricEntryQuerier)(nil)

func (m *MetricEntryQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MetricEntryQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
	panic("not implemented")
}

func (m *MetricEntryQuerier) CountByMetricType(ctx context.Context, metricTypeID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MetricEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// ServiceQuerier mock
type ServiceQuerier struct{}

var _ domain.ServiceQuerier = (*ServiceQuerier)(nil)

func (m *ServiceQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	panic("not implemented")
}

func (m *ServiceQuerier) FindByExternalID(ctx context.Context, agentID domain.UUID, externalID string) (*domain.Service, error) {
	panic("not implemented")
}

func (m *ServiceQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *ServiceQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
	panic("not implemented")
}

func (m *ServiceQuerier) CountByAgent(ctx context.Context, agentID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *ServiceQuerier) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *ServiceQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// ProviderQuerier mock
type ProviderQuerier struct{}

var _ domain.ProviderQuerier = (*ProviderQuerier)(nil)

func (m *ProviderQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Provider, error) {
	panic("not implemented")
}

func (m *ProviderQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *ProviderQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Provider], error) {
	panic("not implemented")
}

func (m *ProviderQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// TokenQuerier mock
type TokenQuerier struct{}

var _ domain.TokenQuerier = (*TokenQuerier)(nil)

func (m *TokenQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	panic("not implemented")
}

func (m *TokenQuerier) FindByValue(ctx context.Context, value string) (*domain.Token, error) {
	panic("not implemented")
}

func (m *TokenQuerier) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	panic("not implemented")
}

func (m *TokenQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
	panic("not implemented")
}

func (m *TokenQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// MetricTypeQuerier mock
type MetricTypeQuerier struct{}

var _ domain.MetricTypeQuerier = (*MetricTypeQuerier)(nil)

func (m *MetricTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MetricTypeQuerier) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MetricTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MetricTypeQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MetricTypeQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error) {
	panic("not implemented")
}

func (m *MetricTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// ServiceGroupQuerier mock
type ServiceGroupQuerier struct{}

var _ domain.ServiceGroupQuerier = (*ServiceGroupQuerier)(nil)

func (m *ServiceGroupQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *ServiceGroupQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *ServiceGroupQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error) {
	panic("not implemented")
}

func (m *ServiceGroupQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

// ServiceTypeQuerier mock
type ServiceTypeQuerier struct{}

var _ domain.ServiceTypeQuerier = (*ServiceTypeQuerier)(nil)

func (m *ServiceTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
	panic("not implemented")
}

func (m *ServiceTypeQuerier) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *ServiceTypeQuerier) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
	panic("not implemented")
}

func (m *ServiceTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}
