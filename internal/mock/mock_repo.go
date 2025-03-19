package mock

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
)

type MockProviderRepo struct {
	baseRepo[domain.Provider]
}

type MockAgentRepo struct {
	baseRepo[domain.Agent]
}

func (m *MockAgentRepo) GetAgentsByAgentTypeID(ctx context.Context, agentTypeID domain.UUID) ([]*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) CountByProvider(ctx context.Context, providerID domain.UUID) (int64, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Agent, error) {
	panic("not implemented")
}

func (m *MockAgentRepo) MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	panic("not implemented")
}

type baseRepo[T any] struct{}

func (m *baseRepo[T]) Create(ctx context.Context, entity *T) error {
	panic("not implemented")
}

func (m *baseRepo[T]) Save(ctx context.Context, entity *T) error {
	panic("not implemented")
}

func (m *baseRepo[T]) Delete(ctx context.Context, id domain.UUID) error {
	panic("not implemented")
}

func (m *baseRepo[T]) FindByID(ctx context.Context, id domain.UUID) (*T, error) {
	panic("not implemented")
}

func (m *baseRepo[T]) List(ctx context.Context, req *domain.PageRequest) (*domain.PageResponse[T], error) {
	panic("not implemented")
}
