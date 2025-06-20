package api

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
)

// BaseMockQuerier provides common mock implementations for BaseEntityQuerier methods
type BaseMockQuerier[T domain.Entity] struct {
	GetFunc       func(ctx context.Context, id properties.UUID) (*T, error)
	ExistsFunc    func(ctx context.Context, id properties.UUID) (bool, error)
	ListFunc      func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[T], error)
	CountFunc     func(ctx context.Context) (int64, error)
	AuthScopeFunc func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}

// Get retrieves an entity by ID
func (m *BaseMockQuerier[T]) Get(ctx context.Context, id properties.UUID) (*T, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, domain.NewNotFoundErrorf("entity not found")
}

// Exists checks if an entity with the given ID exists
func (m *BaseMockQuerier[T]) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, id)
	}
	return true, nil
}

// List retrieves a list of entities based on the provided filters
func (m *BaseMockQuerier[T]) List(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[T], error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[T]{
		Items:       []T{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

// Count returns the number of entities
func (m *BaseMockQuerier[T]) Count(ctx context.Context) (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

// AuthScope returns the authorization scope for the entity
func (m *BaseMockQuerier[T]) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	if m.AuthScopeFunc != nil {
		return m.AuthScopeFunc(ctx, id)
	}
	return &auth.AllwaysMatchObjectScope{}, nil
}

// BaseMockRepository provides common mock implementations for BaseEntityRepository methods
type BaseMockRepository[T domain.Entity] struct {
	BaseMockQuerier[T]
	CreateFunc func(ctx context.Context, entity *T) error
	SaveFunc   func(ctx context.Context, entity *T) error
	DeleteFunc func(ctx context.Context, id properties.UUID) error
}

// Create creates a new entity
func (m *BaseMockRepository[T]) Create(ctx context.Context, entity *T) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, entity)
	}
	return nil
}

// Save updates an existing entity
func (m *BaseMockRepository[T]) Save(ctx context.Context, entity *T) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, entity)
	}
	return nil
}

// Delete removes an entity by ID
func (m *BaseMockRepository[T]) Delete(ctx context.Context, id properties.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// Ensure interface compatibility
var _ domain.AgentQuerier = (*mockAgentQuerier)(nil)

// mockAgentQuerier is a custom mock for AgentQuerier
type mockAgentQuerier struct {
	BaseMockQuerier[domain.Agent]
	countByParticipantFunc       func(ctx context.Context, participantID properties.UUID) (int64, error)
	findByServiceTypeAndTagsFunc func(ctx context.Context, serviceTypeID properties.UUID, tags []string) ([]*domain.Agent, error)
}

// CountByProvider is required by the AgentQuerier interface
func (m *mockAgentQuerier) CountByProvider(ctx context.Context, participantID properties.UUID) (int64, error) {
	if m.countByParticipantFunc != nil {
		return m.countByParticipantFunc(ctx, participantID)
	}
	return 0, nil
}

func (m *mockAgentQuerier) FindByServiceTypeAndTags(ctx context.Context, serviceTypeID properties.UUID, tags []string) ([]*domain.Agent, error) {
	if m.findByServiceTypeAndTagsFunc != nil {
		return m.findByServiceTypeAndTagsFunc(ctx, serviceTypeID, tags)
	}
	return []*domain.Agent{}, nil
}

// Ensure interface compatibility
var _ domain.ServiceTypeQuerier = (*mockServiceTypeQuerier)(nil)

// mockServiceTypeQuerier is a custom mock for ServiceTypeQuerier
type mockServiceTypeQuerier struct {
	BaseMockQuerier[domain.ServiceType]
}

// Ensure interface compatibility
var _ domain.AgentTypeQuerier = (*mockAgentTypeQuerier)(nil)

// mockAgentTypeQuerier is a custom mock for AgentTypeQuerier that allows us to set up expected values and error returns
type mockAgentTypeQuerier struct {
	BaseMockQuerier[domain.AgentType]
}

// Ensure interface compatibility
var _ domain.EventQuerier = (*mockEventQuerier)(nil)

// mockEventQuerier is a custom mock for EventQuerier
type mockEventQuerier struct {
	BaseMockQuerier[domain.Event]
	listFromSequenceFunc func(ctx context.Context, fromSequenceNumber int64, limit int) ([]*domain.Event, error)
}

func (m *mockEventQuerier) ListFromSequence(ctx context.Context, fromSequenceNumber int64, limit int) ([]*domain.Event, error) {
	if m.listFromSequenceFunc != nil {
		return m.listFromSequenceFunc(ctx, fromSequenceNumber, limit)
	}
	return []*domain.Event{}, nil
}

// Ensure interface compatibility
var _ domain.JobQuerier = (*mockJobQuerier)(nil)

// mockJobQuerier is a custom mock for JobQuerier
type mockJobQuerier struct {
	BaseMockQuerier[domain.Job]
	getPendingJobsForAgentFunc func(ctx context.Context, agentID properties.UUID, limit int) ([]*domain.Job, error)
	getTimeOutJobsFunc         func(ctx context.Context, timeout time.Duration) ([]*domain.Job, error)
}

func (m *mockJobQuerier) GetPendingJobsForAgent(ctx context.Context, agentID properties.UUID, limit int) ([]*domain.Job, error) {
	if m.getPendingJobsForAgentFunc != nil {
		return m.getPendingJobsForAgentFunc(ctx, agentID, limit)
	}
	return []*domain.Job{}, nil
}

// GetTimeOutJobs returns a list of timed out jobs
func (m *mockJobQuerier) GetTimeOutJobs(ctx context.Context, timeout time.Duration) ([]*domain.Job, error) {
	if m.getTimeOutJobsFunc != nil {
		return m.getTimeOutJobsFunc(ctx, timeout)
	}
	return []*domain.Job{}, nil
}

// Ensure interface compatibility
var _ domain.MetricEntryQuerier = (*mockMetricEntryQuerier)(nil)

type mockMetricEntryQuerier struct {
	BaseMockQuerier[domain.MetricEntry]
	countByMetricTypeFunc func(ctx context.Context, typeID properties.UUID) (int64, error)
}

func (m *mockMetricEntryQuerier) CountByMetricType(ctx context.Context, typeID properties.UUID) (int64, error) {
	if m.countByMetricTypeFunc != nil {
		return m.countByMetricTypeFunc(ctx, typeID)
	}
	return 0, nil
}

// Ensure interface compatibility
var _ domain.MetricTypeQuerier = (*mockMetricTypeQuerier)(nil)

// mockMetricTypeQuerier is a custom mock for MetricTypeQuerier
type mockMetricTypeQuerier struct {
	BaseMockQuerier[domain.MetricType]
	findByNameFunc func(ctx context.Context, name string) (*domain.MetricType, error)
}

func (m *mockMetricTypeQuerier) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	if m.findByNameFunc != nil {
		return m.findByNameFunc(ctx, name)
	}
	return nil, fmt.Errorf("FindByName not mocked")
}

