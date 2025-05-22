package database

import (
	"context"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewTokenRepository(tdb.DB)

	// Setup initial data for scope IDs
	participant := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, NewParticipantRepository(tdb.DB).Create(context.Background(), participant))

	// Create an agent type and agent for agent token tests
	agentTypeRepo := NewAgentTypeRepository(tdb.DB)
	agentType := createTestAgentType(t)
	require.NoError(t, agentTypeRepo.Create(context.Background(), agentType))

	agentRepo := NewAgentRepository(tdb.DB)
	agent := createTestAgent(t, participant.ID, agentType.ID, domain.AgentConnected)
	require.NoError(t, agentRepo.Create(context.Background(), agent))

	t.Run("Create", func(t *testing.T) {
		t.Run("success - admin token", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleFulcrumAdmin, nil)

			// Execute
			err := repo.Create(ctx, token)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, token.ID)
			assert.NotEmpty(t, token.HashedValue)

			// Verify in database
			found, err := repo.FindByID(ctx, token.ID)
			require.NoError(t, err)
			assert.Equal(t, token.Name, found.Name)
			assert.Equal(t, token.HashedValue, found.HashedValue)
			assert.Equal(t, token.Role, found.Role)
			assert.Nil(t, found.ParticipantID)
			assert.Nil(t, found.AgentID)
		})

		t.Run("success - participant token", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleParticipant, &participant.ID)

			// Execute
			err := repo.Create(ctx, token)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, token.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, token.ID)
			require.NoError(t, err)
			assert.Equal(t, token.Name, found.Name)
			assert.Equal(t, token.Role, found.Role)
			assert.NotNil(t, found.ParticipantID)
			assert.Equal(t, participant.ID, *found.ParticipantID)
		})

		t.Run("success - consumer token", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleParticipant, &participant.ID)

			// Execute
			err := repo.Create(ctx, token)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, token.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, token.ID)
			require.NoError(t, err)
			assert.Equal(t, token.Name, found.Name)
			assert.Equal(t, token.Role, found.Role)
			assert.NotNil(t, found.ParticipantID)
			assert.Equal(t, participant.ID, *found.ParticipantID)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token1 := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, token1))
			token2 := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, token2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			token.Name = "UniqueTokenName"
			require.NoError(t, repo.Create(ctx, token))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"UniqueTokenName"}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			found := false
			for _, item := range result.Items {
				if item.Name == "UniqueTokenName" {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected to find the token with the filtered name")
		})

		t.Run("success - list with role filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleParticipant, &participant.ID)
			require.NoError(t, repo.Create(ctx, token))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"role": {"participant"}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			for _, item := range result.Items {
				assert.Equal(t, domain.RoleParticipant, item.Role)
			}
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token1 := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			token1.Name = "A Token"
			require.NoError(t, repo.Create(ctx, token1))

			token2 := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			token2.Name = "B Token"
			require.NoError(t, repo.Create(ctx, token2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, token))

			// Read
			token, err := repo.FindByID(ctx, token.ID)
			require.NoError(t, err)

			// Update token
			token.Name = "Updated Token"
			newExpiry := time.Now().Add(48 * time.Hour)
			token.ExpireAt = newExpiry

			// Execute
			err = repo.Save(ctx, token)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := repo.FindByID(ctx, token.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Token", updated.Name)
			assert.WithinDuration(t, newExpiry, updated.ExpireAt, time.Second)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, token))

			// Execute
			err := repo.Delete(ctx, token.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := repo.FindByID(ctx, token.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("FindByHashedValue", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, token))

			// Execute
			found, err := repo.FindByHashedValue(ctx, token.HashedValue)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, token.ID, found.ID)
			assert.Equal(t, token.HashedValue, found.HashedValue)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.FindByHashedValue(ctx, "nonexistent-hash")

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("DeleteByAgentID", func(t *testing.T) {
		t.Run("success - deletes tokens with matching agent ID", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create agent tokens
			// For agent tokens, we need a valid agent with valid participant ID
			agentToken1 := &domain.Token{
				Name:          "Agent Token 1",
				Role:          domain.RoleAgent,
				HashedValue:   "agent-token-hash-1",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				AgentID:       &agent.ID,
				ParticipantID: &participant.ID, // Agent tokens need participant ID too
			}
			require.NoError(t, agentToken1.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, agentToken1))

			agentToken2 := &domain.Token{
				Name:          "Agent Token 2",
				Role:          domain.RoleAgent,
				HashedValue:   "agent-token-hash-2",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				AgentID:       &agent.ID,
				ParticipantID: &participant.ID,
			}
			require.NoError(t, agentToken2.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, agentToken2))

			// Create a token with a different role that shouldn't be affected
			otherToken := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, otherToken))

			// Execute
			err := repo.DeleteByAgentID(ctx, agent.ID)

			// Assert
			require.NoError(t, err)

			// Verify agent tokens are deleted
			_, err = repo.FindByID(ctx, agentToken1.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Agent token 1 should be deleted")

			_, err = repo.FindByID(ctx, agentToken2.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Agent token 2 should be deleted")

			// Verify other tokens are not affected
			otherFound, err := repo.FindByID(ctx, otherToken.ID)
			assert.NoError(t, err)
			assert.NotNil(t, otherFound, "Other token should not be affected")
		})
	})

	t.Run("DeleteByParticipantID for participant", func(t *testing.T) {
		t.Run("success - deletes tokens with matching participant ID", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create participant participant tokens
			participantToken1 := &domain.Token{
				Name:          "Provider Token 1",
				Role:          domain.RoleParticipant,
				HashedValue:   "participant-token-hash-1",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				ParticipantID: &participant.ID,
			}
			require.NoError(t, participantToken1.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, participantToken1))

			participantToken2 := &domain.Token{
				Name:          "Provider Token 2",
				Role:          domain.RoleParticipant,
				HashedValue:   "participant-token-hash-2",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				ParticipantID: &participant.ID,
			}
			require.NoError(t, participantToken2.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, participantToken2))

			// Create a token with a different role that shouldn't be affected
			otherToken := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, otherToken))

			// Execute
			err := repo.DeleteByParticipantID(ctx, participant.ID)

			// Assert
			require.NoError(t, err)

			// Verify participant tokens are deleted
			_, err = repo.FindByID(ctx, participantToken1.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Provider token 1 should be deleted")

			_, err = repo.FindByID(ctx, participantToken2.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Provider token 2 should be deleted")

			// Verify other tokens are not affected
			otherFound, err := repo.FindByID(ctx, otherToken.ID)
			assert.NoError(t, err)
			assert.NotNil(t, otherFound, "Other token should not be affected")
		})
	})

	t.Run("DeleteByParticipantID for consumer", func(t *testing.T) {
		t.Run("success - deletes tokens with matching participant ID", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create consumer participant tokens
			consumerToken1 := &domain.Token{
				Name:          "Consumer Token 1",
				Role:          domain.RoleParticipant,
				HashedValue:   "consumer-token-hash-1",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				ParticipantID: &participant.ID,
			}
			require.NoError(t, consumerToken1.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, consumerToken1))

			consumerToken2 := &domain.Token{
				Name:          "Consumer Token 2",
				Role:          domain.RoleParticipant,
				HashedValue:   "consumer-token-hash-2",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				ParticipantID: &participant.ID,
			}
			require.NoError(t, consumerToken2.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, consumerToken2))

			// Create a token with a different role that shouldn't be affected
			otherToken := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, otherToken))

			// Execute
			err := repo.DeleteByParticipantID(ctx, participant.ID)

			// Assert
			require.NoError(t, err)

			// Verify consumer tokens are deleted
			_, err = repo.FindByID(ctx, consumerToken1.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Consumer token 1 should be deleted")

			_, err = repo.FindByID(ctx, consumerToken2.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Consumer token 2 should be deleted")

			// Verify other tokens are not affected
			otherFound, err := repo.FindByID(ctx, otherToken.ID)
			assert.NoError(t, err)
			assert.NotNil(t, otherFound, "Other token should not be affected")
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns correct auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create tokens with different roles and scope IDs
			adminToken := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, adminToken))

			participantToken := &domain.Token{
				Name:          "Participant Scope Test",
				Role:          domain.RoleParticipant,
				HashedValue:   "participant-token-hash-scope",
				ExpireAt:      time.Now().Add(24 * time.Hour),
				ParticipantID: &participant.ID,
			}
			require.NoError(t, participantToken.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, participantToken))

			// For auth scope tests, test different types of tokens
			// Admin token (empty scope)
			adminScope, err := repo.AuthScope(ctx, adminToken.ID)
			require.NoError(t, err)
			assert.Nil(t, adminScope.ParticipantID, "Admin token should have nil participant ID")
			assert.Nil(t, adminScope.AgentID, "Admin token should have nil agent ID")

			// Participant token (participant scope)
			participantScope, err := repo.AuthScope(ctx, participantToken.ID)
			require.NoError(t, err)
			assert.NotNil(t, participantScope.ParticipantID, "Participant token should have participant ID")
			assert.Equal(t, participant.ID, *participantScope.ParticipantID, "Participant ID should match")
			assert.Nil(t, participantScope.AgentID, "Participant token should have nil agent ID")
		})
	})
}
