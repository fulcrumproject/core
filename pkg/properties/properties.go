package properties

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UUID represents a unique identifier
type UUID = uuid.UUID

// NewUUID generates a new UUID using version 7
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
