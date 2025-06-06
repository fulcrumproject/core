package api

import (
	"context"
	"fmt"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

// mockAgentCommander is a custom mock for AgentCommander
type mockAgentCommander struct {
	createFunc       func(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error)
	updateFunc       func(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, status *domain.AgentStatus) (*domain.Agent, error)
	deleteFunc       func(ctx context.Context, id domain.UUID) error
	updateStatusFunc func(ctx context.Context, id domain.UUID, status domain.AgentStatus) (*domain.Agent, error)
}

func (m *mockAgentCommander) Create(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, countryCode, attributes, providerID, agentTypeID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAgentCommander) Update(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, status *domain.AgentStatus) (*domain.Agent, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, countryCode, attributes, status)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockAgentCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockAgentCommander) UpdateStatus(ctx context.Context, id domain.UUID, status domain.AgentStatus) (*domain.Agent, error) {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status)
	}
	return nil, fmt.Errorf("update status not mocked")
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
	createFunc func(ctx context.Context, name string, status domain.ParticipantStatus, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Participant, error)
	updateFunc func(ctx context.Context, id domain.UUID, name *string, status *domain.ParticipantStatus, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Participant, error)
	deleteFunc func(ctx context.Context, id domain.UUID) error
}

func (m *mockParticipantCommander) Create(ctx context.Context, name string, status domain.ParticipantStatus, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Participant, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, status, countryCode, attributes)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Update(ctx context.Context, id domain.UUID, name *string, status *domain.ParticipantStatus, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Participant, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, status, countryCode, attributes)
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

// mockServiceCommander is a custom mock for ServiceCommander
type mockServiceCommander struct {
	createFunc                     func(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error)
	updateFunc                     func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error)
	transitionFunc                 func(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error)
	retryFunc                      func(ctx context.Context, id domain.UUID) (*domain.Service, error)
	failTimeoutServicesAndJobsFunc func(ctx context.Context, timeout time.Duration) (int, error)
}

func (m *mockServiceCommander) Create(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, agentID, serviceTypeID, groupID, name, attributes, properties)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceCommander) Update(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, properties)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceCommander) Transition(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error) {
	if m.transitionFunc != nil {
		return m.transitionFunc(ctx, id, status)
	}
	return nil, fmt.Errorf("transition not mocked")
}

func (m *mockServiceCommander) Retry(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	if m.retryFunc != nil {
		return m.retryFunc(ctx, id)
	}
	return nil, fmt.Errorf("retry not mocked")
}

func (m *mockServiceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	if m.failTimeoutServicesAndJobsFunc != nil {
		return m.failTimeoutServicesAndJobsFunc(ctx, timeout)
	}
	return 0, fmt.Errorf("failTimeoutServicesAndJobs not mocked")
}

// mockTokenCommander is a custom mock for TokenCommander
type mockTokenCommander struct {
	createFunc     func(ctx context.Context, name string, role domain.AuthRole, expireAt *time.Time, scopeID *domain.UUID) (*domain.Token, error)
	updateFunc     func(ctx context.Context, id domain.UUID, name *string, expireAt *time.Time) (*domain.Token, error)
	deleteFunc     func(ctx context.Context, id domain.UUID) error
	regenerateFunc func(ctx context.Context, id domain.UUID) (*domain.Token, error)
}

func (m *mockTokenCommander) Create(ctx context.Context, name string, role domain.AuthRole, expireAt *time.Time, scopeID *domain.UUID) (*domain.Token, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, role, expireAt, scopeID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockTokenCommander) Update(ctx context.Context, id domain.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, expireAt)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockTokenCommander) Delete(ctx context.Context, id domain.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockTokenCommander) Regenerate(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	if m.regenerateFunc != nil {
		return m.regenerateFunc(ctx, id)
	}
	return nil, fmt.Errorf("regenerate not mocked")
}
