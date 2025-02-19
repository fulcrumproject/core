package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instances
	agentRepo := NewAgentRepository(tdb.DB)
	providerRepo := NewProviderRepository(tdb.DB)
	agentTypeRepo := NewAgentTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, providerRepo.Create(ctx, provider))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)

			// Execute
			err := agentRepo.Create(ctx, agent)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, agent.ID)

			// Verify in database
			found, err := agentRepo.FindByID(ctx, agent.ID)
			require.NoError(t, err)
			assert.Equal(t, agent.Name, found.Name)
			assert.Equal(t, agent.State, found.State)
			assert.Equal(t, agent.TokenHash, found.TokenHash)
			assert.Equal(t, agent.CountryCode, found.CountryCode)
			assert.Equal(t, agent.Attributes, found.Attributes)
			assert.Equal(t, agent.Properties, found.Properties)
			assert.Equal(t, agent.ProviderID, found.ProviderID)
			assert.Equal(t, agent.AgentTypeID, found.AgentTypeID)
			assert.NotNil(t, found.Provider)
			assert.NotNil(t, found.AgentType)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, providerRepo.Create(ctx, provider))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent1 := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
			require.NoError(t, agentRepo.Create(ctx, agent1))
			agent2 := createTestAgent(t, provider.ID, agentType.ID, domain.AgentConnected)
			require.NoError(t, agentRepo.Create(ctx, agent2))

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := agentRepo.List(ctx, nil, nil, pagination)

			// Assert
			agents := result.Items
			require.NoError(t, err)
			assert.Greater(t, len(agents), 0)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			filter := &domain.SimpleFilter{
				Field: "state",
				Value: "Connected",
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := agentRepo.List(ctx, filter, nil, pagination)

			// Assert
			require.NoError(t, err)
			for _, a := range result.Items {
				assert.Equal(t, domain.AgentConnected, a.State)
			}
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, providerRepo.Create(ctx, provider))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent1 := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
			agent1.Name = "A Agent"
			require.NoError(t, agentRepo.Create(ctx, agent1))

			agent2 := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
			agent2.Name = "B Agent"
			require.NoError(t, agentRepo.Create(ctx, agent2))

			sorting := &domain.Sorting{
				Field: "name",
				Order: "desc",
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := agentRepo.List(ctx, nil, sorting, pagination)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			assert.GreaterOrEqual(t, result.Items[0].Name, result.Items[1].Name)
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, providerRepo.Create(ctx, provider))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			// Create multiple agents
			for i := 0; i < 5; i++ {
				agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
				require.NoError(t, agentRepo.Create(ctx, agent))
			}

			pagination := &domain.Pagination{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := agentRepo.List(ctx, nil, nil, pagination)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			pagination.Page = 2
			result, err = agentRepo.List(ctx, nil, nil, pagination)

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
			require.NoError(t, providerRepo.Create(ctx, provider))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
			require.NoError(t, agentRepo.Create(ctx, agent))

			// Read
			agent, err := agentRepo.FindByID(ctx, agent.ID)
			require.NoError(t, err)

			// Update agent
			agent.Name = "Updated Agent"
			agent.State = domain.AgentConnected
			agent.CountryCode = "UK"
			agent.Attributes = domain.Attributes{"new_key": []string{"new_value"}}
			agent.Properties = map[string]interface{}{"new_prop": "value"}

			// Execute
			err = agentRepo.Save(ctx, agent)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := agentRepo.FindByID(ctx, agent.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Agent", updated.Name)
			assert.Equal(t, domain.AgentConnected, updated.State)
			assert.Equal(t, "UK", updated.CountryCode)
			assert.Equal(t, domain.Attributes{"new_key": []string{"new_value"}}, updated.Attributes)
			assert.Equal(t, domain.JSON(map[string]interface{}{"new_prop": "value"}), updated.Properties)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			provider := createTestProvider(t, domain.ProviderEnabled)
			require.NoError(t, providerRepo.Create(ctx, provider))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentNew)
			require.NoError(t, agentRepo.Create(ctx, agent))

			// Execute
			err := agentRepo.Delete(ctx, agent.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := agentRepo.FindByID(ctx, agent.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}
