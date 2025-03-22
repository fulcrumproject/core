package api

import (
	"context"
	"fmt"

	"fulcrumproject.org/core/internal/domain"
)

// mockAgentCommander is a custom mock for AgentCommander
type mockAgentCommander struct {
	createFunc      func(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error)
	updateFunc      func(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, state *domain.AgentState) (*domain.Agent, error)
	deleteFunc      func(ctx context.Context, id domain.UUID) error
	updateStateFunc func(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error)
}

func (m *mockAgentCommander) Create(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, countryCode, attributes, providerID, agentTypeID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAgentCommander) Update(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, state *domain.AgentState) (*domain.Agent, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, countryCode, attributes, state)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockAgentCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockAgentCommander) UpdateState(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error) {
	if m.updateStateFunc != nil {
		return m.updateStateFunc(ctx, id, state)
	}
	return nil, fmt.Errorf("update state not mocked")
}

// mockAuditEntryCommander is a custom mock for AuditEntryCommander
type mockAuditEntryCommander struct {
	createFunc            func(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error)
	createWithDiffFunc    func(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error)
	createCtxFunc         func(ctx context.Context, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error)
	createCtxWithDiffFunc func(ctx context.Context, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error)
}

func (m *mockAuditEntryCommander) Create(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, authorityType, authorityID, eventType, properties, entityID, providerID, agentID, brokerID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAuditEntryCommander) CreateWithDiff(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error) {
	if m.createWithDiffFunc != nil {
		return m.createWithDiffFunc(ctx, authorityType, authorityID, eventType, entityID, providerID, agentID, brokerID, beforeEntity, afterEntity)
	}
	return nil, fmt.Errorf("createWithDiff not mocked")
}

func (m *mockAuditEntryCommander) CreateCtx(ctx context.Context, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, brokerID *domain.UUID) (*domain.AuditEntry, error) {
	if m.createCtxFunc != nil {
		return m.createCtxFunc(ctx, eventType, properties, entityID, providerID, agentID, brokerID)
	}
	return nil, fmt.Errorf("createCtx not mocked")
}

func (m *mockAuditEntryCommander) CreateCtxWithDiff(ctx context.Context, eventType domain.EventType, entityID, providerID, agentID, brokerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error) {
	if m.createCtxWithDiffFunc != nil {
		return m.createCtxWithDiffFunc(ctx, eventType, entityID, providerID, agentID, brokerID, beforeEntity, afterEntity)
	}
	return nil, fmt.Errorf("createCtxWithDiff not mocked")
}

// mockBrokerCommander is a custom mock for BrokerCommander
type mockBrokerCommander struct {
	createFunc func(ctx context.Context, name string) (*domain.Broker, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string) (*domain.Broker, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockBrokerCommander) Create(ctx context.Context, name string) (*domain.Broker, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockBrokerCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.Broker, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockBrokerCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockJobCommander is a custom mock for JobCommander
type mockJobCommander struct {
	claimFunc    func(ctx context.Context, jobID domain.UUID) error
	completeFunc func(ctx context.Context, jobID domain.UUID, resources *domain.JSON, externalID *string) error
	failFunc     func(ctx context.Context, jobID domain.UUID, errorMessage string) error
}

func (m *mockJobCommander) Claim(ctx context.Context, jobID domain.UUID) error {
	if m.claimFunc != nil {
		return m.claimFunc(ctx, jobID)
	}
	return nil
}

func (m *mockJobCommander) Complete(ctx context.Context, jobID domain.UUID, resources *domain.JSON, externalID *string) error {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, jobID, resources, externalID)
	}
	return nil
}

func (m *mockJobCommander) Fail(ctx context.Context, jobID domain.UUID, errorMessage string) error {
	if m.failFunc != nil {
		return m.failFunc(ctx, jobID, errorMessage)
	}
	return nil
}

// MockMetricEntryCommander mocks the MetricEntryCommander interface
type mockMetricEntryCommander struct {
	createFunc               func(ctx context.Context, typeName string, agentID domain.UUID, serviceID domain.UUID, resourceID string, value float64) (*domain.MetricEntry, error)
	createWithExternalIDFunc func(ctx context.Context, typeName string, agentID domain.UUID, externalID string, resourceID string, value float64) (*domain.MetricEntry, error)
}

func (m *mockMetricEntryCommander) Create(ctx context.Context, typeName string, agentID domain.UUID, serviceID domain.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, typeName, agentID, serviceID, resourceID, value)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricEntryCommander) CreateWithExternalID(ctx context.Context, typeName string, agentID domain.UUID, externalID string, resourceID string, value float64) (*domain.MetricEntry, error) {
	if m.createWithExternalIDFunc != nil {
		return m.createWithExternalIDFunc(ctx, typeName, agentID, externalID, resourceID, value)
	}
	return nil, fmt.Errorf("createWithExternalID not mocked")
}

// mockMetricTypeCommander is a custom mock for MetricTypeCommander
type mockMetricTypeCommander struct {
	createFunc func(ctx context.Context, name string, entityType domain.MetricEntityType) (*domain.MetricType, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string) (*domain.MetricType, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockMetricTypeCommander) Create(ctx context.Context, name string, entityType domain.MetricEntityType) (*domain.MetricType, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, entityType)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricTypeCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.MetricType, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockMetricTypeCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockProviderCommander is a custom mock for ProviderCommander
type mockProviderCommander struct {
	createFunc func(ctx context.Context, name string, state domain.ProviderState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Provider, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string, state *domain.ProviderState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Provider, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockProviderCommander) Create(ctx context.Context, name string, state domain.ProviderState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Provider, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, state, countryCode, attributes)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockProviderCommander) Update(ctx context.Context, id domain.UUID, name *string, state *domain.ProviderState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Provider, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, state, countryCode, attributes)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockProviderCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockServiceGroupCommander is a custom mock for ServiceGroupCommander
type mockServiceGroupCommander struct {
	createFunc func(ctx context.Context, name string, brokerID domain.UUID) (*domain.ServiceGroup, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string) (*domain.ServiceGroup, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockServiceGroupCommander) Create(ctx context.Context, name string, brokerID domain.UUID) (*domain.ServiceGroup, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, brokerID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceGroupCommander) Update(ctx context.Context, id domain.UUID, name *string) (*domain.ServiceGroup, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceGroupCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}
