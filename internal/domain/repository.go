package domain

import (
	"context"
	"errors"
)

var (
	// ErrNotFound indica che l'entità richiesta non è stata trovata
	ErrNotFound = errors.New("entity not found")
	// ErrConflict indica che l'operazione non può essere completata a causa di un conflitto
	ErrConflict = errors.New("entity conflict")
	// ErrInvalidInput indica che i dati di input non sono validi
	ErrInvalidInput = errors.New("invalid input")
)

// Repository definisce l'interfaccia base per tutti i repository
type Repository[T any] interface {
	// Create crea una nuova entità
	Create(ctx context.Context, entity *T) error

	// Update aggiorna un'entità esistente
	Update(ctx context.Context, entity *T) error

	// Delete elimina un'entità per ID
	Delete(ctx context.Context, id UUID) error

	// FindByID recupera un'entità per ID
	FindByID(ctx context.Context, id UUID) (*T, error)

	// List recupera una lista di entità in base ai filtri forniti
	List(ctx context.Context, filters map[string]interface{}) ([]T, error)
}

// ProviderRepository definisce l'interfaccia per il repository dei Provider
type ProviderRepository interface {
	Repository[Provider]

	// FindByCountryCode recupera i provider per codice paese
	FindByCountryCode(ctx context.Context, code string) ([]Provider, error)

	// UpdateState aggiorna lo stato di un provider
	UpdateState(ctx context.Context, id UUID, state ProviderState) error
}

// AgentRepository definisce l'interfaccia per il repository degli Agent
type AgentRepository interface {
	Repository[Agent]

	// FindByProvider recupera gli agent per un provider specifico
	FindByProvider(ctx context.Context, providerID UUID) ([]Agent, error)

	// FindByAgentType recupera gli agent per un tipo specifico
	FindByAgentType(ctx context.Context, agentTypeID UUID) ([]Agent, error)

	// UpdateState aggiorna lo stato di un agent
	UpdateState(ctx context.Context, id UUID, state AgentState) error
}

// AgentTypeRepository definisce l'interfaccia per il repository degli AgentType
type AgentTypeRepository interface {
	Repository[AgentType]

	// FindByServiceType recupera i tipi di agent che supportano un tipo di servizio specifico
	FindByServiceType(ctx context.Context, serviceTypeID UUID) ([]AgentType, error)

	// AddServiceType aggiunge un tipo di servizio a un tipo di agent
	AddServiceType(ctx context.Context, agentTypeID, serviceTypeID UUID) error

	// RemoveServiceType rimuove un tipo di servizio da un tipo di agent
	RemoveServiceType(ctx context.Context, agentTypeID, serviceTypeID UUID) error
}

// ServiceTypeRepository definisce l'interfaccia per il repository dei ServiceType
type ServiceTypeRepository interface {
	Repository[ServiceType]

	// FindByAgentType recupera i tipi di servizio supportati da un tipo di agent
	FindByAgentType(ctx context.Context, agentTypeID UUID) ([]ServiceType, error)

	// UpdateResourceDefinitions aggiorna le definizioni delle risorse di un tipo di servizio
	UpdateResourceDefinitions(ctx context.Context, id UUID, definitions JSON) error
}
