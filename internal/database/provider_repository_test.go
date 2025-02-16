package database

import (
	"context"
	"fmt"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewProviderRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)

			// Execute
			err := repo.Create(ctx, provider)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, provider.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, provider.ID)
			require.NoError(t, err)
			assert.Equal(t, provider.Name, found.Name)
			assert.Equal(t, provider.State, found.State)
			assert.Equal(t, provider.CountryCode, found.CountryCode)
			assert.Equal(t, provider.Attributes, found.Attributes)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider1 := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, repo.Create(ctx, provider1))
			provider2 := createTestProvider(t, domain.ProviderDisabled)
			require.NoError(t, repo.Create(ctx, provider2))

			// Execute
			providers, err := repo.List(ctx, nil)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(providers), 0)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			filters := map[string]interface{}{
				"state": domain.ProviderEnabled,
			}

			// Execute
			providers, err := repo.List(ctx, filters)

			// Assert
			require.NoError(t, err)
			for _, p := range providers {
				assert.Equal(t, domain.ProviderEnabled, p.State)
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, repo.Create(ctx, provider))

			// Read
			provider, err := repo.FindByID(ctx, provider.ID)
			require.NoError(t, err)

			// Update provider
			provider.Name = "Updated Provider"
			provider.State = domain.ProviderDisabled
			provider.CountryCode = "UK"
			provider.Attributes = domain.Attributes{"new_key": []string{"new_value"}}

			// Execute
			err = repo.Save(ctx, provider)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := repo.FindByID(ctx, provider.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.Name("Updated Provider"), updated.Name)
			assert.Equal(t, domain.ProviderDisabled, updated.State)
			assert.Equal(t, domain.CountryCode("UK"), updated.CountryCode)
			assert.Equal(t, domain.Attributes{"new_key": []string{"new_value"}}, updated.Attributes)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, repo.Create(ctx, provider))

			// Execute
			err := repo.Delete(ctx, provider.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := repo.FindByID(ctx, provider.ID)
			assert.Nil(t, found)
			assert.ErrorIs(t, err, domain.ErrNotFound)
		})
	})
}

func createTestProvider(t *testing.T, state domain.ProviderState) *domain.Provider {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Provider{
		Name:        domain.Name(fmt.Sprintf("Test Provider %s", randomSuffix)),
		State:       state,
		CountryCode: "US",
		Attributes:  domain.Attributes{"key": []string{"value"}},
	}
}
