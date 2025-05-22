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
	createFunc            func(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, consumerID *domain.UUID) (*domain.AuditEntry, error)
	createWithDiffFunc    func(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, entityID, providerID, agentID, consumerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error)
	createCtxFunc         func(ctx context.Context, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, consumerID *domain.UUID) (*domain.AuditEntry, error)
	createCtxWithDiffFunc func(ctx context.Context, eventType domain.EventType, entityID, providerID, agentID, consumerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error)
}

func (m *mockAuditEntryCommander) Create(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, consumerID *domain.UUID) (*domain.AuditEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, authorityType, authorityID, eventType, properties, entityID, providerID, agentID, consumerID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAuditEntryCommander) CreateWithDiff(ctx context.Context, authorityType domain.AuthorityType, authorityID string, eventType domain.EventType, entityID, providerID, agentID, consumerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error) {
	if m.createWithDiffFunc != nil {
		return m.createWithDiffFunc(ctx, authorityType, authorityID, eventType, entityID, providerID, agentID, consumerID, beforeEntity, afterEntity)
	}
	return nil, fmt.Errorf("createWithDiff not mocked")
}

func (m *mockAuditEntryCommander) CreateCtx(ctx context.Context, eventType domain.EventType, properties domain.JSON, entityID, providerID, agentID, consumerID *domain.UUID) (*domain.AuditEntry, error) {
	if m.createCtxFunc != nil {
		return m.createCtxFunc(ctx, eventType, properties, entityID, providerID, agentID, consumerID)
	}
	return nil, fmt.Errorf("createCtx not mocked")
}

func (m *mockAuditEntryCommander) CreateCtxWithDiff(ctx context.Context, eventType domain.EventType, entityID, providerID, agentID, consumerID *domain.UUID, beforeEntity, afterEntity interface{}) (*domain.AuditEntry, error) {
	if m.createCtxWithDiffFunc != nil {
		return m.createCtxWithDiffFunc(ctx, eventType, entityID, providerID, agentID, consumerID, beforeEntity, afterEntity)
	}
	return nil, fmt.Errorf("createCtxWithDiff not mocked")
}

type mockParticipantCommander struct {
	createFunc func(ctx context.Context, name string, state domain.ParticipantState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Participant, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string, state *domain.ParticipantState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Participant, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockParticipantCommander) Create(ctx context.Context, name string, state domain.ParticipantState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Participant, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, state, countryCode, attributes)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Update(ctx context.Context, id domain.UUID, name *string, state *domain.ParticipantState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Participant, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, state, countryCode, attributes)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
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

// mockServiceGroupCommander is a custom mock for ServiceGroupCommander
type mockServiceGroupCommander struct {
	createFunc func(ctx context.Context, name string, consumerID domain.UUID) (*domain.ServiceGroup, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string) (*domain.ServiceGroup, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockServiceGroupCommander) Create(ctx context.Context, name string, consumerID domain.UUID) (*domain.ServiceGroup, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, consumerID)
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
