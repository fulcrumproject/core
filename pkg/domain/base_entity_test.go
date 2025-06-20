package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestBaseEntity_BeforeCreate(t *testing.T) {
	t.Run("When ID is nil, should generate a new properties.UUID", func(t *testing.T) {
		// Create a new BaseEntity with nil ID
		entity := BaseEntity{
			ID:        properties.UUID{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Ensure the ID is actually nil
		if uuid.UUID(entity.ID) != uuid.Nil {
			t.Fatalf("Expected initial ID to be nil, but got %v", entity.ID)
		}

		// Call BeforeCreate (with nil tx as we're not testing DB interactions)
		err := entity.BeforeCreate(nil)
		if err != nil {
			t.Fatalf("BeforeCreate returned an error: %v", err)
		}

		// Verify a new properties.UUID was generated
		if uuid.UUID(entity.ID) == uuid.Nil {
			t.Error("BeforeCreate did not generate a new properties.UUID")
		}
	})

	t.Run("When ID is already set, should not change it", func(t *testing.T) {
		// Create a properties.UUID and a BaseEntity with it
		originalID := properties.NewUUID()
		entity := BaseEntity{
			ID:        originalID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Call BeforeCreate
		err := entity.BeforeCreate(nil)
		if err != nil {
			t.Fatalf("BeforeCreate returned an error: %v", err)
		}

		// Verify ID wasn't changed
		if entity.ID != originalID {
			t.Errorf("BeforeCreate changed the ID. Expected %v, got %v", originalID, entity.ID)
		}
	})

	t.Run("Should accept a non-nil gorm.DB instance", func(t *testing.T) {
		// This test ensures the method handles a real gorm.DB instance without panic
		// Note: We're not actually testing DB interactions, just that it doesn't crash
		entity := BaseEntity{
			ID:        properties.UUID{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create a mock DB instance - we'll use nil here, but in a real case it would be a real DB
		db := &gorm.DB{}

		// Call BeforeCreate with the DB instance
		err := entity.BeforeCreate(db)
		if err != nil {
			t.Fatalf("BeforeCreate returned an error: %v", err)
		}

		// Verify a new properties.UUID was generated
		if uuid.UUID(entity.ID) == uuid.Nil {
			t.Error("BeforeCreate did not generate a new properties.UUID")
		}
	})
}

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
