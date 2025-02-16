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

			// Execute
			agentTypes, err := repo.List(ctx, nil)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(agentTypes), 0)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType := createTestAgentType(t)
			require.NoError(t, repo.Create(ctx, agentType))

			filters := map[string]interface{}{
				"name": agentType.Name,
			}

			// Execute
			agentTypes, err := repo.List(ctx, filters)

			// Assert
			require.NoError(t, err)
			require.Len(t, agentTypes, 1)
			assert.Equal(t, agentType.Name, agentTypes[0].Name)
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
			assert.ErrorIs(t, err, domain.ErrNotFound)
		})
	})
}

func createTestAgentType(t *testing.T) *domain.AgentType {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.AgentType{
		Name: fmt.Sprintf("Test Agent Type %s", randomSuffix),
	}
}
