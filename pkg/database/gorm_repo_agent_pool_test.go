package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestAgentPool(t *testing.T) *domain.AgentPool {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.AgentPool{
		Name:          fmt.Sprintf("Test Pool %s", randomSuffix),
		Type:          fmt.Sprintf("ipv4-%s", randomSuffix),
		PropertyType:  "string",
		GeneratorType: domain.PoolGeneratorList,
	}
}

func TestAgentPoolRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	repo := NewAgentPoolRepository(tdb.DB)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			pool := createTestAgentPool(t)

			err := repo.Create(ctx, pool)

			require.NoError(t, err)
			assert.NotEmpty(t, pool.ID)

			found, err := repo.Get(ctx, pool.ID)
			require.NoError(t, err)
			assert.Equal(t, pool.Name, found.Name)
			assert.Equal(t, pool.Type, found.Type)
			assert.Equal(t, pool.PropertyType, found.PropertyType)
			assert.Equal(t, pool.GeneratorType, found.GeneratorType)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			pool := createTestAgentPool(t)
			require.NoError(t, repo.Create(ctx, pool))

			found, err := repo.Get(ctx, pool.ID)

			require.NoError(t, err)
			assert.Equal(t, pool.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			found, err := repo.Get(ctx, uuid.New())

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			pool1 := createTestAgentPool(t)
			require.NoError(t, repo.Create(ctx, pool1))
			pool2 := createTestAgentPool(t)
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
			pool := createTestAgentPool(t)
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
			pool := createTestAgentPool(t)
			pool.Name = "UniqueAgentPoolName-CaseTest"
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"uniqueagentpoolname-casetest"}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, "UniqueAgentPoolName-CaseTest", result.Items[0].Name)
		})

		t.Run("success - filter by type", func(t *testing.T) {
			uniqueType := fmt.Sprintf("unique-type-%s", uuid.New().String())
			pool := createTestAgentPool(t)
			pool.Type = uniqueType
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"type": {uniqueType}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			for _, item := range result.Items {
				assert.Equal(t, uniqueType, item.Type)
			}
		})

		t.Run("success - filter by generatorType", func(t *testing.T) {
			pool := createTestAgentPool(t)
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

		t.Run("success - with sorting", func(t *testing.T) {
			pool1 := createTestAgentPool(t)
			pool1.Name = "A Agent Pool"
			require.NoError(t, repo.Create(ctx, pool1))

			pool2 := createTestAgentPool(t)
			pool2.Name = "B Agent Pool"
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
			for i := 0; i < 5; i++ {
				pool := createTestAgentPool(t)
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

	t.Run("Update", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			pool := createTestAgentPool(t)
			require.NoError(t, repo.Create(ctx, pool))

			pool, err := repo.Get(ctx, pool.ID)
			require.NoError(t, err)

			pool.Name = "Updated Agent Pool Name"
			err = repo.Update(ctx, pool)

			require.NoError(t, err)

			found, err := repo.Get(ctx, pool.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Agent Pool Name", found.Name)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			pool := createTestAgentPool(t)
			require.NoError(t, repo.Create(ctx, pool))

			err := repo.Delete(ctx, pool.ID)

			require.NoError(t, err)

			found, err := repo.Get(ctx, pool.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("FindByType", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			uniqueType := fmt.Sprintf("unique-type-%s", uuid.New().String())
			pool := createTestAgentPool(t)
			pool.Type = uniqueType
			require.NoError(t, repo.Create(ctx, pool))

			found, err := repo.FindByType(ctx, uniqueType)

			require.NoError(t, err)
			assert.Equal(t, pool.ID, found.ID)
			assert.Equal(t, uniqueType, found.Type)
		})

		t.Run("not found", func(t *testing.T) {
			found, err := repo.FindByType(ctx, "nonexistent_type")

			assert.Nil(t, found)
			assert.IsType(t, domain.NotFoundError{}, err)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns always match scope", func(t *testing.T) {
			pool := createTestAgentPool(t)
			require.NoError(t, repo.Create(ctx, pool))

			scope, err := repo.AuthScope(ctx, pool.ID)

			require.NoError(t, err)
			_, ok := scope.(*authz.AllwaysMatchObjectScope)
			require.True(t, ok)
		})
	})
}
