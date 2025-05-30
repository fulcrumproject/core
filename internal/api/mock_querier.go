package api

import (
	"context"
	"fmt"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

// Ensure interface compatibility
var _ domain.AgentQuerier = (*mockAgentQuerier)(nil)

// mockAgentQuerier is a custom mock for AgentQuerier
type mockAgentQuerier struct {
	findByIDFunc           func(ctx context.Context, id domain.UUID) (*domain.Agent, error)
	listFunc               func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error)
	countByParticipantFunc func(ctx context.Context, participantID domain.UUID) (int64, error)
	existsFunc             func(ctx context.Context, id domain.UUID) (bool, error)
	authScopeFunc          func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockAgentQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domain.NewNotFoundErrorf("agent not found")
}

func (m *mockAgentQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.Agent]{
		Items:       []domain.Agent{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockAgentQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

// CountByProvider is required by the AgentQuerier interface
func (m *mockAgentQuerier) CountByProvider(ctx context.Context, participantID domain.UUID) (int64, error) {
	if m.countByParticipantFunc != nil {
		return m.countByParticipantFunc(ctx, participantID)
	}
	return 0, nil
}

// Exists checks if an agent with the given ID exists
func (m *mockAgentQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	// Default implementation returns true
	return true, nil
}

// Ensure interface compatibility
var _ domain.ServiceTypeQuerier = (*mockServiceTypeQuerier)(nil)

// mockServiceTypeQuerier is a custom mock for ServiceTypeQuerier
type mockServiceTypeQuerier struct {
	findByIDFunc  func(ctx context.Context, id domain.UUID) (*domain.ServiceType, error)
	existsFunc    func(ctx context.Context, id domain.UUID) (bool, error)
	listFunc      func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
	countFunc     func(ctx context.Context) (int64, error)
}

func (m *mockServiceTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("FindByID not mocked")
}

func (m *mockServiceTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockServiceTypeQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.ServiceType]{
		Items:       []domain.ServiceType{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockServiceTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

func (m *mockServiceTypeQuerier) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

// Ensure interface compatibility
var _ domain.AgentTypeQuerier = (*mockAgentTypeQuerier)(nil)

// mockAgentTypeQuerier is a custom mock for AgentTypeQuerier that allows us to set up expected values and error returns
type mockAgentTypeQuerier struct {
	findByIDFunc  func(ctx context.Context, id domain.UUID) (*domain.AgentType, error)
	existsFunc    func(ctx context.Context, id domain.UUID) (bool, error)
	listFunc      func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error)
	countFunc     func(ctx context.Context) (int64, error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockAgentTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.AgentType, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domain.NewNotFoundErrorf("agent type not found")
}

// Exists checks if an agent type with the given ID exists
func (m *mockAgentTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockAgentTypeQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.AgentType]{
		Items:       []domain.AgentType{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockAgentTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

// Count returns the total count of agent types
func (m *mockAgentTypeQuerier) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

// Ensure interface compatibility
var _ domain.AuditEntryQuerier = (*mockAuditEntryQuerier)(nil)

// mockAuditEntryQuerier is a custom mock for AuditEntryQuerier
type mockAuditEntryQuerier struct {
	listFunc      func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockAuditEntryQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AuditEntry], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.AuditEntry]{
		Items:       []domain.AuditEntry{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockAuditEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

// Ensure interface compatibility
var _ domain.JobQuerier = (*mockJobQuerier)(nil)

// mockJobQuerier is a custom mock for JobQuerier
type mockJobQuerier struct {
	findByIDFunc               func(ctx context.Context, id domain.UUID) (*domain.Job, error)
	existsFunc                 func(ctx context.Context, id domain.UUID) (bool, error)
	listFunc                   func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error)
	getPendingJobsForAgentFunc func(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error)
	getTimeOutJobsFunc         func(ctx context.Context, timeout time.Duration) ([]*domain.Job, error)
	authScopeFunc              func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockJobQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Job, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domain.NewNotFoundErrorf("job not found")
}

// Exists checks if a job with the given ID exists
func (m *mockJobQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockJobQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.Job]{
		Items:       []domain.Job{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockJobQuerier) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
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

func (m *mockJobQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

// Ensure interface compatibility
var _ domain.MetricEntryQuerier = (*mockMetricEntryQuerier)(nil)

type mockMetricEntryQuerier struct {
	listFunc      func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error)
	existsFunc    func(ctx context.Context, id domain.UUID) (bool, error)
	countFunc     func(ctx context.Context, typeID domain.UUID) (int64, error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockMetricEntryQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.MetricEntry]{
		Items:       []domain.MetricEntry{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockMetricEntryQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockMetricEntryQuerier) CountByMetricType(ctx context.Context, typeID domain.UUID) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, typeID)
	}
	return 0, nil
}

func (m *mockMetricEntryQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

// Ensure interface compatibility
var _ domain.MetricTypeQuerier = (*mockMetricTypeQuerier)(nil)

// mockMetricTypeQuerier is a custom mock for MetricTypeQuerier
type mockMetricTypeQuerier struct {
	findByIDFunc   func(ctx context.Context, id domain.UUID) (*domain.MetricType, error)
	existsFunc     func(ctx context.Context, id domain.UUID) (bool, error)
	findByNameFunc func(ctx context.Context, name string) (*domain.MetricType, error)
	listFunc       func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error)
	authScopeFunc  func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
	countFunc      func(ctx context.Context) (int64, error)
}

func (m *mockMetricTypeQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("FindByID not mocked")
}

func (m *mockMetricTypeQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockMetricTypeQuerier) FindByName(ctx context.Context, name string) (*domain.MetricType, error) {
	if m.findByNameFunc != nil {
		return m.findByNameFunc(ctx, name)
	}
	return nil, fmt.Errorf("FindByName not mocked")
}

func (m *mockMetricTypeQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.MetricType]{
		Items:       []domain.MetricType{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockMetricTypeQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

func (m *mockMetricTypeQuerier) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

// Ensure interface compatibility
var _ domain.ServiceGroupQuerier = (*mockServiceGroupQuerier)(nil)

// mockServiceGroupQuerier is a custom mock for ServiceGroupQuerier
type mockServiceGroupQuerier struct {
	findByIDFunc  func(ctx context.Context, id domain.UUID) (*domain.ServiceGroup, error)
	existsFunc    func(ctx context.Context, id domain.UUID) (bool, error)
	listFunc      func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
	countFunc     func(ctx context.Context) (int64, error)
}

func (m *mockServiceGroupQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceGroup, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("FindByID not mocked")
}

func (m *mockServiceGroupQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockServiceGroupQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.ServiceGroup]{
		Items:       []domain.ServiceGroup{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockServiceGroupQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

func (m *mockServiceGroupQuerier) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

// Ensure interface compatibility
var _ domain.ServiceQuerier = (*mockServiceQuerier)(nil)

// mockServiceQuerier is a custom mock for ServiceQuerier
type mockServiceQuerier struct {
	findByIDFunc         func(ctx context.Context, id domain.UUID) (*domain.Service, error)
	existsFunc           func(ctx context.Context, id domain.UUID) (bool, error)
	findByExternalIDFunc func(ctx context.Context, agentID domain.UUID, externalID string) (*domain.Service, error)
	listFunc             func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error)
	countByGroupFunc     func(ctx context.Context, groupID domain.UUID) (int64, error)
	countByAgentFunc     func(ctx context.Context, agentID domain.UUID) (int64, error)
	authScopeFunc        func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockServiceQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("FindByID not mocked")
}

func (m *mockServiceQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockServiceQuerier) FindByExternalID(ctx context.Context, agentID domain.UUID, externalID string) (*domain.Service, error) {
	if m.findByExternalIDFunc != nil {
		return m.findByExternalIDFunc(ctx, agentID, externalID)
	}
	return nil, fmt.Errorf("FindByExternalID not mocked")
}

func (m *mockServiceQuerier) List(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, authScope, req)
	}
	return &domain.PageResponse[domain.Service]{
		Items:       []domain.Service{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockServiceQuerier) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	if m.countByGroupFunc != nil {
		return m.countByGroupFunc(ctx, groupID)
	}
	return 0, nil
}

func (m *mockServiceQuerier) CountByAgent(ctx context.Context, agentID domain.UUID) (int64, error) {
	if m.countByAgentFunc != nil {
		return m.countByAgentFunc(ctx, agentID)
	}
	return 0, nil
}

func (m *mockServiceQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}

// Mock interfaces for testing
type mockParticipantQuerier struct {
	findByIDFunc  func(ctx context.Context, id domain.UUID) (*domain.Participant, error)
	listFunc      func(ctx context.Context, scope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Participant], error)
	authScopeFunc func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
	existsFunc    func(ctx context.Context, id domain.UUID) (bool, error)
}

func (m *mockParticipantQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Participant, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockParticipantQuerier) Exists(ctx context.Context, id domain.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return false, nil
}

func (m *mockParticipantQuerier) List(ctx context.Context, scope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Participant], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, scope, req)
	}
	return nil, nil
}

func (m *mockParticipantQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return nil, nil
}

// mockTokenQuerier is a custom mock for TokenQuerier
type mockTokenQuerier struct {
	findByIDFunc          func(ctx context.Context, id domain.UUID) (*domain.Token, error)
	listFunc              func(ctx context.Context, scope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error)
	findByHashedValueFunc func(ctx context.Context, hashedValue string) (*domain.Token, error)
	authScopeFunc         func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error)
}

func (m *mockTokenQuerier) FindByID(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domain.NewNotFoundErrorf("token not found")
}

func (m *mockTokenQuerier) List(ctx context.Context, scope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, scope, req)
	}
	return &domain.PageResponse[domain.Token]{
		Items:       []domain.Token{},
		TotalItems:  0,
		CurrentPage: 1,
		TotalPages:  0,
		HasNext:     false,
	}, nil
}

func (m *mockTokenQuerier) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	if m.findByHashedValueFunc != nil {
		return m.findByHashedValueFunc(ctx, hashedValue)
	}
	return nil, domain.NewNotFoundErrorf("token not found")
}

func (m *mockTokenQuerier) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	if m.authScopeFunc != nil {
		return m.authScopeFunc(ctx, id)
	}
	return &domain.EmptyAuthTargetScope, nil
}
