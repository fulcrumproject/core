package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
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

// JSON type alias
type JSON = datatypes.JSONMap

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

func ParseCountryCode(value string) (CountryCode, error) {
	code := CountryCode(value)
	return code, code.Validate()
}

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

// UUID represents a unique identifier
type UUID = datatypes.UUID

func NewUUID() UUID {
	return UUID(uuid.Must(uuid.NewV7()))
}

// parseID is a helper function to parse and validate IDs
func ParseID(id string) (UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return UUID{}, err
	}
	return UUID(uid), nil
}

// Attributes represents a map of string arrays used for flexible entity attributes
type Attributes map[string][]string

// Scan implements the sql.Scanner interface
func (a *Attributes) Scan(value interface{}) error {
	if value == nil {
		*a = make(Attributes)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Attributes value: %v", value)
	}

	return json.Unmarshal(bytes, a)
}

// Value implements the driver.Valuer interface
func (a Attributes) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// GormDataType returns the GORM data type for Attributes
func (a Attributes) GormDataType() string {
	return "jsonb"
}

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
