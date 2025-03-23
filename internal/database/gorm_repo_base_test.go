package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGormRepository_Exists(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// We'll use the agent repository as a concrete example to test the base repository's methods
	agentRepo := NewAgentRepository(tdb.DB)
	providerRepo := NewProviderRepository(tdb.DB)
	agentTypeRepo := NewAgentTypeRepository(tdb.DB)

	t.Run("success - returns true for existing entity", func(t *testing.T) {
		ctx := context.Background()

		// Create a provider
		provider := createTestProvider(t, domain.ProviderEnabled)
		require.NoError(t, providerRepo.Create(ctx, provider))

		// Create an agent type
		agentType := createTestAgentType(t)
		require.NoError(t, agentTypeRepo.Create(ctx, agentType))

		// Create an agent
		agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
		require.NoError(t, agentRepo.Create(ctx, agent))

		// Execute the Exists method
		exists, err := agentRepo.Exists(ctx, agent.ID)

		// Assert
		require.NoError(t, err)
		assert.True(t, exists, "Should return true for an existing entity ID")
	})

	t.Run("success - returns false for non-existent entity", func(t *testing.T) {
		ctx := context.Background()

		// Generate a random UUID that should not exist in the database
		nonExistentID := domain.NewUUID()

		// Execute the Exists method
		exists, err := agentRepo.Exists(ctx, nonExistentID)

		// Assert
		require.NoError(t, err)
		assert.False(t, exists, "Should return false for a non-existent entity ID")
	})
}
