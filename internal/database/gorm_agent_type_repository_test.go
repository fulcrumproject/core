package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTypeRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewAgentTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType := createTestAgentType(t)

			// Execute
			err := repo.Create(ctx, agentType)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, agentType.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, agentType.ID)
			require.NoError(t, err)
			assert.Equal(t, agentType.Name, found.Name)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType1 := createTestAgentType(t)
			require.NoError(t, repo.Create(ctx, agentType1))
			agentType2 := createTestAgentType(t)
			require.NoError(t, repo.Create(ctx, agentType2))

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, nil, nil, pagination)

			// Assert
			require.NoError(t, err)
			agentTypes := result.Items
			assert.Greater(t, len(agentTypes), 0)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType := createTestAgentType(t)
			require.NoError(t, repo.Create(ctx, agentType))

			filter := &domain.SimpleFilter{
				Field: "name",
				Value: agentType.Name,
			}
			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, filter, nil, pagination)

			// Assert
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			agentTypes := result.Items
			assert.Equal(t, agentType.Name, agentTypes[0].Name)
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType1 := createTestAgentType(t)
			agentType1.Name = "A Agent Type"
			require.NoError(t, repo.Create(ctx, agentType1))

			agentType2 := createTestAgentType(t)
			agentType2.Name = "B Agent Type"
			require.NoError(t, repo.Create(ctx, agentType2))

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

			// Setup - Create multiple agent types
			for i := 0; i < 5; i++ {
				agentType := createTestAgentType(t)
				require.NoError(t, repo.Create(ctx, agentType))
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

	t.Run("FindByID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType := createTestAgentType(t)
			require.NoError(t, repo.Create(ctx, agentType))

			// Execute
			found, err := repo.FindByID(ctx, agentType.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, agentType.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.FindByID(ctx, domain.UUID(uuid.New()))

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}
