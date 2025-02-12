package api

import (
	"fmt"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
)

// ParseUUID converte una stringa UUID in domain.UUID
func ParseUUID(s string) (domain.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format: %w", err)
	}
	return id, nil
}
