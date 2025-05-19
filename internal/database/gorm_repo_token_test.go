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
	provider := createTestProvider(t, domain.ProviderEnabled)
	require.NoError(t, NewProviderRepository(tdb.DB).Create(context.Background(), provider))

	broker := createTestBroker(t)
	require.NoError(t, NewBrokerRepository(tdb.DB).Create(context.Background(), broker))

	// Create an agent type and agent for agent token tests
	agentTypeRepo := NewAgentTypeRepository(tdb.DB)
	agentType := createTestAgentType(t)
	require.NoError(t, agentTypeRepo.Create(context.Background(), agentType))

	agentRepo := NewAgentRepository(tdb.DB)
	agent := createTestAgent(t, provider.ID, agentType.ID, domain.AgentConnected)
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
			assert.Nil(t, found.ProviderID)
			assert.Nil(t, found.BrokerID)
			assert.Nil(t, found.AgentID)
		})

		t.Run("success - provider admin token", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleProviderAdmin, &provider.ID)

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
			assert.NotNil(t, found.ProviderID)
			assert.Equal(t, provider.ID, *found.ProviderID)
		})

		t.Run("success - broker token", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token := createTestToken(t, domain.RoleBroker, &broker.ID)

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
			assert.NotNil(t, found.BrokerID)
			assert.Equal(t, broker.ID, *found.BrokerID)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			token1 := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, token1))
			token2 := createTestToken(t, domain.RoleFulcrumAdmin, &provider.ID)
			require.NoError(t, repo.Create(ctx, token2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

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
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

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
			token := createTestToken(t, domain.RoleBroker, &broker.ID)
			require.NoError(t, repo.Create(ctx, token))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"role": {"broker"}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			for _, item := range result.Items {
				assert.Equal(t, domain.RoleBroker, item.Role)
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
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

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
			// For agent tokens, we need a valid agent with valid provider ID
			agentToken1 := &domain.Token{
				Name:        "Agent Token 1",
				Role:        domain.RoleAgent,
				HashedValue: "agent-token-hash-1",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				AgentID:     &agent.ID,
				ProviderID:  &provider.ID, // Agent tokens need provider ID too
			}
			require.NoError(t, agentToken1.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, agentToken1))

			agentToken2 := &domain.Token{
				Name:        "Agent Token 2",
				Role:        domain.RoleAgent,
				HashedValue: "agent-token-hash-2",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				AgentID:     &agent.ID,
				ProviderID:  &provider.ID,
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

	t.Run("DeleteByProviderID", func(t *testing.T) {
		t.Run("success - deletes tokens with matching provider ID", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create provider tokens
			providerToken1 := &domain.Token{
				Name:        "Provider Token 1",
				Role:        domain.RoleProviderAdmin,
				HashedValue: "provider-token-hash-1",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				ProviderID:  &provider.ID,
			}
			require.NoError(t, providerToken1.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, providerToken1))

			providerToken2 := &domain.Token{
				Name:        "Provider Token 2",
				Role:        domain.RoleProviderAdmin,
				HashedValue: "provider-token-hash-2",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				ProviderID:  &provider.ID,
			}
			require.NoError(t, providerToken2.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, providerToken2))

			// Create a token with a different role that shouldn't be affected
			otherToken := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, otherToken))

			// Execute
			err := repo.DeleteByProviderID(ctx, provider.ID)

			// Assert
			require.NoError(t, err)

			// Verify provider tokens are deleted
			_, err = repo.FindByID(ctx, providerToken1.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Provider token 1 should be deleted")

			_, err = repo.FindByID(ctx, providerToken2.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Provider token 2 should be deleted")

			// Verify other tokens are not affected
			otherFound, err := repo.FindByID(ctx, otherToken.ID)
			assert.NoError(t, err)
			assert.NotNil(t, otherFound, "Other token should not be affected")
		})
	})

	t.Run("DeleteByBrokerID", func(t *testing.T) {
		t.Run("success - deletes tokens with matching broker ID", func(t *testing.T) {
			ctx := context.Background()

			// Setup - create broker tokens
			brokerToken1 := &domain.Token{
				Name:        "Broker Token 1",
				Role:        domain.RoleBroker,
				HashedValue: "broker-token-hash-1",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				BrokerID:    &broker.ID,
			}
			require.NoError(t, brokerToken1.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, brokerToken1))

			brokerToken2 := &domain.Token{
				Name:        "Broker Token 2",
				Role:        domain.RoleBroker,
				HashedValue: "broker-token-hash-2",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				BrokerID:    &broker.ID,
			}
			require.NoError(t, brokerToken2.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, brokerToken2))

			// Create a token with a different role that shouldn't be affected
			otherToken := createTestToken(t, domain.RoleFulcrumAdmin, nil)
			require.NoError(t, repo.Create(ctx, otherToken))

			// Execute
			err := repo.DeleteByBrokerID(ctx, broker.ID)

			// Assert
			require.NoError(t, err)

			// Verify broker tokens are deleted
			_, err = repo.FindByID(ctx, brokerToken1.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Broker token 1 should be deleted")

			_, err = repo.FindByID(ctx, brokerToken2.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{}, "Broker token 2 should be deleted")

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

			providerToken := &domain.Token{
				Name:        "Provider Scope Test",
				Role:        domain.RoleProviderAdmin,
				HashedValue: "provider-token-hash-scope",
				ExpireAt:    time.Now().Add(24 * time.Hour),
				ProviderID:  &provider.ID,
			}
			require.NoError(t, providerToken.GenerateTokenValue())
			require.NoError(t, repo.Create(ctx, providerToken))

			// For auth scope tests, test different types of tokens
			// Admin token (empty scope)
			adminScope, err := repo.AuthScope(ctx, adminToken.ID)
			require.NoError(t, err)
			assert.Nil(t, adminScope.ParticipantID, "Admin token should have nil provider ID")
			assert.Nil(t, adminScope.BrokerID, "Admin token should have nil broker ID")
			assert.Nil(t, adminScope.AgentID, "Admin token should have nil agent ID")

			// Provider token (provider scope)
			providerScope, err := repo.AuthScope(ctx, providerToken.ID)
			require.NoError(t, err)
			assert.NotNil(t, providerScope.ParticipantID, "Provider token should have provider ID")
			assert.Equal(t, provider.ID, *providerScope.ParticipantID, "Provider ID should match")
			assert.Nil(t, providerScope.BrokerID, "Provider token should have nil broker ID")
			assert.Nil(t, providerScope.AgentID, "Provider token should have nil agent ID")
		})
	})
}
