package database

import (
	"context"

	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

// GormStore implements the domain.Store interface using GORM
type GormStore struct {
	db                    *gorm.DB
	participantRepo       domain.ParticipantRepository
	tokenRepo             domain.TokenRepository
	agentTypeRepo         domain.AgentTypeRepository
	agentRepo             domain.AgentRepository
	serviceTypeRepo       domain.ServiceTypeRepository
	serviceGroupRepo      domain.ServiceGroupRepository
	serviceRepo           domain.ServiceRepository
	jobRepo               domain.JobRepository
	eventEntryRepo        domain.EventRepository
	eventSubscriptionRepo domain.EventSubscriptionRepository
	metricTypeRepo        domain.MetricTypeRepository
}

// NewGormStore creates a new GormStore instance
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{
		db: db,
	}
}

// Atomic executes the given function within a transaction
func (s *GormStore) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create a new store with the transaction
		txStore := NewGormStore(tx)
		// Execute the function with the transaction store
		return fn(txStore)
	})
}

func (s *GormStore) ParticipantRepo() domain.ParticipantRepository {
	if s.participantRepo == nil {
		s.participantRepo = NewParticipantRepository(s.db)
	}
	return s.participantRepo
}

func (s *GormStore) TokenRepo() domain.TokenRepository {
	if s.tokenRepo == nil {
		s.tokenRepo = NewTokenRepository(s.db)
	}
	return s.tokenRepo
}

func (s *GormStore) AgentTypeRepo() domain.AgentTypeRepository {
	if s.agentTypeRepo == nil {
		s.agentTypeRepo = NewAgentTypeRepository(s.db)
	}
	return s.agentTypeRepo
}

func (s *GormStore) AgentRepo() domain.AgentRepository {
	if s.agentRepo == nil {
		s.agentRepo = NewAgentRepository(s.db)
	}
	return s.agentRepo
}

func (s *GormStore) ServiceTypeRepo() domain.ServiceTypeRepository {
	if s.serviceTypeRepo == nil {
		s.serviceTypeRepo = NewServiceTypeRepository(s.db)
	}
	return s.serviceTypeRepo
}

func (s *GormStore) ServiceGroupRepo() domain.ServiceGroupRepository {
	if s.serviceGroupRepo == nil {
		s.serviceGroupRepo = NewServiceGroupRepository(s.db)
	}
	return s.serviceGroupRepo
}

func (s *GormStore) ServiceRepo() domain.ServiceRepository {
	if s.serviceRepo == nil {
		s.serviceRepo = NewServiceRepository(s.db)
	}
	return s.serviceRepo
}

func (s *GormStore) JobRepo() domain.JobRepository {
	if s.jobRepo == nil {
		s.jobRepo = NewJobRepository(s.db)
	}
	return s.jobRepo
}

func (s *GormStore) EventRepo() domain.EventRepository {
	if s.eventEntryRepo == nil {
		s.eventEntryRepo = NewEventRepository(s.db)
	}
	return s.eventEntryRepo
}

func (s *GormStore) EventSubscriptionRepo() domain.EventSubscriptionRepository {
	if s.eventSubscriptionRepo == nil {
		s.eventSubscriptionRepo = NewEventSubscriptionRepository(s.db)
	}
	return s.eventSubscriptionRepo
}

func (s *GormStore) MetricTypeRepo() domain.MetricTypeRepository {
	if s.metricTypeRepo == nil {
		s.metricTypeRepo = NewMetricTypeRepository(s.db)
	}
	return s.metricTypeRepo
}

// GormReadOnlyStore implements the domain.ReadOnlyStore interface using GORM
type GormReadOnlyStore struct {
	db *gorm.DB
}

// Check if GormReadOnlyStore implements the domain.ReadOnlyStore interface
var _ domain.ReadOnlyStore = (*GormReadOnlyStore)(nil)

func NewGormReadOnlyStore(db *gorm.DB) *GormReadOnlyStore {
	return &GormReadOnlyStore{db: db}
}

func (s *GormReadOnlyStore) AgentTypeQuerier() domain.AgentTypeQuerier {
	return NewAgentTypeRepository(s.db)
}

func (s *GormReadOnlyStore) AgentQuerier() domain.AgentQuerier {
	return NewAgentRepository(s.db)
}

func (s *GormReadOnlyStore) TokenQuerier() domain.TokenQuerier {
	return NewTokenRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceTypeQuerier() domain.ServiceTypeQuerier {
	return NewServiceTypeRepository(s.db)
}

func (s *GormReadOnlyStore) EventQuerier() domain.EventQuerier {
	return NewEventRepository(s.db)
}

func (s *GormReadOnlyStore) EventSubscriptionQuerier() domain.EventSubscriptionQuerier {
	return NewEventSubscriptionRepository(s.db)
}

func (s *GormReadOnlyStore) MetricTypeQuerier() domain.MetricTypeQuerier {
	return NewMetricTypeRepository(s.db)
}

func (s *GormReadOnlyStore) ParticipantQuerier() domain.ParticipantQuerier {
	return NewParticipantRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceGroupQuerier() domain.ServiceGroupQuerier {
	return NewServiceGroupRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceQuerier() domain.ServiceQuerier {
	return NewServiceRepository(s.db)
}

func (s *GormReadOnlyStore) JobQuerier() domain.JobQuerier {
	return NewJobRepository(s.db)
}
