package domain

import (
	"github.com/google/uuid"
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
