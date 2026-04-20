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

func createTestAgentPoolValue(t *testing.T, poolID properties.UUID) *domain.AgentPoolValue {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.AgentPoolValue{
		Name:        fmt.Sprintf("Value %s", randomSuffix),
		Value:       fmt.Sprintf("192.168.1.%s", randomSuffix[:3]),
		AgentPoolID: poolID,
	}
}

func TestAgentPoolValueRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	poolRepo := NewAgentPoolRepository(tdb.DB)
	repo := NewAgentPoolValueRepository(tdb.DB)
	ctx := context.Background()

	// Create a parent pool for FK
	pool := createTestAgentPool(t)
	require.NoError(t, poolRepo.Create(ctx, pool))

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			value := createTestAgentPoolValue(t, pool.ID)

			err := repo.Create(ctx, value)

			require.NoError(t, err)
			assert.NotEmpty(t, value.ID)

			found, err := repo.Get(ctx, value.ID)
			require.NoError(t, err)
			assert.Equal(t, value.Name, found.Name)
			assert.Equal(t, value.Value, found.Value)
			assert.Equal(t, pool.ID, found.AgentPoolID)
			assert.Nil(t, found.AgentID)
			assert.Nil(t, found.PropertyName)
			assert.Nil(t, found.AllocatedAt)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			value := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			found, err := repo.Get(ctx, value.ID)

			require.NoError(t, err)
			assert.Equal(t, value.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			found, err := repo.Get(ctx, uuid.New())

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			value1 := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value1))
			value2 := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 100,
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - filter by name", func(t *testing.T) {
			value := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {value.Name}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, value.Name, result.Items[0].Name)
		})

		t.Run("success - filter by agentPoolId", func(t *testing.T) {
			// Create a second pool
			pool2 := createTestAgentPool(t)
			pool2.Type = fmt.Sprintf("type-%s", uuid.New().String())
			require.NoError(t, poolRepo.Create(ctx, pool2))

			value := createTestAgentPoolValue(t, pool2.ID)
			require.NoError(t, repo.Create(ctx, value))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"agentPoolId": {pool2.ID.String()}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, pool2.ID, result.Items[0].AgentPoolID)
		})

		t.Run("success - filter by agentId", func(t *testing.T) {
			agentID := properties.NewUUID()
			value := createTestAgentPoolValue(t, pool.ID)
			value.AgentID = &agentID
			propName := "test_prop"
			value.PropertyName = &propName
			require.NoError(t, repo.Create(ctx, value))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"agentId": {agentID.String()}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, &agentID, result.Items[0].AgentID)
		})

		t.Run("success - with sorting", func(t *testing.T) {
			v1 := createTestAgentPoolValue(t, pool.ID)
			v1.Name = "A Value"
			require.NoError(t, repo.Create(ctx, v1))

			v2 := createTestAgentPoolValue(t, pool.ID)
			v2.Name = "B Value"
			require.NoError(t, repo.Create(ctx, v2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 100,
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
			// Create enough values for pagination
			uniquePool := createTestAgentPool(t)
			uniquePool.Type = fmt.Sprintf("pagination-type-%s", uuid.New().String())
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			for i := 0; i < 5; i++ {
				v := createTestAgentPoolValue(t, uniquePool.ID)
				require.NoError(t, repo.Create(ctx, v))
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
				Filters:  map[string][]string{"agentPoolId": {uniquePool.ID.String()}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)

			page.Page = 2
			result, err = repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasPrev)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("success - allocate and release", func(t *testing.T) {
			value := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			// Allocate
			agentID := properties.NewUUID()
			value.Allocate(agentID, "ip_address")
			err := repo.Update(ctx, value)
			require.NoError(t, err)

			found, err := repo.Get(ctx, value.ID)
			require.NoError(t, err)
			assert.True(t, found.IsAllocated())
			assert.Equal(t, &agentID, found.AgentID)

			// Release
			found.Release()
			err = repo.Update(ctx, found)
			require.NoError(t, err)

			found, err = repo.Get(ctx, value.ID)
			require.NoError(t, err)
			assert.False(t, found.IsAllocated())
			assert.Nil(t, found.AgentID)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			value := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			err := repo.Delete(ctx, value.ID)

			require.NoError(t, err)

			found, err := repo.Get(ctx, value.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("FindAvailable", func(t *testing.T) {
		t.Run("success - returns unallocated values", func(t *testing.T) {
			uniquePool := createTestAgentPool(t)
			uniquePool.Type = fmt.Sprintf("avail-type-%s", uuid.New().String())
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			available := createTestAgentPoolValue(t, uniquePool.ID)
			require.NoError(t, repo.Create(ctx, available))

			allocated := createTestAgentPoolValue(t, uniquePool.ID)
			agentID := properties.NewUUID()
			allocated.Allocate(agentID, "prop")
			require.NoError(t, repo.Create(ctx, allocated))

			values, err := repo.FindAvailable(ctx, uniquePool.ID)

			require.NoError(t, err)
			require.Len(t, values, 1)
			assert.Equal(t, available.ID, values[0].ID)
		})

		t.Run("success - empty when all allocated", func(t *testing.T) {
			uniquePool := createTestAgentPool(t)
			uniquePool.Type = fmt.Sprintf("all-alloc-type-%s", uuid.New().String())
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			allocated := createTestAgentPoolValue(t, uniquePool.ID)
			agentID := properties.NewUUID()
			allocated.Allocate(agentID, "prop")
			require.NoError(t, repo.Create(ctx, allocated))

			values, err := repo.FindAvailable(ctx, uniquePool.ID)

			require.NoError(t, err)
			assert.Empty(t, values)
		})
	})

	t.Run("FindByAgent", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			agentID := properties.NewUUID()

			v1 := createTestAgentPoolValue(t, pool.ID)
			v1.Allocate(agentID, "prop1")
			require.NoError(t, repo.Create(ctx, v1))

			v2 := createTestAgentPoolValue(t, pool.ID)
			v2.Allocate(agentID, "prop2")
			require.NoError(t, repo.Create(ctx, v2))

			// Different agent
			otherAgentID := properties.NewUUID()
			v3 := createTestAgentPoolValue(t, pool.ID)
			v3.Allocate(otherAgentID, "prop3")
			require.NoError(t, repo.Create(ctx, v3))

			values, err := repo.FindByAgent(ctx, agentID)

			require.NoError(t, err)
			assert.Len(t, values, 2)
			for _, v := range values {
				assert.Equal(t, &agentID, v.AgentID)
			}
		})

		t.Run("success - empty when none allocated", func(t *testing.T) {
			values, err := repo.FindByAgent(ctx, properties.NewUUID())

			require.NoError(t, err)
			assert.Empty(t, values)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns always match scope", func(t *testing.T) {
			value := createTestAgentPoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			scope, err := repo.AuthScope(ctx, value.ID)

			require.NoError(t, err)
			_, ok := scope.(*authz.AllwaysMatchObjectScope)
			require.True(t, ok)
		})
	})
}
