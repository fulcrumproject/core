package domain

import "context"

// Store provides data access to all repositories and supports transactions.
type Store interface {
	// Atomic executes function in a transaction
	Atomic(context.Context, func(Store) error) error

	// Repositories
	AgentTypeRepo() AgentTypeRepository
	AgentRepo() AgentRepository
	TokenRepo() TokenRepository
	ServiceTypeRepo() ServiceTypeRepository
	ServiceGroupRepo() ServiceGroupRepository
	ServiceRepo() ServiceRepository
	JobRepo() JobRepository
	AuditEntryRepo() AuditEntryRepository
	MetricTypeRepo() MetricTypeRepository
	MetricEntryRepo() MetricEntryRepository
	ParticipantRepo() ParticipantRepository
}
