package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        UUID      `gorm:"type:uuid;primary_key"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// BeforeCreate ensures UUID is set before creating a record
func (b *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if uuid.UUID(b.ID) == uuid.Nil {
		b.ID = NewUUID()
	}
	return nil
}

// GetID returns the entity's ID
func (b BaseEntity) GetID() UUID {
	return b.ID
}
