package common

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UUID represents a unique identifier
type UUID = uuid.UUID

// State represents a generic state type that can be used across different entities
type State string

// Attributes represents a map of string arrays used for flexible entity attributes
type Attributes map[string][]string

// GormAttributes handles the custom JSON serialization for GORM
type GormAttributes datatypes.JSON

// JSON represents a generic JSON object
type JSON map[string]interface{}

// GormJSON handles the custom JSON serialization for GORM
type GormJSON datatypes.JSON

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// BeforeCreate ensures UUID is set before creating a record
func (b *BaseEntity) BeforeCreate() error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
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

// ToGormAttributes converts Attributes to GormAttributes
func (a Attributes) ToGormAttributes() (GormAttributes, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return GormAttributes(datatypes.JSON(data)), nil
}

// ToAttributes converts GormAttributes to Attributes
func (ga GormAttributes) ToAttributes() (Attributes, error) {
	var attrs Attributes
	if err := json.Unmarshal([]byte(ga), &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

// ToGormJSON converts JSON to GormJSON
func (j JSON) ToGormJSON() (GormJSON, error) {
	data, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return GormJSON(datatypes.JSON(data)), nil
}

// ToJSON converts GormJSON to JSON
func (gj GormJSON) ToJSON() (JSON, error) {
	var j JSON
	if err := json.Unmarshal([]byte(gj), &j); err != nil {
		return nil, err
	}
	return j, nil
}

// Value implements the driver.Valuer interface for GormAttributes
func (ga GormAttributes) Value() (interface{}, error) {
	return datatypes.JSON(ga).Value()
}

// Scan implements the sql.Scanner interface for GormAttributes
func (ga *GormAttributes) Scan(value interface{}) error {
	return (*datatypes.JSON)(ga).Scan(value)
}

// Value implements the driver.Valuer interface for GormJSON
func (gj GormJSON) Value() (interface{}, error) {
	return datatypes.JSON(gj).Value()
}

// Scan implements the sql.Scanner interface for GormJSON
func (gj *GormJSON) Scan(value interface{}) error {
	return (*datatypes.JSON)(gj).Scan(value)
}
