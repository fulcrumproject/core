package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfrastructureTypeRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	repo := NewInfrastructureTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			it := createTestInfrastructureType(t)
			err := repo.Create(ctx, it)
			require.NoError(t, err)
			assert.NotEmpty(t, it.ID)

			found, err := repo.Get(ctx, it.ID)
			require.NoError(t, err)
			assert.Equal(t, it.Name, found.Name)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			it := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it))

			found, err := repo.Get(ctx, it.ID)
			require.NoError(t, err)
			assert.Equal(t, it.Name, found.Name)
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

			it1 := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it1))
			it2 := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it2))

			page := &domain.PageReq{Page: 1, PageSize: 10}
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			ctx := context.Background()

			it := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {it.Name}},
			}
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, it.Name, result.Items[0].Name)
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			it1 := createTestInfrastructureType(t)
			it1.Name = "A Infra Type"
			require.NoError(t, repo.Create(ctx, it1))

			it2 := createTestInfrastructureType(t)
			it2.Name = "B Infra Type"
			require.NoError(t, repo.Create(ctx, it2))

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

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			for i := 0; i < 5; i++ {
				it := createTestInfrastructureType(t)
				require.NoError(t, repo.Create(ctx, it))
			}

			page := &domain.PageReq{Page: 1, PageSize: 2}
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
			assert.True(t, result.HasPrev)

			count, err := repo.Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, result.TotalItems, count)
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("updates fields", func(t *testing.T) {
			ctx := context.Background()

			it := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it))

			it.Name = "Renamed Infra Type"
			require.NoError(t, repo.Save(ctx, it))

			found, err := repo.Get(ctx, it.ID)
			require.NoError(t, err)
			assert.Equal(t, "Renamed Infra Type", found.Name)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			it := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it))

			require.NoError(t, repo.Delete(ctx, it.ID))

			_, err := repo.Get(ctx, it.ID)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("returns AllwaysMatchObjectScope (global resource)", func(t *testing.T) {
			ctx := context.Background()

			it := createTestInfrastructureType(t)
			require.NoError(t, repo.Create(ctx, it))

			scope, err := repo.AuthScope(ctx, it.ID)
			require.NoError(t, err)
			assert.NotNil(t, scope)
			assert.Equal(t, &authz.AllwaysMatchObjectScope{}, scope)
		})
	})
}
