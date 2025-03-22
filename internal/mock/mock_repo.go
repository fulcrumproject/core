package mock

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

type MockAgentTypeRepo struct {
	baseRepo[domain.AgentType]
}

// Compile-time interface implementation checks
var _ domain.AgentTypeRepository = (*MockAgentTypeRepo)(nil)

func (m *MockAgentTypeRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockAgentTypeRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
	panic("not implemented")
}

func (m *MockAgentTypeRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockAgentTypeRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockProviderRepo struct {
	baseRepo[domain.Provider]
}

// Compile-time interface implementation checks
var _ domain.ProviderRepository = (*MockProviderRepo)(nil)

func (m *MockProviderRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockProviderRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Provider], error) {
	panic("not implemented")
}

func (m *MockProviderRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

func (m *MockProviderRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

type MockAgentRepo struct {
	baseRepo[domain.Agent]
}

// Compile-time interface implementation checks
var _ domain.AgentRepository = (*MockAgentRepo)(nil)

func (m *MockAgentRepo) GetAgentsByAgentTypeID(ctx context.Context, agentTypeID domain.UUID) ([]*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) CountByProvider(ctx context.Context, providerID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error) {
	panic("not implemented")
}

func (m *MockAgentRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockBrokerRepo struct {
	baseRepo[domain.Broker]
}

// Compile-time interface implementation checks
var _ domain.BrokerRepository = (*MockBrokerRepo)(nil)

func (m *MockBrokerRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockBrokerRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Broker], error) {
	panic("not implemented")
}

func (m *MockBrokerRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockBrokerRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockTokenRepo struct {
	baseRepo[domain.Token]
}

// Compile-time interface implementation checks
var _ domain.TokenRepository = (*MockTokenRepo)(nil)

func (m *MockTokenRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
	panic("not implemented")
}

func (m *MockTokenRepo) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenRepo) DeleteByBrokerID(ctx context.Context, brokerID domain.UUID) error {
	panic("not implemented")
}

func (m *MockTokenRepo) DeleteByProviderID(ctx context.Context, providerID domain.UUID) error {
	panic("not implemented")
}

func (m *MockTokenRepo) DeleteByAgentID(ctx context.Context, agentID domain.UUID) error {
	panic("not implemented")
}

func (m *MockTokenRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockTokenRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockTokenRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockServiceTypeRepo struct {
	baseRepo[domain.ServiceType]
}

// Compile-time interface implementation checks
var _ domain.ServiceTypeRepository = (*MockServiceTypeRepo)(nil)

func (m *MockServiceTypeRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
	panic("not implemented")
}

func (m *MockServiceTypeRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockServiceTypeRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceTypeRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockServiceGroupRepo struct {
	baseRepo[domain.ServiceGroup]
}

// Compile-time interface implementation checks
var _ domain.ServiceGroupRepository = (*MockServiceGroupRepo)(nil)

func (m *MockServiceGroupRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error) {
	panic("not implemented")
}

func (m *MockServiceGroupRepo) FindByProviderID(ctx context.Context, providerID domain.UUID) ([]*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *MockServiceGroupRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockServiceGroupRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceGroupRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockServiceRepo struct {
	baseRepo[domain.Service]
}

// Compile-time interface implementation checks
var _ domain.ServiceRepository = (*MockServiceRepo)(nil)

func (m *MockServiceRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
	panic("not implemented")
}

func (m *MockServiceRepo) CountByAgent(ctx context.Context, agentID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) FindByServiceGroupID(ctx context.Context, serviceGroupID domain.UUID) ([]*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) FindByExternalID(ctx context.Context, agentID domain.UUID, externalID string) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) CountByServiceGroup(ctx context.Context, serviceGroupID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockServiceRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockJobRepo struct {
	baseRepo[domain.Job]
}

// Compile-time interface implementation checks
var _ domain.JobRepository = (*MockJobRepo)(nil)

func (m *MockJobRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
	panic("not implemented")
}

func (m *MockJobRepo) FindByServiceID(ctx context.Context, serviceID domain.UUID) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobRepo) DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	panic("not implemented")
}

func (m *MockJobRepo) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobRepo) GetTimeOutJobs(ctx context.Context, olderThan time.Duration) ([]*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockJobRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockJobRepo) FindNextPendingJob(ctx context.Context) (*domain.Job, error) {
	panic("not implemented")
}

func (m *MockJobRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockAuditEntryRepo struct {
	baseRepo[domain.AuditEntry]
}

// Compile-time interface implementation checks
var _ domain.AuditEntryRepository = (*MockAuditEntryRepo)(nil)

func (m *MockAuditEntryRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
	panic("not implemented")
}

func (m *MockAuditEntryRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockMetricTypeRepo struct {
	baseRepo[domain.MetricType]
}

// Compile-time interface implementation checks
var _ domain.MetricTypeRepository = (*MockMetricTypeRepo)(nil)

func (m *MockMetricTypeRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockMetricTypeRepo) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MockMetricTypeRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error) {
	panic("not implemented")
}

func (m *MockMetricTypeRepo) Count(ctx context.Context) (int64, error) {
	panic("not implemented")
}

func (m *MockMetricTypeRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type MockMetricEntryRepo struct {
	baseRepo[domain.MetricEntry]
}

// Compile-time interface implementation checks
var _ domain.MetricEntryRepository = (*MockMetricEntryRepo)(nil)

func (m *MockMetricEntryRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
	panic("not implemented")
}

func (m *MockMetricEntryRepo) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	panic("not implemented")
}

func (m *MockMetricEntryRepo) CountByMetricType(ctx context.Context, metricTypeID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockMetricEntryRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	panic("not implemented")
}

type baseRepo[T any] struct{}

func (m *baseRepo[T]) List(ctx context.Context, req *domain.PageRequest) (*domain.PageResponse[T], error) {
	panic("not implemented - use the overridden version with AuthScope")
}

func (m *baseRepo[T]) Save(ctx context.Context, entity *T) error {
	panic("not implemented")
}

func (m *baseRepo[T]) Create(ctx context.Context, entity *T) error {
	panic("not implemented")
}

func (m *baseRepo[T]) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

func (m *baseRepo[T]) FindByID(ctx context.Context, id domain.UUID) (*T, error) {
	panic("not implemented")
}
