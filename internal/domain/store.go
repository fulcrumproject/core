package domain

import "context"

<<<<<<< HEAD
type WriteStore interface {
	// Transactional callback
	Atomic(context.Context, func(WriteStore) error) error
=======
type Store interface {
	// Transactional callback
	Atomic(context.Context, func(Store) error) error
>>>>>>> 14-transactions

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
