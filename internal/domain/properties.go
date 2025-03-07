package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UUID represents a unique identifier
type UUID = uuid.UUID

func NewUUID() UUID {
	return UUID(uuid.Must(uuid.NewV7()))
}

// ParseUUID is a helper function to parse and validate IDs
func ParseUUID(id string) (UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return UUID{}, err
	}
	return UUID(uid), nil
}

// JSON type alias
type JSON = datatypes.JSONMap

// CountryCode represents a validated ISO 3166-1 alpha-2 country code
type CountryCode string

// Validate ensures the CountryCode is a valid ISO 3166-1 alpha-2 code
func (c CountryCode) Validate() error {
	code := string(c)
	if len(code) != 2 {
		return fmt.Errorf("invalid lentgh for ISO 3166-1 alpha-2 country code %s", code)
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return fmt.Errorf("invalid chars for ISO 3166-1 alpha-2 country code %s", code)
		}
	}
	return nil
}

func ParseCountryCode(value string) (CountryCode, error) {
	code := CountryCode(value)
	return code, code.Validate()
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
			return errors.New("attribute keys cannot be empty")
		}
		if len(values) == 0 {
			return fmt.Errorf("attribute key %s has empty values array", key)
		}
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("attribute key %s has an empty value", key)
			}
		}
	}
	return nil
}
