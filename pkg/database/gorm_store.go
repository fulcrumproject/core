package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/pkg/domain"
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
	metricEntryRepo       domain.MetricEntryRepository
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

func (s *GormStore) MetricEntryRepo() domain.MetricEntryRepository {
	if s.metricEntryRepo == nil {
		s.metricEntryRepo = NewMetricEntryRepository(s.db)
	}
	return s.metricEntryRepo
}
