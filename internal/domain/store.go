package domain

import "context"

type Store interface {
	// Transactional callback
	Atomic(context.Context, func(Store) error) error

	// Repositories
	AgentTypeRepo() AgentTypeRepository
	AgentRepo() AgentRepository
	ProviderRepo() ProviderRepository
	ServiceTypeRepo() ServiceTypeRepository
	ServiceGroupRepo() ServiceGroupRepository
	ServiceRepo() ServiceRepository
	JobRepo() JobRepository
	AuditEntryRepo() AuditEntryRepository
	MetricTypeRepo() MetricTypeRepository
	MetricEntryRepo() MetricEntryRepository
}
