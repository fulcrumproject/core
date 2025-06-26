package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

func TestBaseEntity_GetID(t *testing.T) {
	t.Run("Should return the entity's ID", func(t *testing.T) {
		// Create a properties.UUID and a BaseEntity with it
		expectedID := properties.NewUUID()
		entity := BaseEntity{
			ID:        expectedID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Call GetID
		actualID := entity.GetID()

		// Verify the ID is returned correctly
		if actualID != expectedID {
			t.Errorf("GetID returned incorrect ID. Expected %v, got %v", expectedID, actualID)
		}
	})

	t.Run("Should return nil properties.UUID when ID is nil", func(t *testing.T) {
		// Create a BaseEntity with nil ID
		entity := BaseEntity{
			ID:        properties.UUID{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Call GetID
		actualID := entity.GetID()

		// Verify nil properties.UUID is returned
		if uuid.UUID(actualID) != uuid.Nil {
			t.Errorf("GetID should return nil properties.UUID, got %v", actualID)
		}
	})
}
