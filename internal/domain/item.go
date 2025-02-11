package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/datatypes"
)

type Item struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	Description  string
	Properties   datatypes.JSONMap
	JsonProperty JsonProperty `gorm:"type:jsonb"`
	CreatedAt    time.Time    `gorm:"not null"`
	UpdatedAt    time.Time    `gorm:"not null"`
}

type JsonProperty struct {
	StringProperty string `json:"stringProperty"`
	IntProperty    int    `json:"intProperty"`
}

// Scan implements the sql.Scanner interface for JsonProperty
func (j *JsonProperty) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JsonProperty value")
	}

	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface for JsonProperty
func (j JsonProperty) Value() (driver.Value, error) {
	return json.Marshal(j)
}

type Repository interface {
	Create(item *Item) error
	GetByID(id uint) (*Item, error)
	Update(item *Item) error
	Delete(id uint) error
	List() ([]Item, error)
}
