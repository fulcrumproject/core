package domain

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

// Entity defines the interface that all domain entities must implement
type Entity interface {
	GetID() properties.UUID
}

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        properties.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time       `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time       `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// GetID returns the entity's ID
func (b BaseEntity) GetID() properties.UUID {
	return b.ID
}

// BaseEntityRepository defines the interface for the BaseEntity repository
type BaseEntityRepository[T Entity] interface {
	BaseEntityQuerier[T]

	// Create creates a new entity
	Create(ctx context.Context, entity *T) error

	// Save updates an existing entity
	Save(ctx context.Context, entity *T) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id properties.UUID) error
}

// BaseEntityQuerier defines the interface for the BaseEntity read-only queries
type BaseEntityQuerier[T Entity] interface {

	// Get retrieves an entity by ID
	Get(ctx context.Context, id properties.UUID) (*T, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id properties.UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, scope *auth.IdentityScope, req *PageReq) (*PageRes[T], error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)

	// AuthScope returns the authorization scope for the entity
	AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}
