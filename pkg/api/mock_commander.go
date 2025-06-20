package api

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
)

// mockEventSubscriptionCommander is a custom mock for EventSubscriptionCommander
type mockEventSubscriptionCommander struct {
	updateProgressFunc    func(ctx context.Context, subscriberID string, lastEventSequenceProcessed int64) (*domain.EventSubscription, error)
	acquireLeaseFunc      func(ctx context.Context, subscriberID string, instanceID string, duration time.Duration) (*domain.EventSubscription, error)
	renewLeaseFunc        func(ctx context.Context, subscriberID string, instanceID string, duration time.Duration) (*domain.EventSubscription, error)
	releaseLeaseFunc      func(ctx context.Context, subscriberID string, instanceID string) (*domain.EventSubscription, error)
	acknowledgeEventsFunc func(ctx context.Context, subscriberID string, instanceID string, lastEventSequenceProcessed int64) (*domain.EventSubscription, error)
	setActiveFunc         func(ctx context.Context, subscriberID string, isActive bool) (*domain.EventSubscription, error)
	deleteFunc            func(ctx context.Context, subscriberID string) error
}

func (m *mockEventSubscriptionCommander) UpdateProgress(ctx context.Context, subscriberID string, lastEventSequenceProcessed int64) (*domain.EventSubscription, error) {
	if m.updateProgressFunc != nil {
		return m.updateProgressFunc(ctx, subscriberID, lastEventSequenceProcessed)
	}
	return nil, fmt.Errorf("updateProgress not mocked")
}

func (m *mockEventSubscriptionCommander) AcquireLease(ctx context.Context, subscriberID string, instanceID string, duration time.Duration) (*domain.EventSubscription, error) {
	if m.acquireLeaseFunc != nil {
		return m.acquireLeaseFunc(ctx, subscriberID, instanceID, duration)
	}
	return nil, fmt.Errorf("acquireLease not mocked")
}

func (m *mockEventSubscriptionCommander) RenewLease(ctx context.Context, subscriberID string, instanceID string, duration time.Duration) (*domain.EventSubscription, error) {
	if m.renewLeaseFunc != nil {
		return m.renewLeaseFunc(ctx, subscriberID, instanceID, duration)
	}
	return nil, fmt.Errorf("renewLease not mocked")
}

func (m *mockEventSubscriptionCommander) ReleaseLease(ctx context.Context, subscriberID string, instanceID string) (*domain.EventSubscription, error) {
	if m.releaseLeaseFunc != nil {
		return m.releaseLeaseFunc(ctx, subscriberID, instanceID)
	}
	return nil, fmt.Errorf("releaseLease not mocked")
}

func (m *mockEventSubscriptionCommander) AcknowledgeEvents(ctx context.Context, subscriberID string, instanceID string, lastEventSequenceProcessed int64) (*domain.EventSubscription, error) {
	if m.acknowledgeEventsFunc != nil {
		return m.acknowledgeEventsFunc(ctx, subscriberID, instanceID, lastEventSequenceProcessed)
	}
	return nil, fmt.Errorf("acknowledgeEvents not mocked")
}

func (m *mockEventSubscriptionCommander) SetActive(ctx context.Context, subscriberID string, isActive bool) (*domain.EventSubscription, error) {
	if m.setActiveFunc != nil {
		return m.setActiveFunc(ctx, subscriberID, isActive)
	}
	return nil, fmt.Errorf("setActive not mocked")
}

func (m *mockEventSubscriptionCommander) Delete(ctx context.Context, subscriberID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, subscriberID)
	}
	return fmt.Errorf("delete not mocked")
}

// mockAgentCommander is a custom mock for AgentCommander
type mockAgentCommander struct {
	createFunc       func(ctx context.Context, name string, providerID properties.UUID, agentTypeID properties.UUID, tags []string) (*domain.Agent, error)
	updateFunc       func(ctx context.Context, id properties.UUID, name *string, status *domain.AgentStatus, tags *[]string) (*domain.Agent, error)
	deleteFunc       func(ctx context.Context, id properties.UUID) error
	updateStatusFunc func(ctx context.Context, id properties.UUID, status domain.AgentStatus) (*domain.Agent, error)
}

func (m *mockAgentCommander) Create(ctx context.Context, name string, providerID properties.UUID, agentTypeID properties.UUID, tags []string) (*domain.Agent, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, providerID, agentTypeID, tags)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAgentCommander) Update(ctx context.Context, id properties.UUID, name *string, status *domain.AgentStatus, tags *[]string) (*domain.Agent, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, status, tags)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockAgentCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockAgentCommander) UpdateStatus(ctx context.Context, id properties.UUID, status domain.AgentStatus) (*domain.Agent, error) {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status)
	}
	return nil, fmt.Errorf("update status not mocked")
}

type mockParticipantCommander struct {
	createFunc func(ctx context.Context, name string, status domain.ParticipantStatus) (*domain.Participant, error)
	updateFunc func(ctx context.Context, id properties.UUID, name *string, status *domain.ParticipantStatus) (*domain.Participant, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockParticipantCommander) Create(ctx context.Context, name string, status domain.ParticipantStatus) (*domain.Participant, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, status)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Update(ctx context.Context, id properties.UUID, name *string, status *domain.ParticipantStatus) (*domain.Participant, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, status)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// mockJobCommander is a custom mock for JobCommander
type mockJobCommander struct {
	claimFunc    func(ctx context.Context, jobID properties.UUID) error
	completeFunc func(ctx context.Context, jobID properties.UUID, resources *properties.JSON, externalID *string) error
	failFunc     func(ctx context.Context, jobID properties.UUID, errorMessage string) error
}

func (m *mockJobCommander) Claim(ctx context.Context, jobID properties.UUID) error {
	if m.claimFunc != nil {
		return m.claimFunc(ctx, jobID)
	}
	return nil
}

func (m *mockJobCommander) Complete(ctx context.Context, jobID properties.UUID, resources *properties.JSON, externalID *string) error {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, jobID, resources, externalID)
	}
	return nil
}

