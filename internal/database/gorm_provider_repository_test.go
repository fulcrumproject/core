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

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, nil, nil, pagination)

			// Assert
			providers := result.Items
			require.NoError(t, err)
			assert.Greater(t, len(providers), 0)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			filter := &domain.SimpleFilter{
				Field: "state",
				Value: "Enabled",
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, filter, nil, pagination)

			// Assert
			require.NoError(t, err)
			for _, p := range result.Items {
				assert.Equal(t, domain.ProviderEnabled, p.State)
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

			sorting := &domain.Sorting{
				Field: "name",
				Order: "desc",
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, nil, sorting, pagination)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			assert.GreaterOrEqual(t, result.Items[0].Name, result.Items[1].Name)
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup - Create multiple providers
			for i := 0; i < 5; i++ {
				provider := createTestProvider(t, domain.ProviderEnabled)
				require.NoError(t, repo.Create(ctx, provider))
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, nil, nil, pagination)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			pagination.Page = 2
			result, err = repo.List(ctx, nil, nil, pagination)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)

			// Verify total count
			count, err := repo.Count(ctx, nil)
			require.NoError(t, err)
			assert.Equal(t, result.TotalItems, count)
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

func createTestProvider(t *testing.T, state domain.ProviderState) *domain.Provider {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Provider{
		Name:        fmt.Sprintf("Test Provider %s", randomSuffix),
		State:       state,
		CountryCode: "US",
		Attributes:  domain.Attributes{"key": []string{"value"}},
	}
}
