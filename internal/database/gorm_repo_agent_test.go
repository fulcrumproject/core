package database

import (
	"context"
	"testing"
	"time"

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
	participantRepo := NewParticipantRepository(tdb.DB)
	agentTypeRepo := NewAgentTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)

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
			assert.Equal(t, agent.CountryCode, found.CountryCode)
			assert.Equal(t, agent.Attributes, found.Attributes)
			assert.Equal(t, agent.ParticipantID, found.ParticipantID)
			assert.Equal(t, agent.AgentTypeID, found.AgentTypeID)
			assert.NotNil(t, found.Participant)
			assert.NotNil(t, found.AgentType)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent1 := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
			require.NoError(t, agentRepo.Create(ctx, agent1))
			agent2 := createTestAgent(t, participant.ID, agentType.ID, domain.AgentConnected)
			require.NoError(t, agentRepo.Create(ctx, agent2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := agentRepo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			// Verify Participant is preloaded but not AgentType (as per repository config)
			assert.NotNil(t, result.Items[0].Participant)
			assert.Nil(t, result.Items[0].AgentType)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"state": {"Connected"}},
			}

			// Execute
			result, err := agentRepo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			for _, a := range result.Items {
				assert.Equal(t, domain.AgentConnected, a.State)
			}
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent1 := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
			agent1.Name = "A Agent"
			require.NoError(t, agentRepo.Create(ctx, agent1))

			agent2 := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
			agent2.Name = "B Agent"
			require.NoError(t, agentRepo.Create(ctx, agent2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false,
			}

			// Execute
			result, err := agentRepo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			assert.GreaterOrEqual(t, result.Items[0].Name, result.Items[1].Name)
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			// Create multiple agents
			for i := 0; i < 5; i++ {
				agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
				require.NoError(t, agentRepo.Create(ctx, agent))
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := agentRepo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = agentRepo.List(ctx, &domain.EmptyAuthIdentityScope, page)

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
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
			require.NoError(t, agentRepo.Create(ctx, agent))

			// Read
			agent, err := agentRepo.FindByID(ctx, agent.ID)
			require.NoError(t, err)

			// Update agent
			agent.Name = "Updated Agent"
			agent.State = domain.AgentConnected
			agent.CountryCode = "UK"
			agent.Attributes = domain.Attributes{"new_key": []string{"new_value"}}

			// Execute
			err = agentRepo.Save(ctx, agent)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := agentRepo.FindByID(ctx, agent.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Agent", updated.Name)
			assert.Equal(t, domain.AgentConnected, updated.State)
			assert.Equal(t, domain.CountryCode("UK"), updated.CountryCode)
			assert.Equal(t, domain.Attributes{"new_key": []string{"new_value"}}, updated.Attributes)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
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

	t.Run("MarkInactiveAgentsAsDisconnected", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			// Create a connected agent with recent status update (should NOT be marked as disconnected)
			recentAgent := createTestAgentWithStatusUpdate(t, participant.ID, agentType.ID, domain.AgentConnected, time.Now().Add(-2*time.Minute))
			require.NoError(t, agentRepo.Create(ctx, recentAgent))

			// Create a connected agent with old status update (should be marked as disconnected)
			oldAgent := createTestAgentWithStatusUpdate(t, participant.ID, agentType.ID, domain.AgentConnected, time.Now().Add(-10*time.Minute))
			require.NoError(t, agentRepo.Create(ctx, oldAgent))

			// Create a disconnected agent with old status update (should NOT be marked as disconnected because it's already disconnected)
			discoAgent := createTestAgentWithStatusUpdate(t, participant.ID, agentType.ID, domain.AgentDisconnected, time.Now().Add(-10*time.Minute))
			require.NoError(t, agentRepo.Create(ctx, discoAgent))

			// Execute the method with 5-minute inactive duration
			count, err := agentRepo.MarkInactiveAgentsAsDisconnected(ctx, 5*time.Minute)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, int64(1), count, "Should mark exactly one agent as disconnected")

			// Verify the states of all agents
			found, err := agentRepo.FindByID(ctx, recentAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.AgentConnected, found.State, "Recent agent should still be connected")

			found, err = agentRepo.FindByID(ctx, oldAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.AgentDisconnected, found.State, "Old agent should be disconnected")

			found, err = agentRepo.FindByID(ctx, discoAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.AgentDisconnected, found.State, "Disconnected agent should remain disconnected")
		})

		t.Run("no agents to update", func(t *testing.T) {
			ctx := context.Background()

			// Execute with a very long inactive duration that no agent should match
			count, err := agentRepo.MarkInactiveAgentsAsDisconnected(ctx, 24*time.Hour)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, int64(0), count, "Should not mark any agents as disconnected")
		})
	})

	t.Run("CountByParticipant", func(t *testing.T) {
		t.Run("success - returns correct count", func(t *testing.T) {
			ctx := context.Background()

			// Create a participant
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			// Create a participant with no agents (to test zero count)
			emptyParticipant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, emptyParticipant))

			// Create an agent type
			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			// Create multiple agents for our test provider
			expectedCount := int64(3)
			for i := 0; i < int(expectedCount); i++ {
				agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
				require.NoError(t, agentRepo.Create(ctx, agent))
			}

			// Execute count for the participant with agents
			count, err := agentRepo.CountByParticipant(ctx, participant.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, expectedCount, count, "Should return the correct count of agents")

			// Execute count for the participant with no agents
			emptyCount, err := agentRepo.CountByParticipant(ctx, emptyParticipant.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, int64(0), emptyCount, "Should return zero for provider with no agents")
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns correct auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a participant
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			// Create an agent type
			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			// Create an agent
			agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentNew)
			require.NoError(t, agentRepo.Create(ctx, agent))

			// Execute
			scope, err := agentRepo.AuthScope(ctx, agent.ID)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, scope, "AuthScope should not return nil")
			assert.NotNil(t, scope.ParticipantID, "ParticipantID should not be nil")
			assert.Equal(t, participant.ID, *scope.ParticipantID, "Should return the participant ID in the scope")
			assert.NotNil(t, scope.AgentID, "AgentID should not be nil")
			assert.Equal(t, agent.ID, *scope.AgentID, "Should return the agent ID in the scope")

			// Test with non-existent agent - checking the actual behavior
			nonExistentID := domain.NewUUID()
			nonExistentScope, err := agentRepo.AuthScope(ctx, nonExistentID)
			require.NoError(t, err, "AuthScope should not return an error for non-existent agent")
			assert.NotNil(t, nonExistentScope, "Should return an empty auth scope")
			assert.Equal(t, &domain.AuthScope{}, nonExistentScope, "Should return empty auth scope for non-existent agent")
		})
	})
}
