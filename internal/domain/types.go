package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Validatable is the interface for validatable types
type Validatable interface {
	Validate() error
}

// ValidationError represents a domain validation error
type ValidationError string

// Error implements the error interface
func (e ValidationError) Error() string {
	return string(e)
}

// Name represents a validated string that cannot be empty
type Name string

// Validate ensures the Name is not empty
func (n Name) Validate() error {
	if string(n) == "" {
		return ValidationError("name cannot be empty")
	}
	return nil
}

// CountryCode represents a validated ISO 3166-1 alpha-2 country code
type CountryCode string

// Validate ensures the CountryCode is a valid ISO 3166-1 alpha-2 code
func (c CountryCode) Validate() error {
	code := string(c)
	if len(code) != 2 {
		return ValidationError("country code must be exactly 2 characters")
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return ValidationError("country code must contain only uppercase letters")
		}
	}
	return nil
}

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// BeforeCreate ensures UUID is set before creating a record
func (b *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	if uuid.UUID(b.ID) == uuid.Nil {
		b.ID = UUID(uuid.New())
	}
	return nil
}

// UUID represents a unique identifier
type UUID uuid.UUID

// JSON handles the JSON serialization for GORM
type JSON datatypes.JSON

// Attributes represents a map of string arrays used for flexible entity attributes
type Attributes map[string][]string

// MarshalJSON implements custom JSON marshaling for Attributes
func (a Attributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string][]string(a))
}

// UnmarshalJSON implements custom JSON unmarshaling for Attributes
func (a *Attributes) UnmarshalJSON(data []byte) error {
	var m map[string][]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*a = Attributes(m)
	return nil
}

// Validate checks if the Attributes are valid:
// - All keys must be non-empty strings
// - All arrays must have at least one value
// - All values must be non-empty strings
func (a Attributes) Validate() error {
	for key, values := range a {
		if key == "" {
			return ValidationError("attribute key cannot be empty")
		}
		if len(values) == 0 {
			return ValidationError("attribute values array cannot be empty")
		}
		for _, value := range values {
			if value == "" {
				return ValidationError("attribute value cannot be empty")
			}
		}
	}
	return nil
}
