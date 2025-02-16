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
			found, err := repo.FindByID(ctx, domain.UUID(uuid.New()))

			// Assert
			assert.Nil(t, found)
			assert.ErrorIs(t, err, domain.ErrNotFound)
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

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, nil, nil, pagination)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			assert.Greater(t, result.TotalItems, int64(2))
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			filters := domain.Filters{
				"name": serviceType.Name,
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, filters, nil, pagination)

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

			sorting := &domain.Sorting{
				SortField: "name",
				SortOrder: "desc",
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

			// Setup - Create multiple service types
			for i := 0; i < 5; i++ {
				serviceType := createTestServiceType(t)
				require.NoError(t, repo.Create(ctx, serviceType))
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

			// Verify total count matches
			count, err := repo.Count(ctx, nil)
			require.NoError(t, err)
			assert.Equal(t, result.TotalItems, count)
		})
	})
}

func createTestServiceType(t *testing.T) *domain.ServiceType {
	t.Helper()
	randomSuffix := uuid.New().String()

	return &domain.ServiceType{
		Name: fmt.Sprintf("Test Service Type %s", randomSuffix),
	}
}