func (m *mockJobCommander) Fail(ctx context.Context, jobID properties.UUID, errorMessage string) error {
	if m.failFunc != nil {
		return m.failFunc(ctx, jobID, errorMessage)
	}
	return nil
}

// MockMetricEntryCommander mocks the MetricEntryCommander interface
type mockMetricEntryCommander struct {
	createFunc               func(ctx context.Context, typeName string, agentID properties.UUID, serviceID properties.UUID, resourceID string, value float64) (*domain.MetricEntry, error)
	createWithExternalIDFunc func(ctx context.Context, typeName string, agentID properties.UUID, externalID string, resourceID string, value float64) (*domain.MetricEntry, error)
}

func (m *mockMetricEntryCommander) Create(ctx context.Context, typeName string, agentID properties.UUID, serviceID properties.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, typeName, agentID, serviceID, resourceID, value)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricEntryCommander) CreateWithExternalID(ctx context.Context, typeName string, agentID properties.UUID, externalID string, resourceID string, value float64) (*domain.MetricEntry, error) {
	if m.createWithExternalIDFunc != nil {
		return m.createWithExternalIDFunc(ctx, typeName, agentID, externalID, resourceID, value)
	}
	return nil, fmt.Errorf("createWithExternalID not mocked")
}

// mockMetricTypeCommander is a custom mock for MetricTypeCommander
type mockMetricTypeCommander struct {
	createFunc func(ctx context.Context, name string, entityType domain.MetricEntityType) (*domain.MetricType, error)
	updateFunc func(ctx context.Context, id properties.UUID, name *string) (*domain.MetricType, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockMetricTypeCommander) Create(ctx context.Context, name string, entityType domain.MetricEntityType) (*domain.MetricType, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, entityType)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricTypeCommander) Update(ctx context.Context, id properties.UUID, name *string) (*domain.MetricType, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockMetricTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockServiceGroupCommander is a custom mock for ServiceGroupCommander
type mockServiceGroupCommander struct {
	createFunc func(ctx context.Context, name string, consumerID properties.UUID) (*domain.ServiceGroup, error)
	updateFunc func(ctx context.Context, id properties.UUID, name *string) (*domain.ServiceGroup, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockServiceGroupCommander) Create(ctx context.Context, name string, consumerID properties.UUID) (*domain.ServiceGroup, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, consumerID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceGroupCommander) Update(ctx context.Context, id properties.UUID, name *string) (*domain.ServiceGroup, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceGroupCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

// mockServiceCommander is a custom mock for ServiceCommander
type mockServiceCommander struct {
	createFunc                     func(ctx context.Context, agentID properties.UUID, serviceTypeID properties.UUID, groupID properties.UUID, name string, properties properties.JSON) (*domain.Service, error)
	createWithTagsFunc             func(ctx context.Context, serviceTypeID properties.UUID, groupID properties.UUID, name string, properties properties.JSON, serviceTags []string) (*domain.Service, error)
	updateFunc                     func(ctx context.Context, id properties.UUID, name *string, properties *properties.JSON) (*domain.Service, error)
	transitionFunc                 func(ctx context.Context, id properties.UUID, status domain.ServiceStatus) (*domain.Service, error)
	retryFunc                      func(ctx context.Context, id properties.UUID) (*domain.Service, error)
	failTimeoutServicesAndJobsFunc func(ctx context.Context, timeout time.Duration) (int, error)
}

func (m *mockServiceCommander) Create(ctx context.Context, agentID properties.UUID, serviceTypeID properties.UUID, groupID properties.UUID, name string, properties properties.JSON) (*domain.Service, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, agentID, serviceTypeID, groupID, name, properties)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceCommander) CreateWithTags(ctx context.Context, serviceTypeID properties.UUID, groupID properties.UUID, name string, properties properties.JSON, serviceTags []string) (*domain.Service, error) {
	if m.createWithTagsFunc != nil {
		return m.createWithTagsFunc(ctx, serviceTypeID, groupID, name, properties, serviceTags)
	}
	return nil, fmt.Errorf("createWithTags not mocked")
}

func (m *mockServiceCommander) Update(ctx context.Context, id properties.UUID, name *string, properties *properties.JSON) (*domain.Service, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, properties)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceCommander) Transition(ctx context.Context, id properties.UUID, status domain.ServiceStatus) (*domain.Service, error) {
	if m.transitionFunc != nil {
		return m.transitionFunc(ctx, id, status)
	}
	return nil, fmt.Errorf("transition not mocked")
}

func (m *mockServiceCommander) Retry(ctx context.Context, id properties.UUID) (*domain.Service, error) {
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
	createFunc     func(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*domain.Token, error)
	updateFunc     func(ctx context.Context, id properties.UUID, name *string, expireAt *time.Time) (*domain.Token, error)
	deleteFunc     func(ctx context.Context, id properties.UUID) error
	regenerateFunc func(ctx context.Context, id properties.UUID) (*domain.Token, error)
}

func (m *mockTokenCommander) Create(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*domain.Token, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, role, expireAt, scopeID)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockTokenCommander) Update(ctx context.Context, id properties.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, expireAt)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockTokenCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockTokenCommander) Regenerate(ctx context.Context, id properties.UUID) (*domain.Token, error) {
	if m.regenerateFunc != nil {
		return m.regenerateFunc(ctx, id)
	}
	return nil, fmt.Errorf("regenerate not mocked")
}
