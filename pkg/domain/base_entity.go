package domain

import (
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        properties.UUID `json:"id" gorm:"type:uuid;primary_key"`
	CreatedAt time.Time       `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time       `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// BeforeCreate ensures properties.UUID is set before creating a record
func (b *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if uuid.UUID(b.ID) == uuid.Nil {
		b.ID = properties.NewUUID()
	}
	return nil
}

// GetID returns the entity's ID
func (b BaseEntity) GetID() properties.UUID {
	return b.ID
}
