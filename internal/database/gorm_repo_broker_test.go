package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrokerRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewBrokerRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			broker := createTestBroker(t)

			// Execute
			err := repo.Create(ctx, broker)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, broker.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, broker.ID)
			require.NoError(t, err)
			assert.Equal(t, broker.Name, found.Name)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			broker1 := createTestBroker(t)
			require.NoError(t, repo.Create(ctx, broker1))
			broker2 := createTestBroker(t)
			require.NoError(t, repo.Create(ctx, broker2))

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
			broker := createTestBroker(t)
			require.NoError(t, repo.Create(ctx, broker))

			nameFilter := broker.Name

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters: map[string][]string{
					"name": {nameFilter},
				},
			}

			// Execute
			result, err := repo.List(ctx, page)

			// Assert
			require.NoError(t, err)
			found := false
			for _, b := range result.Items {
				if b.ID == broker.ID {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected to find the broker with the filtered name")
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			broker1 := createTestBroker(t)
			broker1.Name = "A Broker"
			require.NoError(t, repo.Create(ctx, broker1))

			broker2 := createTestBroker(t)
			broker2.Name = "B Broker"
			require.NoError(t, repo.Create(ctx, broker2))

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

			// Setup - Create multiple brokers
			for i := 0; i < 5; i++ {
				broker := createTestBroker(t)
				require.NoError(t, repo.Create(ctx, broker))
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

	t.Run("Update", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			broker := createTestBroker(t)
			require.NoError(t, repo.Create(ctx, broker))

			// Read
			broker, err := repo.FindByID(ctx, broker.ID)
			require.NoError(t, err)

			// Update broker
			broker.Name = "Updated Broker"

			// Execute
			err = repo.Save(ctx, broker)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := repo.FindByID(ctx, broker.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Broker", updated.Name)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			broker := createTestBroker(t)
			require.NoError(t, repo.Create(ctx, broker))

			// Execute
			err := repo.Delete(ctx, broker.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := repo.FindByID(ctx, broker.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}
