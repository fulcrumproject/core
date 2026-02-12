package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestServicePoolSet(t *testing.T, providerID properties.UUID) *domain.ServicePoolSet {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ServicePoolSet{
		Name:       fmt.Sprintf("Test Pool Set %s", randomSuffix),
		ProviderID: providerID,
	}
}

func TestServicePoolSetRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServicePoolSetRepository(testDB.DB)

	// Create a provider participant as FK dependency
	participantRepo := NewParticipantRepository(testDB.DB)
	provider := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(context.Background(), provider))

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			poolSet := createTestServicePoolSet(t, provider.ID)

			err := repo.Create(ctx, poolSet)

			require.NoError(t, err)
			assert.NotEmpty(t, poolSet.ID)
			assert.NotZero(t, poolSet.CreatedAt)
			assert.NotZero(t, poolSet.UpdatedAt)

			found, err := repo.Get(ctx, poolSet.ID)
			require.NoError(t, err)
			assert.Equal(t, poolSet.Name, found.Name)
			assert.Equal(t, provider.ID, found.ProviderID)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			poolSet := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet))

			found, err := repo.Get(ctx, poolSet.ID)

			require.NoError(t, err)
			assert.Equal(t, poolSet.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			found, err := repo.Get(ctx, properties.NewUUID())

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()
			poolSet1 := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet1))
			poolSet2 := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet2))

			page := &domain.PageReq{Page: 1, PageSize: 10}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
		})
	})

	t.Run("FindByProvider", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			poolSet := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet))

			results, err := repo.FindByProvider(ctx, provider.ID)

			require.NoError(t, err)
			assert.Greater(t, len(results), 0)
		})
	})

	t.Run("FindByProviderAndName", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			poolSet := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet))

			found, err := repo.FindByProviderAndName(ctx, provider.ID, poolSet.Name)

			require.NoError(t, err)
			assert.Equal(t, poolSet.ID, found.ID)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			found, err := repo.FindByProviderAndName(ctx, provider.ID, "nonexistent")

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			poolSet := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet))

			poolSet.Name = "Updated Pool Set"
			err := repo.Update(ctx, poolSet)

			require.NoError(t, err)

			found, err := repo.Get(ctx, poolSet.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Pool Set", found.Name)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()
			poolSet := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet))

			err := repo.Delete(ctx, poolSet.ID)

			require.NoError(t, err)

			found, err := repo.Get(ctx, poolSet.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns provider-only auth scope", func(t *testing.T) {
			ctx := context.Background()

			poolSet := createTestServicePoolSet(t, provider.ID)
			require.NoError(t, repo.Create(ctx, poolSet))

			scope, err := repo.AuthScope(ctx, poolSet.ID)

			require.NoError(t, err)
			assert.NotNil(t, scope, "AuthScope should not return nil")

			// Verify all 4 scopeFields are correctly mapped
			defaultScope, ok := scope.(*authz.DefaultObjectScope)
			require.True(t, ok, "AuthScope should return a authz.DefaultObjectScope")
			assert.Nil(t, defaultScope.ParticipantID, "ParticipantID should be nil for service pool sets")
			assert.NotNil(t, defaultScope.ProviderID, "ProviderID should not be nil")
			assert.Equal(t, provider.ID, *defaultScope.ProviderID, "ProviderID should match the provider's ID")
			assert.Nil(t, defaultScope.AgentID, "AgentID should be nil for service pool sets")
			assert.Nil(t, defaultScope.ConsumerID, "ConsumerID should be nil for service pool sets")
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			nonExistentScope, err := repo.AuthScope(ctx, properties.NewUUID())

			require.Error(t, err)
			assert.Nil(t, nonExistentScope)
		})
	})
}
