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
	EventRepo() EventRepository
	EventSubscriptionRepo() EventSubscriptionRepository
	MetricTypeRepo() MetricTypeRepository
	ParticipantRepo() ParticipantRepository
}

// MetricStore provides access to metric entry repository and supports transactions for metrics.
type MetricStore interface {
	// Atomic executes function in a transaction for metric operations
	Atomic(context.Context, func(MetricStore) error) error

	// Repository
	MetricEntryRepo() MetricEntryRepository
}
