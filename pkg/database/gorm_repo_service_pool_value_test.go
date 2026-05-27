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

func createTestServicePoolValue(t *testing.T, poolID properties.UUID) *domain.ServicePoolValue {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ServicePoolValue{
		Name:          fmt.Sprintf("Value %s", randomSuffix),
		Value:         fmt.Sprintf("10.0.0.%s", randomSuffix[:3]),
		ServicePoolID: poolID,
	}
}

func TestServicePoolValueRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	participantRepo := NewParticipantRepository(tdb.DB)
	poolSetRepo := NewServicePoolSetRepository(tdb.DB)
	poolRepo := NewServicePoolRepository(tdb.DB)
	repo := NewServicePoolValueRepository(tdb.DB)
	ctx := context.Background()

	// Shared participant + pool set + pool for FK
	participant := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(ctx, participant))
	poolSet := createTestServicePoolSet(t, participant.ID)
	require.NoError(t, poolSetRepo.Create(ctx, poolSet))
	pool := createTestServicePool(t, poolSet.ID)
	pool.ParticipantID = &participant.ID
	require.NoError(t, poolRepo.Create(ctx, pool))

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			value := createTestServicePoolValue(t, pool.ID)

			err := repo.Create(ctx, value)

			require.NoError(t, err)
			assert.NotEmpty(t, value.ID)

			found, err := repo.Get(ctx, value.ID)
			require.NoError(t, err)
			assert.Equal(t, value.Name, found.Name)
			assert.Equal(t, value.Value, found.Value)
			assert.Equal(t, pool.ID, found.ServicePoolID)
			assert.Nil(t, found.ServiceID)
			assert.Nil(t, found.PropertyName)
			assert.Nil(t, found.AllocatedAt)
			assert.Nil(t, found.ParticipantID)
		})

		t.Run("success with participant", func(t *testing.T) {
			value := createTestServicePoolValue(t, pool.ID)
			value.ParticipantID = &participant.ID
			require.NoError(t, repo.Create(ctx, value))

			found, err := repo.Get(ctx, value.ID)
			require.NoError(t, err)
			require.NotNil(t, found.ParticipantID, "participant_id should round-trip")
			assert.Equal(t, participant.ID, *found.ParticipantID)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			value := createTestServicePoolValue(t, pool.ID)
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
			v1 := createTestServicePoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, v1))
			v2 := createTestServicePoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, v2))

			page := &domain.PageReq{Page: 1, PageSize: 100}
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - filter by name", func(t *testing.T) {
			value := createTestServicePoolValue(t, pool.ID)
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

		t.Run("success - filter by servicePoolId", func(t *testing.T) {
			pool2 := createTestServicePool(t, poolSet.ID)
			pool2.Type = fmt.Sprintf("type-%s", uuid.New().String())
			pool2.ParticipantID = &participant.ID
			require.NoError(t, poolRepo.Create(ctx, pool2))

			value := createTestServicePoolValue(t, pool2.ID)
			require.NoError(t, repo.Create(ctx, value))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"servicePoolId": {pool2.ID.String()}},
			}
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, pool2.ID, result.Items[0].ServicePoolID)
		})

		t.Run("success - filter by serviceId", func(t *testing.T) {
			serviceID := properties.NewUUID()
			value := createTestServicePoolValue(t, pool.ID)
			value.Allocate(serviceID, "publicIp")
			require.NoError(t, repo.Create(ctx, value))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"serviceId": {serviceID.String()}},
			}
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			require.NotNil(t, result.Items[0].ServiceID)
			assert.Equal(t, serviceID, *result.Items[0].ServiceID)
		})

		t.Run("success - with sorting", func(t *testing.T) {
			uniquePool := createTestServicePool(t, poolSet.ID)
			uniquePool.Type = fmt.Sprintf("sort-type-%s", uuid.New().String())
			uniquePool.ParticipantID = &participant.ID
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			v1 := createTestServicePoolValue(t, uniquePool.ID)
			v1.Name = "A Value"
			require.NoError(t, repo.Create(ctx, v1))

			v2 := createTestServicePoolValue(t, uniquePool.ID)
			v2.Name = "B Value"
			require.NoError(t, repo.Create(ctx, v2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 100,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false,
				Filters:  map[string][]string{"servicePoolId": {uniquePool.ID.String()}},
			}
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
		})

		t.Run("success - with pagination", func(t *testing.T) {
			uniquePool := createTestServicePool(t, poolSet.ID)
			uniquePool.Type = fmt.Sprintf("page-type-%s", uuid.New().String())
			uniquePool.ParticipantID = &participant.ID
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			for range 5 {
				v := createTestServicePoolValue(t, uniquePool.ID)
				require.NoError(t, repo.Create(ctx, v))
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
				Filters:  map[string][]string{"servicePoolId": {uniquePool.ID.String()}},
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
			value := createTestServicePoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			serviceID := properties.NewUUID()
			value.Allocate(serviceID, "publicIp")
			require.NoError(t, repo.Update(ctx, value))

			found, err := repo.Get(ctx, value.ID)
			require.NoError(t, err)
			assert.True(t, found.IsAllocated())
			assert.Equal(t, &serviceID, found.ServiceID)

			found.Release()
			require.NoError(t, repo.Update(ctx, found))

			found, err = repo.Get(ctx, value.ID)
			require.NoError(t, err)
			assert.False(t, found.IsAllocated())
			assert.Nil(t, found.ServiceID)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			value := createTestServicePoolValue(t, pool.ID)
			require.NoError(t, repo.Create(ctx, value))

			require.NoError(t, repo.Delete(ctx, value.ID))

			found, err := repo.Get(ctx, value.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("FindAvailable", func(t *testing.T) {
		t.Run("success - returns unallocated values", func(t *testing.T) {
			uniquePool := createTestServicePool(t, poolSet.ID)
			uniquePool.Type = fmt.Sprintf("avail-type-%s", uuid.New().String())
			uniquePool.ParticipantID = &participant.ID
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			available := createTestServicePoolValue(t, uniquePool.ID)
			require.NoError(t, repo.Create(ctx, available))

			allocated := createTestServicePoolValue(t, uniquePool.ID)
			allocated.Allocate(properties.NewUUID(), "publicIp")
			require.NoError(t, repo.Create(ctx, allocated))

			values, err := repo.FindAvailable(ctx, uniquePool.ID)

			require.NoError(t, err)
			require.Len(t, values, 1)
			assert.Equal(t, available.ID, values[0].ID)
		})

		t.Run("success - empty when all allocated", func(t *testing.T) {
			uniquePool := createTestServicePool(t, poolSet.ID)
			uniquePool.Type = fmt.Sprintf("all-alloc-type-%s", uuid.New().String())
			uniquePool.ParticipantID = &participant.ID
			require.NoError(t, poolRepo.Create(ctx, uniquePool))

			v := createTestServicePoolValue(t, uniquePool.ID)
			v.Allocate(properties.NewUUID(), "publicIp")
			require.NoError(t, repo.Create(ctx, v))

			values, err := repo.FindAvailable(ctx, uniquePool.ID)

			require.NoError(t, err)
			assert.Empty(t, values)
		})
	})

	t.Run("FindByService", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			serviceID := properties.NewUUID()

			v1 := createTestServicePoolValue(t, pool.ID)
			v1.Allocate(serviceID, "publicIp")
			require.NoError(t, repo.Create(ctx, v1))

			v2 := createTestServicePoolValue(t, pool.ID)
			v2.Allocate(serviceID, "internalIp")
			require.NoError(t, repo.Create(ctx, v2))

			otherServiceID := properties.NewUUID()
			v3 := createTestServicePoolValue(t, pool.ID)
			v3.Allocate(otherServiceID, "publicIp")
			require.NoError(t, repo.Create(ctx, v3))

			values, err := repo.FindByService(ctx, serviceID)

			require.NoError(t, err)
			assert.Len(t, values, 2)
			for _, v := range values {
				assert.Equal(t, &serviceID, v.ServiceID)
			}
		})

		t.Run("success - empty when none allocated", func(t *testing.T) {
			values, err := repo.FindByService(ctx, properties.NewUUID())

			require.NoError(t, err)
			assert.Empty(t, values)
		})
	})

	t.Run("CountByPool", func(t *testing.T) {
		uniquePool := createTestServicePool(t, poolSet.ID)
		uniquePool.Type = fmt.Sprintf("count-type-%s", uuid.New().String())
		uniquePool.ParticipantID = &participant.ID
		require.NoError(t, poolRepo.Create(ctx, uniquePool))

		for range 3 {
			v := createTestServicePoolValue(t, uniquePool.ID)
			require.NoError(t, repo.Create(ctx, v))
		}

		count, err := repo.CountByPool(ctx, uniquePool.ID)

		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("ReleaseByService", func(t *testing.T) {
		serviceID := properties.NewUUID()

		v1 := createTestServicePoolValue(t, pool.ID)
		v1.Allocate(serviceID, "publicIp")
		require.NoError(t, repo.Create(ctx, v1))

		v2 := createTestServicePoolValue(t, pool.ID)
		v2.Allocate(serviceID, "internalIp")
		require.NoError(t, repo.Create(ctx, v2))

		require.NoError(t, repo.ReleaseByService(ctx, serviceID))

		for _, id := range []properties.UUID{v1.ID, v2.ID} {
			found, err := repo.Get(ctx, id)
			require.NoError(t, err)
			assert.False(t, found.IsAllocated())
			assert.Nil(t, found.ServiceID)
			assert.Nil(t, found.PropertyName)
			assert.Nil(t, found.AllocatedAt)
		}
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("returns DefaultObjectScope with denormalized ParticipantID", func(t *testing.T) {
			value := createTestServicePoolValue(t, pool.ID)
			value.ParticipantID = &participant.ID
			require.NoError(t, repo.Create(ctx, value))

			scope, err := repo.AuthScope(ctx, value.ID)

			require.NoError(t, err)
			def, ok := scope.(*authz.DefaultObjectScope)
			require.True(t, ok, "expected *DefaultObjectScope, got %T", scope)
			require.NotNil(t, def.ParticipantID)
			assert.Equal(t, participant.ID, *def.ParticipantID)
		})

		t.Run("not found returns NotFoundError", func(t *testing.T) {
			_, err := repo.AuthScope(ctx, properties.NewUUID())
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}