// Ensure interface compatibility
var _ domain.ServiceGroupQuerier = (*mockServiceGroupQuerier)(nil)

// mockServiceGroupQuerier is a custom mock for ServiceGroupQuerier
type mockServiceGroupQuerier struct {
	BaseMockQuerier[domain.ServiceGroup]
}

// Ensure interface compatibility
var _ domain.ServiceQuerier = (*mockServiceQuerier)(nil)

// mockServiceQuerier is a custom mock for ServiceQuerier
type mockServiceQuerier struct {
	BaseMockQuerier[domain.Service]
	findByExternalIDFunc func(ctx context.Context, agentID properties.UUID, externalID string) (*domain.Service, error)
	countByGroupFunc     func(ctx context.Context, groupID properties.UUID) (int64, error)
	countByAgentFunc     func(ctx context.Context, agentID properties.UUID) (int64, error)
}

func (m *mockServiceQuerier) FindByExternalID(ctx context.Context, agentID properties.UUID, externalID string) (*domain.Service, error) {
	if m.findByExternalIDFunc != nil {
		return m.findByExternalIDFunc(ctx, agentID, externalID)
	}
	return nil, fmt.Errorf("FindByExternalID not mocked")
}

func (m *mockServiceQuerier) CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error) {
	if m.countByGroupFunc != nil {
		return m.countByGroupFunc(ctx, groupID)
	}
	return 0, nil
}

func (m *mockServiceQuerier) CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error) {
	if m.countByAgentFunc != nil {
		return m.countByAgentFunc(ctx, agentID)
	}
	return 0, nil
}

// Mock interfaces for testing
type mockParticipantQuerier struct {
	BaseMockQuerier[domain.Participant]
}

// mockTokenQuerier is a custom mock for TokenQuerier
type mockTokenQuerier struct {
	BaseMockQuerier[domain.Token]
	findByHashedValueFunc func(ctx context.Context, hashedValue string) (*domain.Token, error)
}

func (m *mockTokenQuerier) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	if m.findByHashedValueFunc != nil {
		return m.findByHashedValueFunc(ctx, hashedValue)
	}
	return nil, domain.NewNotFoundErrorf("token not found")
}
