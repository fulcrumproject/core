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
			result, err := repo.List(ctx, page)

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
			result, err := repo.List(ctx, page)

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
			result, err := repo.List(ctx, page)

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
			result, err := repo.List(ctx, page)

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
}
