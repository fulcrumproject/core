package api

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
)

// mockEventSubscriptionCommander is a custom mock for EventSubscriptionCommander
type mockEventSubscriptionCommander struct {
	updateProgressFunc    func(ctx context.Context, params domain.UpdateProgressParams) (*domain.EventSubscription, error)
	acquireLeaseFunc      func(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error)
	renewLeaseFunc        func(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error)
	releaseLeaseFunc      func(ctx context.Context, params domain.ReleaseLeaseParams) (*domain.EventSubscription, error)
	acknowledgeEventsFunc func(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error)
	setActiveFunc         func(ctx context.Context, params domain.SetActiveParams) (*domain.EventSubscription, error)
	deleteFunc            func(ctx context.Context, subscriberID string) error
}

func (m *mockEventSubscriptionCommander) UpdateProgress(ctx context.Context, params domain.UpdateProgressParams) (*domain.EventSubscription, error) {
	if m.updateProgressFunc != nil {
		return m.updateProgressFunc(ctx, params)
	}
	return nil, fmt.Errorf("updateProgress not mocked")
}

func (m *mockEventSubscriptionCommander) AcquireLease(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error) {
	if m.acquireLeaseFunc != nil {
		return m.acquireLeaseFunc(ctx, params)
	}
	return nil, fmt.Errorf("acquireLease not mocked")
}

func (m *mockEventSubscriptionCommander) RenewLease(ctx context.Context, params domain.LeaseParams) (*domain.EventSubscription, error) {
	if m.renewLeaseFunc != nil {
		return m.renewLeaseFunc(ctx, params)
	}
	return nil, fmt.Errorf("renewLease not mocked")
}

func (m *mockEventSubscriptionCommander) ReleaseLease(ctx context.Context, params domain.ReleaseLeaseParams) (*domain.EventSubscription, error) {
	if m.releaseLeaseFunc != nil {
		return m.releaseLeaseFunc(ctx, params)
	}
	return nil, fmt.Errorf("releaseLease not mocked")
}

func (m *mockEventSubscriptionCommander) AcknowledgeEvents(ctx context.Context, params domain.AcknowledgeEventsParams) (*domain.EventSubscription, error) {
	if m.acknowledgeEventsFunc != nil {
		return m.acknowledgeEventsFunc(ctx, params)
	}
	return nil, fmt.Errorf("acknowledgeEvents not mocked")
}

func (m *mockEventSubscriptionCommander) SetActive(ctx context.Context, params domain.SetActiveParams) (*domain.EventSubscription, error) {
	if m.setActiveFunc != nil {
		return m.setActiveFunc(ctx, params)
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
	createFunc       func(ctx context.Context, params domain.CreateAgentParams) (*domain.Agent, error)
	updateFunc       func(ctx context.Context, params domain.UpdateAgentParams) (*domain.Agent, error)
	deleteFunc       func(ctx context.Context, id properties.UUID) error
	updateStatusFunc func(ctx context.Context, params domain.UpdateAgentStatusParams) (*domain.Agent, error)
}

func (m *mockAgentCommander) Create(ctx context.Context, params domain.CreateAgentParams) (*domain.Agent, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockAgentCommander) Update(ctx context.Context, params domain.UpdateAgentParams) (*domain.Agent, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockAgentCommander) Delete(ctx context.Context, id properties.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return fmt.Errorf("delete not mocked")
}

func (m *mockAgentCommander) UpdateStatus(ctx context.Context, params domain.UpdateAgentStatusParams) (*domain.Agent, error) {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, params)
	}
	return nil, fmt.Errorf("update status not mocked")
}

type mockParticipantCommander struct {
	createFunc func(ctx context.Context, params domain.CreateParticipantParams) (*domain.Participant, error)
	updateFunc func(ctx context.Context, params domain.UpdateParticipantParams) (*domain.Participant, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockParticipantCommander) Create(ctx context.Context, params domain.CreateParticipantParams) (*domain.Participant, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, nil
}

func (m *mockParticipantCommander) Update(ctx context.Context, params domain.UpdateParticipantParams) (*domain.Participant, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
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
	claimFunc       func(ctx context.Context, jobID properties.UUID) error
	completeFunc    func(ctx context.Context, params domain.CompleteJobParams) error
	failFunc        func(ctx context.Context, params domain.FailJobParams) error
	unsupportedFunc func(ctx context.Context, params domain.UnsupportedJobParams) error
}

func (m *mockJobCommander) Claim(ctx context.Context, jobID properties.UUID) error {
	if m.claimFunc != nil {
		return m.claimFunc(ctx, jobID)
	}
	return nil
}

func (m *mockJobCommander) Complete(ctx context.Context, params domain.CompleteJobParams) error {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, params)
	}
	return nil
}

