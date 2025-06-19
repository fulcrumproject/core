package database

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/domain"
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
			found, err := agentRepo.Get(ctx, agent.ID)
			require.NoError(t, err)
			assert.Equal(t, agent.Name, found.Name)
			assert.Equal(t, agent.Status, found.Status)
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
			result, err := agentRepo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			// Verify Participant is preloaded but not AgentType (as per repository config)
			assert.NotNil(t, result.Items[0].Provider)
			assert.Nil(t, result.Items[0].AgentType)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"status": {"Connected"}},
			}

			// Execute
			result, err := agentRepo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			for _, a := range result.Items {
				assert.Equal(t, domain.AgentConnected, a.Status)
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
			result, err := agentRepo.List(ctx, &auth.IdentityScope{}, page)

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
			result, err := agentRepo.List(ctx, &auth.IdentityScope{}, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = agentRepo.List(ctx, &auth.IdentityScope{}, page)

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
			agent, err := agentRepo.Get(ctx, agent.ID)
			require.NoError(t, err)

			// Update agent
			agent.Name = "Updated Agent"
			agent.Status = domain.AgentConnected

			// Execute
			err = agentRepo.Save(ctx, agent)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := agentRepo.Get(ctx, agent.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Agent", updated.Name)
			assert.Equal(t, domain.AgentConnected, updated.Status)
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
			found, err := agentRepo.Get(ctx, agent.ID)
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

			// Verify the statuss of all agents
			found, err := agentRepo.Get(ctx, recentAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.AgentConnected, found.Status, "Recent agent should still be connected")

			found, err = agentRepo.Get(ctx, oldAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.AgentDisconnected, found.Status, "Old agent should be disconnected")

			found, err = agentRepo.Get(ctx, discoAgent.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.AgentDisconnected, found.Status, "Disconnected agent should remain disconnected")
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
			count, err := agentRepo.CountByProvider(ctx, participant.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, expectedCount, count, "Should return the correct count of agents")

			// Execute count for the participant with no agents
			emptyCount, err := agentRepo.CountByProvider(ctx, emptyParticipant.ID)

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

			// Check that the returned scope is a auth.DefaultObjectScope
			defaultScope, ok := scope.(*auth.DefaultObjectScope)
			require.True(t, ok, "AuthScope should return a auth.DefaultObjectScope")
			assert.NotNil(t, defaultScope.ProviderID, "ProviderID should not be nil")
			assert.Equal(t, participant.ID, *defaultScope.ProviderID, "Should return the participant ID in the scope")
			assert.NotNil(t, defaultScope.AgentID, "AgentID should not be nil")
			assert.Equal(t, agent.ID, *defaultScope.AgentID, "Should return the agent ID in the scope")

			// Test with non-existent agent - checking the actual behavior
			nonExistentID := properties.NewUUID()
			nonExistentScope, err := agentRepo.AuthScope(ctx, nonExistentID)
			require.Error(t, err, "AuthScope should not return an error for non-existent agent")
			assert.Nil(t, nonExistentScope, "Should return an empty auth scope")
		})
	})

	t.Run("FindByServiceTypeAndTags", func(t *testing.T) {
		t.Run("success with matching agents", func(t *testing.T) {
			ctx := context.Background()

			// Setup participants
			participant1 := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant1))
			participant2 := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant2))

			// Setup agent types
			agentType1 := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType1))
			agentType2 := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType2))

			// Setup service types
			serviceTypeRepo := NewServiceTypeRepository(tdb.DB)
			serviceType1 := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType1))
			serviceType2 := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType2))

			// Create agent type service type relationships
			require.NoError(t, tdb.DB.Exec("INSERT INTO agent_type_service_types (agent_type_id, service_type_id) VALUES (?, ?)", agentType1.ID, serviceType1.ID).Error)
			require.NoError(t, tdb.DB.Exec("INSERT INTO agent_type_service_types (agent_type_id, service_type_id) VALUES (?, ?)", agentType2.ID, serviceType1.ID).Error)

			// Setup agents with different tags
			agent1 := createTestAgentWithTags(t, participant1.ID, agentType1.ID, domain.AgentConnected, []string{"tag1", "tag2"})
			require.NoError(t, agentRepo.Create(ctx, agent1))

			agent2 := createTestAgentWithTags(t, participant2.ID, agentType2.ID, domain.AgentConnected, []string{"tag1", "tag3"})
			require.NoError(t, agentRepo.Create(ctx, agent2))

			agent3 := createTestAgentWithTags(t, participant1.ID, agentType1.ID, domain.AgentDisconnected, []string{"tag2", "tag3"})
			require.NoError(t, agentRepo.Create(ctx, agent3))

			// Test case 1: Find agents with tag1 for serviceType1
			agents, err := agentRepo.FindByServiceTypeAndTags(ctx, serviceType1.ID, []string{"tag1"})
			require.NoError(t, err)
			assert.Len(t, agents, 2)

			agentIDs := make([]properties.UUID, len(agents))
			for i, agent := range agents {
				agentIDs[i] = agent.ID
			}
			assert.Contains(t, agentIDs, agent1.ID)
			assert.Contains(t, agentIDs, agent2.ID)

			// Test case 2: Find agents with both tag1 and tag2
			agents, err = agentRepo.FindByServiceTypeAndTags(ctx, serviceType1.ID, []string{"tag1", "tag2"})
			require.NoError(t, err)
			assert.Len(t, agents, 1)
			assert.Equal(t, agent1.ID, agents[0].ID)

			// Test case 3: Find agents with no tags specified
			agents, err = agentRepo.FindByServiceTypeAndTags(ctx, serviceType1.ID, []string{})
			require.NoError(t, err)
			assert.Len(t, agents, 3) // All agents that support serviceType1

			// Test case 4: Find agents with non-existent tag
			agents, err = agentRepo.FindByServiceTypeAndTags(ctx, serviceType1.ID, []string{"nonexistent"})
			require.NoError(t, err)
			assert.Len(t, agents, 0)

			// Test case 5: Find agents for service type that no agent supports
			agents, err = agentRepo.FindByServiceTypeAndTags(ctx, serviceType2.ID, []string{"tag1"})
			require.NoError(t, err)
			assert.Len(t, agents, 0)
		})

		t.Run("success with no matching agents", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			agentType := createTestAgentType(t)
			require.NoError(t, agentTypeRepo.Create(ctx, agentType))

			serviceTypeRepo := NewServiceTypeRepository(tdb.DB)
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			// Create agent type service type relationship
			require.NoError(t, tdb.DB.Exec("INSERT INTO agent_type_service_types (agent_type_id, service_type_id) VALUES (?, ?)", agentType.ID, serviceType.ID).Error)

			agent := createTestAgentWithTags(t, participant.ID, agentType.ID, domain.AgentConnected, []string{"different-tag"})
			require.NoError(t, agentRepo.Create(ctx, agent))

			// Execute - search for agents with tags that don't match
			agents, err := agentRepo.FindByServiceTypeAndTags(ctx, serviceType.ID, []string{"required-tag"})

			// Assert
			require.NoError(t, err)
			assert.Len(t, agents, 0)
		})

		t.Run("success with non-existent service type", func(t *testing.T) {
			ctx := context.Background()

			// Execute - search for agents with non-existent service type
			nonExistentServiceTypeID := properties.NewUUID()
			agents, err := agentRepo.FindByServiceTypeAndTags(ctx, nonExistentServiceTypeID, []string{"tag1"})

			// Assert
			require.NoError(t, err)
			assert.Len(t, agents, 0)
		})
	})
}
