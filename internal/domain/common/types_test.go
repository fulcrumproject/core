package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBaseEntity(t *testing.T) {
	t.Run("BeforeCreate sets UUID if nil", func(t *testing.T) {
		entity := &BaseEntity{}
		err := entity.BeforeCreate()
		if err != nil {
			t.Errorf("BeforeCreate returned error: %v", err)
		}
		if entity.ID == uuid.Nil {
			t.Error("BeforeCreate did not set UUID")
		}
	})

	t.Run("BeforeCreate keeps existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		entity := &BaseEntity{ID: existingID}
		err := entity.BeforeCreate()
		if err != nil {
			t.Errorf("BeforeCreate returned error: %v", err)
		}
		if entity.ID != existingID {
			t.Error("BeforeCreate changed existing UUID")
		}
	})
}

func TestAttributes(t *testing.T) {
	t.Run("Marshal and Unmarshal Attributes", func(t *testing.T) {
		attrs := Attributes{
			"key1": {"value1", "value2"},
			"key2": {"value3"},
		}

		// Marshal
		data, err := json.Marshal(attrs)
		if err != nil {
			t.Fatalf("Failed to marshal Attributes: %v", err)
		}

		// Unmarshal
		var unmarshaledAttrs Attributes
		err = json.Unmarshal(data, &unmarshaledAttrs)
		if err != nil {
			t.Fatalf("Failed to unmarshal Attributes: %v", err)
		}

		// Compare
		if len(attrs) != len(unmarshaledAttrs) {
			t.Error("Unmarshaled Attributes has different length")
		}

		for key, values := range attrs {
			unmarshaledValues, ok := unmarshaledAttrs[key]
			if !ok {
				t.Errorf("Key %s not found in unmarshaled Attributes", key)
				continue
			}

			if len(values) != len(unmarshaledValues) {
				t.Errorf("Values length mismatch for key %s", key)
				continue
			}

			for i, value := range values {
				if value != unmarshaledValues[i] {
					t.Errorf("Value mismatch at index %d for key %s", i, key)
				}
			}
		}
	})
}

func TestBaseEntityJSON(t *testing.T) {
	t.Run("Marshal BaseEntity to JSON", func(t *testing.T) {
		now := time.Now()
		entity := BaseEntity{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		}

		data, err := json.Marshal(entity)
		if err != nil {
			t.Fatalf("Failed to marshal BaseEntity: %v", err)
		}

		var unmarshaled BaseEntity
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal BaseEntity: %v", err)
		}

		if entity.ID != unmarshaled.ID {
			t.Error("ID mismatch after JSON marshal/unmarshal")
		}

		// Compare timestamps with some tolerance for JSON serialization
		tolerance := time.Second
		if entity.CreatedAt.Sub(unmarshaled.CreatedAt) > tolerance {
			t.Error("CreatedAt mismatch after JSON marshal/unmarshal")
		}
		if entity.UpdatedAt.Sub(unmarshaled.UpdatedAt) > tolerance {
			t.Error("UpdatedAt mismatch after JSON marshal/unmarshal")
		}
	})
}
