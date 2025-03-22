package api

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

// MockAgentCommander mock
type MockAgentCommander struct{}

var _ domain.AgentCommander = (*MockAgentCommander)(nil)

func (m *MockAgentCommander) Create(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentCommander) Update(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, state *domain.AgentState) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

func (m *MockAgentCommander) UpdateState(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error) {
	panic("not implemented")
}

// MockAuditEntryCommander mock
type MockAuditEntryCommander struct{}

var _ domain.AuditEntryCommander = (*MockAuditEntryCommander)(nil)

func (m *MockAuditEntryCommander) Create(
	ctx context.Context,
	authorityType domain.AuthorityType,
	authorityID string,
	eventType domain.EventType,
	properties domain.JSON,
	entityID, providerID, agentID, brokerID *domain.UUID,
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

func (m *MockAuditEntryCommander) CreateWithDiff(
	ctx context.Context,
	authorityType domain.AuthorityType,
	authorityID string,
	eventType domain.EventType,
	entityID, providerID, agentID, brokerID *domain.UUID,
	beforeEntity, afterEntity interface{},
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

func (m *MockAuditEntryCommander) CreateCtx(
	ctx context.Context,
	eventType domain.EventType,
	properties domain.JSON,
	entityID, providerID, agentID, brokerID *domain.UUID,
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

func (m *MockAuditEntryCommander) CreateCtxWithDiff(
	ctx context.Context,
	eventType domain.EventType,
	entityID, providerID, agentID, brokerID *domain.UUID,
	beforeEntity, afterEntity interface{},
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

// MockBrokerCommander mock
type MockBrokerCommander struct{}

var _ domain.BrokerCommander = (*MockBrokerCommander)(nil)

func (m *MockBrokerCommander) Create(ctx context.Context, name string) (*domain.Broker, error) {
	panic("not implemented")
}

func (m *MockBrokerCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.Broker, error) {
	panic("not implemented")
}

func (m *MockBrokerCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

// MockJobCommander mock
type MockJobCommander struct{}

var _ domain.JobCommander = (*MockJobCommander)(nil)

func (m *MockJobCommander) Claim(ctx context.Context, jobID domain.UUID) error {
	panic("not implemented")
}

func (m *MockJobCommander) Complete(ctx context.Context, jobID domain.UUID, resources *domain.JSON, externalID *string) error {
	panic("not implemented")
}

func (m *MockJobCommander) Fail(ctx context.Context, jobID domain.UUID, errorMessage string) error {
	panic("not implemented")
}

// MockServiceCommander mock
type MockServiceCommander struct{}

var _ domain.ServiceCommander = (*MockServiceCommander)(nil)

func (m *MockServiceCommander) Create(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceCommander) Update(ctx context.Context, id domain.UUID, name *string, props *domain.JSON) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceCommander) Transition(ctx context.Context, id domain.UUID, target domain.ServiceState) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceCommander) Retry(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	panic("not implemented")
}

func (m *MockServiceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	panic("not implemented")
}

// MockTokenCommander mock
type MockTokenCommander struct{}

var _ domain.TokenCommander = (*MockTokenCommander)(nil)

func (m *MockTokenCommander) Create(ctx context.Context, name string, role domain.AuthRole, expireAt time.Time, scopeID *domain.UUID) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenCommander) Update(ctx context.Context, id domain.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
	panic("not implemented")
}

func (m *MockTokenCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

func (m *MockTokenCommander) Regenerate(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	panic("not implemented")
}

// MockProviderCommander mock
type MockProviderCommander struct{}

var _ domain.ProviderCommander = (*MockProviderCommander)(nil)

func (m *MockProviderCommander) Create(ctx context.Context, name string, state domain.ProviderState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Provider, error) {
	panic("not implemented")
}

func (m *MockProviderCommander) Update(ctx context.Context, id domain.UUID, name *string, state *domain.ProviderState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Provider, error) {
	panic("not implemented")
}

func (m *MockProviderCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

// MockMetricEntryCommander mock
type MockMetricEntryCommander struct{}

var _ domain.MetricEntryCommander = (*MockMetricEntryCommander)(nil)

func (m *MockMetricEntryCommander) Create(ctx context.Context, typeName string, agentID domain.UUID, serviceID domain.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
	panic("not implemented")
}

func (m *MockMetricEntryCommander) CreateWithExternalID(ctx context.Context, typeName string, agentID domain.UUID, externalID string, resourceID string, value float64) (*domain.MetricEntry, error) {
	panic("not implemented")
}

// MockServiceGroupCommander mock
type MockServiceGroupCommander struct{}

var _ domain.ServiceGroupCommander = (*MockServiceGroupCommander)(nil)

func (m *MockServiceGroupCommander) Create(ctx context.Context, name string, brokerID domain.UUID) (*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *MockServiceGroupCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *MockServiceGroupCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

// MockMetricTypeCommander mock
type MockMetricTypeCommander struct{}

var _ domain.MetricTypeCommander = (*MockMetricTypeCommander)(nil)

func (m *MockMetricTypeCommander) Create(ctx context.Context, name string, kind domain.MetricEntityType) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MockMetricTypeCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MockMetricTypeCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}
