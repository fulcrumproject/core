package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewProviderRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			// Setup
			provider := &domain.Provider{
				Name:        "Test Provider",
				State:       domain.ProviderEnabled,
				CountryCode: "US",
				Attributes:  domain.Attributes{"key": []string{"value"}},
			}

			// Execute
			err := repo.Create(context.Background(), provider)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, provider.ID)

			// Verify in database
			var found domain.Provider
			err = tdb.DB.First(&found, provider.ID).Error
			require.NoError(t, err)
			assert.Equal(t, provider.Name, found.Name)
			assert.Equal(t, provider.State, found.State)
			assert.Equal(t, provider.CountryCode, found.CountryCode)
			assert.Equal(t, provider.Attributes, found.Attributes)
		})
	})
}
