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
	ServiceOptionTypeRepo() ServiceOptionTypeRepository
	ServiceOptionRepo() ServiceOptionRepository
	JobRepo() JobRepository
	EventRepo() EventRepository
	EventSubscriptionRepo() EventSubscriptionRepository
	MetricTypeRepo() MetricTypeRepository
	ParticipantRepo() ParticipantRepository
}

// ReadOnlyStore provides data access to all repositories and supports transactions.
type ReadOnlyStore interface {
	// Queriers
	AgentTypeQuerier() AgentTypeQuerier
	AgentQuerier() AgentQuerier
	TokenQuerier() TokenQuerier
	ServiceTypeQuerier() ServiceTypeQuerier
	ServiceGroupQuerier() ServiceGroupQuerier
	ServiceQuerier() ServiceQuerier
	ServiceOptionTypeQuerier() ServiceOptionTypeQuerier
	ServiceOptionQuerier() ServiceOptionQuerier
	JobQuerier() JobQuerier
	EventQuerier() EventQuerier
	EventSubscriptionQuerier() EventSubscriptionQuerier
	MetricTypeQuerier() MetricTypeQuerier
	ParticipantQuerier() ParticipantQuerier
}
