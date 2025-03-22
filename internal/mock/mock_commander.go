package mock

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

// AgentCommander mock
type AgentCommander struct{}

var _ domain.AgentCommander = (*AgentCommander)(nil)

func (m *AgentCommander) Create(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *AgentCommander) Update(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, state *domain.AgentState) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *AgentCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

func (m *AgentCommander) UpdateState(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error) {
	panic("not implemented")
}

// AuditEntryCommander mock
type AuditEntryCommander struct{}

var _ domain.AuditEntryCommander = (*AuditEntryCommander)(nil)

func (m *AuditEntryCommander) Create(
	ctx context.Context,
	authorityType domain.AuthorityType,
	authorityID string,
	eventType domain.EventType,
	properties domain.JSON,
	entityID, providerID, agentID, brokerID *domain.UUID,
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

func (m *AuditEntryCommander) CreateWithDiff(
	ctx context.Context,
	authorityType domain.AuthorityType,
	authorityID string,
	eventType domain.EventType,
	entityID, providerID, agentID, brokerID *domain.UUID,
	beforeEntity, afterEntity interface{},
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

func (m *AuditEntryCommander) CreateCtx(
	ctx context.Context,
	eventType domain.EventType,
	properties domain.JSON,
	entityID, providerID, agentID, brokerID *domain.UUID,
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

func (m *AuditEntryCommander) CreateCtxWithDiff(
	ctx context.Context,
	eventType domain.EventType,
	entityID, providerID, agentID, brokerID *domain.UUID,
	beforeEntity, afterEntity interface{},
) (*domain.AuditEntry, error) {
	panic("not implemented")
}

// BrokerCommander mock
type BrokerCommander struct{}

var _ domain.BrokerCommander = (*BrokerCommander)(nil)

func (m *BrokerCommander) Create(ctx context.Context, name string) (*domain.Broker, error) {
	panic("not implemented")
}

func (m *BrokerCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.Broker, error) {
	panic("not implemented")
}

func (m *BrokerCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

// JobCommander mock
type JobCommander struct{}

var _ domain.JobCommander = (*JobCommander)(nil)

func (m *JobCommander) Claim(ctx context.Context, jobID domain.UUID) error {
	panic("not implemented")
}

func (m *JobCommander) Complete(ctx context.Context, jobID domain.UUID, resources *domain.JSON, externalID *string) error {
	panic("not implemented")
}

func (m *JobCommander) Fail(ctx context.Context, jobID domain.UUID, errorMessage string) error {
	panic("not implemented")
}

// ServiceCommander mock
type ServiceCommander struct{}

var _ domain.ServiceCommander = (*ServiceCommander)(nil)

func (m *ServiceCommander) Create(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error) {
	panic("not implemented")
}

func (m *ServiceCommander) Update(ctx context.Context, id domain.UUID, name *string, props *domain.JSON) (*domain.Service, error) {
	panic("not implemented")
}

func (m *ServiceCommander) Transition(ctx context.Context, id domain.UUID, target domain.ServiceState) (*domain.Service, error) {
	panic("not implemented")
}

func (m *ServiceCommander) Retry(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	panic("not implemented")
}

func (m *ServiceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	panic("not implemented")
}

// TokenCommander mock
type TokenCommander struct{}

var _ domain.TokenCommander = (*TokenCommander)(nil)

func (m *TokenCommander) Create(ctx context.Context, name string, role domain.AuthRole, expireAt time.Time, scopeID *domain.UUID) (*domain.Token, error) {
	panic("not implemented")
}

func (m *TokenCommander) Update(ctx context.Context, id domain.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
	panic("not implemented")
}

func (m *TokenCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

func (m *TokenCommander) Regenerate(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	panic("not implemented")
}

// ProviderCommander mock
type ProviderCommander struct{}

var _ domain.ProviderCommander = (*ProviderCommander)(nil)

func (m *ProviderCommander) Create(ctx context.Context, name string, state domain.ProviderState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Provider, error) {
	panic("not implemented")
}

func (m *ProviderCommander) Update(ctx context.Context, id domain.UUID, name *string, state *domain.ProviderState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Provider, error) {
	panic("not implemented")
}

func (m *ProviderCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

// MetricEntryCommander mock
type MetricEntryCommander struct{}

var _ domain.MetricEntryCommander = (*MetricEntryCommander)(nil)

func (m *MetricEntryCommander) Create(ctx context.Context, typeName string, agentID domain.UUID, serviceID domain.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
	panic("not implemented")
}

func (m *MetricEntryCommander) CreateWithExternalID(ctx context.Context, typeName string, agentID domain.UUID, externalID string, resourceID string, value float64) (*domain.MetricEntry, error) {
	panic("not implemented")
}

// ServiceGroupCommander mock
type ServiceGroupCommander struct{}

var _ domain.ServiceGroupCommander = (*ServiceGroupCommander)(nil)

func (m *ServiceGroupCommander) Create(ctx context.Context, name string, brokerID domain.UUID) (*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *ServiceGroupCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.ServiceGroup, error) {
	panic("not implemented")
}

func (m *ServiceGroupCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

// MetricTypeCommander mock
type MetricTypeCommander struct{}

var _ domain.MetricTypeCommander = (*MetricTypeCommander)(nil)

func (m *MetricTypeCommander) Create(ctx context.Context, name string, kind domain.MetricEntityType) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MetricTypeCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.MetricType, error) {
	panic("not implemented")
}

func (m *MetricTypeCommander) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}
