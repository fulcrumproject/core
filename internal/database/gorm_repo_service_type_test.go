package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceTypeRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewServiceTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)

			// Execute
			err := repo.Create(ctx, serviceType)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, serviceType.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, serviceType.ID)
			require.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			// Execute
			found, err := repo.FindByID(ctx, serviceType.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.FindByID(ctx, uuid.New())

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType1 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType1))
			serviceType2 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			assert.Greater(t, result.TotalItems, int64(2))
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {serviceType.Name}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, serviceType.Name, result.Items[0].Name)
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType1 := createTestServiceType(t)
			serviceType1.Name = "A Service Type"
			require.NoError(t, repo.Create(ctx, serviceType1))

			serviceType2 := createTestServiceType(t)
			serviceType2.Name = "B Service Type"
			require.NoError(t, repo.Create(ctx, serviceType2))

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

			// Setup - Create multiple service types
			for i := 0; i < 5; i++ {
				serviceType := createTestServiceType(t)
				require.NoError(t, repo.Create(ctx, serviceType))
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

			// Verify total count matches
			count, err := repo.Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, result.TotalItems, count)
		})
	})

	t.Run("Count", func(t *testing.T) {
		t.Run("success - count all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType1 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType1))
			serviceType2 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType2))

			// Execute
			count, err := repo.Count(ctx)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, count, int64(1))
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns empty auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a service type
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			// Execute with existing service type ID
			scope, err := repo.AuthScope(ctx, serviceType.ID)
			require.NoError(t, err)
			assert.Equal(t, &domain.AuthScope{}, scope, "Should return empty auth scope for service types")
		})
	})
}
