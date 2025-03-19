package mock

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
)

// Store provides a simple implementation of the Store interface for testing
// All repository methods return nil (no implementation)
type Store struct{}

// Ensure MockStore implements Store
var _ domain.Store = (*Store)(nil)

// Atomic implements the Store.Atomic method
func (m *Store) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return fn(m)
}

func (m *Store) BrokerRepo() domain.BrokerRepository {
	panic("not implemented")
}

func (m *Store) TokenRepo() domain.TokenRepository {
	panic("not implemented")
}

func (m *Store) AgentTypeRepo() domain.AgentTypeRepository {
	panic("not implemented")
}

func (m *Store) AgentRepo() domain.AgentRepository {
	panic("not implemented")
}

func (m *Store) ProviderRepo() domain.ProviderRepository {
	panic("not implemented")
}

func (m *Store) ServiceTypeRepo() domain.ServiceTypeRepository {
	panic("not implemented")
}

func (m *Store) ServiceGroupRepo() domain.ServiceGroupRepository {
	panic("not implemented")
}

func (m *Store) ServiceRepo() domain.ServiceRepository {
	panic("not implemented")
}

func (m *Store) JobRepo() domain.JobRepository {
	panic("not implemented")
}

func (m *Store) AuditEntryRepo() domain.AuditEntryRepository {
	panic("not implemented")
}

func (m *Store) MetricTypeRepo() domain.MetricTypeRepository {
	panic("not implemented")
}

func (m *Store) MetricEntryRepo() domain.MetricEntryRepository {
	panic("not implemented")
}
