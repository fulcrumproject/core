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
	serviceOptionTypeRepo domain.ServiceOptionTypeRepository
	serviceOptionRepo     domain.ServiceOptionRepository
	servicePoolSetRepo    domain.ServicePoolSetRepository
	servicePoolRepo       domain.ServicePoolRepository
	servicePoolValueRepo  domain.ServicePoolValueRepository
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
		s.participantRepo = NewGenParticipantRepository(s.db)
	}
	return s.participantRepo
}

func (s *GormStore) TokenRepo() domain.TokenRepository {
	if s.tokenRepo == nil {
		s.tokenRepo = NewGenTokenRepository(s.db)
	}
	return s.tokenRepo
}

func (s *GormStore) AgentTypeRepo() domain.AgentTypeRepository {
	if s.agentTypeRepo == nil {
		s.agentTypeRepo = NewGenAgentTypeRepository(s.db)
	}
	return s.agentTypeRepo
}

func (s *GormStore) AgentRepo() domain.AgentRepository {
	if s.agentRepo == nil {
		s.agentRepo = NewGenAgentRepository(s.db)
	}
	return s.agentRepo
}

func (s *GormStore) ServiceTypeRepo() domain.ServiceTypeRepository {
	if s.serviceTypeRepo == nil {
		s.serviceTypeRepo = NewGenServiceTypeRepository(s.db)
	}
	return s.serviceTypeRepo
}

func (s *GormStore) ServiceGroupRepo() domain.ServiceGroupRepository {
	if s.serviceGroupRepo == nil {
		s.serviceGroupRepo = NewGenServiceGroupRepository(s.db)
	}
	return s.serviceGroupRepo
}

func (s *GormStore) ServiceRepo() domain.ServiceRepository {
	if s.serviceRepo == nil {
		s.serviceRepo = NewGenServiceRepository(s.db)
	}
	return s.serviceRepo
}

func (s *GormStore) JobRepo() domain.JobRepository {
	if s.jobRepo == nil {
		s.jobRepo = NewGenJobRepository(s.db)
	}
	return s.jobRepo
}

func (s *GormStore) EventRepo() domain.EventRepository {
	if s.eventEntryRepo == nil {
		s.eventEntryRepo = NewGenEventRepository(s.db)
	}
	return s.eventEntryRepo
}

func (s *GormStore) EventSubscriptionRepo() domain.EventSubscriptionRepository {
	if s.eventSubscriptionRepo == nil {
		s.eventSubscriptionRepo = NewGenEventSubscriptionRepository(s.db)
	}
	return s.eventSubscriptionRepo
}

func (s *GormStore) MetricTypeRepo() domain.MetricTypeRepository {
	if s.metricTypeRepo == nil {
		s.metricTypeRepo = NewGenMetricTypeRepository(s.db)
	}
	return s.metricTypeRepo
}

func (s *GormStore) ServiceOptionTypeRepo() domain.ServiceOptionTypeRepository {
	if s.serviceOptionTypeRepo == nil {
		s.serviceOptionTypeRepo = NewGenServiceOptionTypeRepository(s.db)
	}
	return s.serviceOptionTypeRepo
}

func (s *GormStore) ServiceOptionRepo() domain.ServiceOptionRepository {
	if s.serviceOptionRepo == nil {
		s.serviceOptionRepo = NewGenServiceOptionRepository(s.db)
	}
	return s.serviceOptionRepo
}

func (s *GormStore) ServicePoolSetRepo() domain.ServicePoolSetRepository {
	if s.servicePoolSetRepo == nil {
		s.servicePoolSetRepo = NewGenServicePoolSetRepository(s.db)
	}
	return s.servicePoolSetRepo
}

func (s *GormStore) ServicePoolRepo() domain.ServicePoolRepository {
	if s.servicePoolRepo == nil {
		s.servicePoolRepo = NewGenServicePoolRepository(s.db)
	}
	return s.servicePoolRepo
}

func (s *GormStore) ServicePoolValueRepo() domain.ServicePoolValueRepository {
	if s.servicePoolValueRepo == nil {
		s.servicePoolValueRepo = NewGenServicePoolValueRepository(s.db)
	}
	return s.servicePoolValueRepo
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
	return NewGenAgentTypeRepository(s.db)
}

func (s *GormReadOnlyStore) AgentQuerier() domain.AgentQuerier {
	return NewGenAgentRepository(s.db)
}

func (s *GormReadOnlyStore) TokenQuerier() domain.TokenQuerier {
	return NewGenTokenRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceTypeQuerier() domain.ServiceTypeQuerier {
	return NewGenServiceTypeRepository(s.db)
}

func (s *GormReadOnlyStore) EventQuerier() domain.EventQuerier {
	return NewGenEventRepository(s.db)
}

func (s *GormReadOnlyStore) EventSubscriptionQuerier() domain.EventSubscriptionQuerier {
	return NewGenEventSubscriptionRepository(s.db)
}

func (s *GormReadOnlyStore) MetricTypeQuerier() domain.MetricTypeQuerier {
	return NewGenMetricTypeRepository(s.db)
}

func (s *GormReadOnlyStore) ParticipantQuerier() domain.ParticipantQuerier {
	return NewGenParticipantRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceGroupQuerier() domain.ServiceGroupQuerier {
	return NewGenServiceGroupRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceQuerier() domain.ServiceQuerier {
	return NewGenServiceRepository(s.db)
}

func (s *GormReadOnlyStore) JobQuerier() domain.JobQuerier {
	return NewGenJobRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceOptionTypeQuerier() domain.ServiceOptionTypeQuerier {
	return NewGenServiceOptionTypeRepository(s.db)
}

func (s *GormReadOnlyStore) ServiceOptionQuerier() domain.ServiceOptionQuerier {
	return NewGenServiceOptionRepository(s.db)
}

func (s *GormReadOnlyStore) ServicePoolSetQuerier() domain.ServicePoolSetQuerier {
	return NewGenServicePoolSetRepository(s.db)
}

func (s *GormReadOnlyStore) ServicePoolQuerier() domain.ServicePoolQuerier {
	return NewGenServicePoolRepository(s.db)
}

func (s *GormReadOnlyStore) ServicePoolValueQuerier() domain.ServicePoolValueQuerier {
	return NewGenServicePoolValueRepository(s.db)
}
