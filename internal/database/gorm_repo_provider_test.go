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

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - list with state filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, repo.Create(ctx, provider))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"state": {"Enabled"}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			for _, p := range result.Items {
				assert.Equal(t, domain.ProviderEnabled, p.State)
			}
		})

		t.Run("success - list with country code filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			provider.CountryCode = "UK"
			require.NoError(t, repo.Create(ctx, provider))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"countryCode": {"UK"}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			for _, p := range result.Items {
				assert.Equal(t, domain.CountryCode("UK"), p.CountryCode)
			}
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider1 := createTestProvider(t, domain.ProviderEnabled)
			provider1.Name = "A Provider"
			require.NoError(t, repo.Create(ctx, provider1))

			provider2 := createTestProvider(t, domain.ProviderEnabled)
			provider2.Name = "B Provider"
			require.NoError(t, repo.Create(ctx, provider2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup - Create multiple providers
			for i := 0; i < 5; i++ {
				provider := createTestProvider(t, domain.ProviderEnabled)
				require.NoError(t, repo.Create(ctx, provider))
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
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
			assert.Equal(t, "Updated Provider", updated.Name)
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
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}
