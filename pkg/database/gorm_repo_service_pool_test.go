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

func createTestServicePool(t *testing.T, poolSetID properties.UUID) *domain.ServicePool {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ServicePool{
		Name:             fmt.Sprintf("Test Pool %s", randomSuffix),
		Type:             "ipv4",
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: poolSetID,
	}
}

func TestServicePoolRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	repo := NewServicePoolRepository(tdb.DB)
	poolSetRepo := NewServicePoolSetRepository(tdb.DB)
	participantRepo := NewParticipantRepository(tdb.DB)

	// Create shared participant and pool set
	ctx := context.Background()
	participant := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(ctx, participant))
	poolSet := createTestServicePoolSet(t, participant.ID)
	require.NoError(t, poolSetRepo.Create(ctx, poolSet))

	t.Run("create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)

			err := repo.Create(ctx, pool)

			require.NoError(t, err)
			assert.NotEmpty(t, pool.ID)

			found, err := repo.Get(ctx, pool.ID)
			require.NoError(t, err)
			assert.Equal(t, pool.Name, found.Name)
			assert.Equal(t, pool.Type, found.Type)
			assert.Equal(t, pool.GeneratorType, found.GeneratorType)
			assert.Equal(t, pool.ServicePoolSetID, found.ServicePoolSetID)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool))

			found, err := repo.Get(ctx, pool.ID)

			require.NoError(t, err)
			assert.Equal(t, pool.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			found, err := repo.Get(ctx, uuid.New())

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			pool1 := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool1))
			pool2 := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - filter by name", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {pool.Name}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, pool.Name, result.Items[0].Name)
		})

		t.Run("success - filter by name case insensitive", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			pool.Name = "UniquePoolName-CaseTest"
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"uniquepoolname-casetest"}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, "UniquePoolName-CaseTest", result.Items[0].Name)
		})

		t.Run("success - filter by type", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			pool.Type = "ipv6"
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"type": {"ipv6"}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			for _, item := range result.Items {
				assert.Equal(t, "ipv6", item.Type)
			}
		})

		t.Run("success - filter by generatorType", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			pool.GeneratorType = domain.PoolGeneratorSubnet
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"generatorType": {"subnet"}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			for _, item := range result.Items {
				assert.Equal(t, domain.PoolGeneratorSubnet, item.GeneratorType)
			}
		})

		t.Run("success - filter by servicePoolSetId", func(t *testing.T) {
			ctx := context.Background()

			// Create a separate pool set
			otherPoolSet := createTestServicePoolSet(t, participant.ID)
			require.NoError(t, poolSetRepo.Create(ctx, otherPoolSet))

			pool := createTestServicePool(t, otherPoolSet.ID)
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"servicePoolSetId": {otherPoolSet.ID.String()}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, otherPoolSet.ID, result.Items[0].ServicePoolSetID)
		})

		t.Run("success - with sorting", func(t *testing.T) {
			ctx := context.Background()

			pool1 := createTestServicePool(t, poolSet.ID)
			pool1.Name = "A Pool"
			require.NoError(t, repo.Create(ctx, pool1))

			pool2 := createTestServicePool(t, poolSet.ID)
			pool2.Name = "B Pool"
			require.NoError(t, repo.Create(ctx, pool2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false,
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
		})

		t.Run("success - with pagination", func(t *testing.T) {
			ctx := context.Background()

			for i := 0; i < 5; i++ {
				pool := createTestServicePool(t, poolSet.ID)
				require.NoError(t, repo.Create(ctx, pool))
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			page.Page = 2
			result, err = repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})

	t.Run("ListByPoolSet", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Create a separate pool set for isolation
			isolatedPoolSet := createTestServicePoolSet(t, participant.ID)
			require.NoError(t, poolSetRepo.Create(ctx, isolatedPoolSet))

			pool1 := createTestServicePool(t, isolatedPoolSet.ID)
			require.NoError(t, repo.Create(ctx, pool1))
			pool2 := createTestServicePool(t, isolatedPoolSet.ID)
			require.NoError(t, repo.Create(ctx, pool2))

			result, err := repo.ListByPoolSet(ctx, isolatedPoolSet.ID)

			require.NoError(t, err)
			assert.Len(t, result, 2)
		})
	})

	t.Run("FindByPoolSetAndType", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			pool.Type = fmt.Sprintf("unique-type-%s", uuid.New().String())
			require.NoError(t, repo.Create(ctx, pool))

			found, err := repo.FindByPoolSetAndType(ctx, poolSet.ID, pool.Type)

			require.NoError(t, err)
			assert.Equal(t, pool.ID, found.ID)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			found, err := repo.FindByPoolSetAndType(ctx, poolSet.ID, "nonexistent-type")

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool))

			pool, err := repo.Get(ctx, pool.ID)
			require.NoError(t, err)

			pool.Name = "Updated Pool Name"
			err = repo.Update(ctx, pool)

			require.NoError(t, err)

			found, err := repo.Get(ctx, pool.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Pool Name", found.Name)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool))

			err := repo.Delete(ctx, pool.ID)

			require.NoError(t, err)

			found, err := repo.Get(ctx, pool.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns provider scope", func(t *testing.T) {
			ctx := context.Background()

			pool := createTestServicePool(t, poolSet.ID)
			require.NoError(t, repo.Create(ctx, pool))

			scope, err := repo.AuthScope(ctx, pool.ID)

			require.NoError(t, err)
			defaultScope, ok := scope.(*authz.DefaultObjectScope)
			require.True(t, ok)
			assert.NotNil(t, defaultScope.ProviderID)
			assert.Equal(t, participant.ID, *defaultScope.ProviderID)
		})
	})
}