func (m *mockJobCommander) Fail(ctx context.Context, params domain.FailJobParams) error {
	if m.failFunc != nil {
		return m.failFunc(ctx, params)
	}
	return nil
}

func (m *mockJobCommander) Unsupported(ctx context.Context, params domain.UnsupportedJobParams) error {
	if m.unsupportedFunc != nil {
		return m.unsupportedFunc(ctx, params)
	}
	return nil
}

// MockMetricEntryCommander mocks the MetricEntryCommander interface
type mockMetricEntryCommander struct {
	createFunc               func(ctx context.Context, params domain.CreateMetricEntryParams) (*domain.MetricEntry, error)
	createWithExternalIDFunc func(ctx context.Context, params domain.CreateMetricEntryWithExternalIDParams) (*domain.MetricEntry, error)
}

func (m *mockMetricEntryCommander) Create(ctx context.Context, params domain.CreateMetricEntryParams) (*domain.MetricEntry, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricEntryCommander) CreateWithExternalID(ctx context.Context, params domain.CreateMetricEntryWithExternalIDParams) (*domain.MetricEntry, error) {
	if m.createWithExternalIDFunc != nil {
		return m.createWithExternalIDFunc(ctx, params)
	}
	return nil, fmt.Errorf("createWithExternalID not mocked")
}

// mockMetricTypeCommander is a custom mock for MetricTypeCommander
type mockMetricTypeCommander struct {
	createFunc func(ctx context.Context, params domain.CreateMetricTypeParams) (*domain.MetricType, error)
	updateFunc func(ctx context.Context, params domain.UpdateMetricTypeParams) (*domain.MetricType, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockMetricTypeCommander) Create(ctx context.Context, params domain.CreateMetricTypeParams) (*domain.MetricType, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockMetricTypeCommander) Update(ctx context.Context, params domain.UpdateMetricTypeParams) (*domain.MetricType, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
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
	createFunc func(ctx context.Context, params domain.CreateServiceGroupParams) (*domain.ServiceGroup, error)
	updateFunc func(ctx context.Context, params domain.UpdateServiceGroupParams) (*domain.ServiceGroup, error)
	deleteFunc func(ctx context.Context, id properties.UUID) error
}

func (m *mockServiceGroupCommander) Create(ctx context.Context, params domain.CreateServiceGroupParams) (*domain.ServiceGroup, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceGroupCommander) Update(ctx context.Context, params domain.UpdateServiceGroupParams) (*domain.ServiceGroup, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
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
	createFunc                     func(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error)
	createWithTagsFunc             func(ctx context.Context, params domain.CreateServiceWithTagsParams) (*domain.Service, error)
	updateFunc                     func(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error)
	doActionFunc                   func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error)
	retryFunc                      func(ctx context.Context, id properties.UUID) (*domain.Service, error)
	failTimeoutServicesAndJobsFunc func(ctx context.Context, timeout time.Duration) (int, error)
}

func (m *mockServiceCommander) Create(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceCommander) CreateWithTags(ctx context.Context, params domain.CreateServiceWithTagsParams) (*domain.Service, error) {
	if m.createWithTagsFunc != nil {
		return m.createWithTagsFunc(ctx, params)
	}
	return nil, fmt.Errorf("createWithTags not mocked")
}

func (m *mockServiceCommander) Update(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceCommander) DoAction(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
	if m.doActionFunc != nil {
		return m.doActionFunc(ctx, params)
	}
	return nil, fmt.Errorf("doAction not mocked")
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
	createFunc     func(ctx context.Context, params domain.CreateTokenParams) (*domain.Token, error)
	updateFunc     func(ctx context.Context, params domain.UpdateTokenParams) (*domain.Token, error)
	deleteFunc     func(ctx context.Context, id properties.UUID) error
	regenerateFunc func(ctx context.Context, id properties.UUID) (*domain.Token, error)
}

func (m *mockTokenCommander) Create(ctx context.Context, params domain.CreateTokenParams) (*domain.Token, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockTokenCommander) Update(ctx context.Context, params domain.UpdateTokenParams) (*domain.Token, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
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
