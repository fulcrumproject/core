// Common helper functions for GORM Gen repositories
// Provides utility functions used across multiple repositories
package database

import (
	"github.com/fulcrumproject/core/pkg/properties"
)

// parseUUIDs converts string slice to UUID slice, filtering invalid UUIDs
func parseUUIDs(values []string) []properties.UUID {
	ids := make([]properties.UUID, 0, len(values))
	for _, v := range values {
		if id, err := properties.ParseUUID(v); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

