package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestServiceGroupRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceGroupRepository(testDB.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceGroup := createTestServiceGroup(t)

			// Execute
			err := repo.Create(ctx, serviceGroup)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, serviceGroup.ID)
			assert.NotZero(t, serviceGroup.CreatedAt)
			assert.NotZero(t, serviceGroup.UpdatedAt)

			// Verify in database
			found, err := repo.FindByID(ctx, serviceGroup.ID)
			require.NoError(t, err)
			assert.Equal(t, serviceGroup.Name, found.Name)
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceGroup := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, serviceGroup))

			// Execute
			found, err := repo.FindByID(ctx, serviceGroup.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, serviceGroup.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.FindByID(ctx, domain.NewUUID())

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			group1 := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, group1))
			group2 := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, group2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceGroup := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, serviceGroup))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {serviceGroup.Name}},
			}

			// Execute
			result, err := repo.List(ctx, page)

			// Assert
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, serviceGroup.Name, result.Items[0].Name)
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			group1 := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, group1))

			group2 := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, group2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, page)

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

			// Setup - Create multiple service groups
			for i := 0; i < 5; i++ {
				group := createTestServiceGroup(t)
				require.NoError(t, repo.Create(ctx, group))
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceGroup := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, serviceGroup))

			// Update the service group
			serviceGroup.Name = "Updated Group"

			// Execute
			err := repo.Save(ctx, serviceGroup)

			// Assert
			require.NoError(t, err)

			// Verify in database
			found, err := repo.FindByID(ctx, serviceGroup.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Group", found.Name)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceGroup := createTestServiceGroup(t)
			require.NoError(t, repo.Create(ctx, serviceGroup))

			// Execute
			err := repo.Delete(ctx, serviceGroup.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := repo.FindByID(ctx, serviceGroup.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}
