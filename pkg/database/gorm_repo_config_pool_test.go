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

func createTestConfigPool(t *testing.T) *domain.ConfigPool {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ConfigPool{
		Name:          fmt.Sprintf("Test Pool %s", randomSuffix),
		Type:          fmt.Sprintf("ipv4-%s", randomSuffix),
		PropertyType:  "string",
		GeneratorType: domain.PoolGeneratorList,
	}
}

func TestConfigPoolRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	repo := NewConfigPoolRepository(tdb.DB)
	participantRepo := NewParticipantRepository(tdb.DB)
	ctx := context.Background()

	// Seed a participant once for the per-participant scope cases below.
	scopedParticipant := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(ctx, scopedParticipant))

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			pool := createTestConfigPool(t)

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
			pool := createTestConfigPool(t)
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
			pool1 := createTestConfigPool(t)
			require.NoError(t, repo.Create(ctx, pool1))
			pool2 := createTestConfigPool(t)
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
			pool := createTestConfigPool(t)
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
			pool := createTestConfigPool(t)
			pool.Name = "UniqueConfigPoolName-CaseTest"
			require.NoError(t, repo.Create(ctx, pool))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"uniqueconfigpoolname-casetest"}},
			}

			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, "UniqueConfigPoolName-CaseTest", result.Items[0].Name)
		})

		t.Run("success - filter by type", func(t *testing.T) {
			uniqueType := fmt.Sprintf("unique-type-%s", uuid.New().String())
			pool := createTestConfigPool(t)
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
			pool := createTestConfigPool(t)
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
			pool1 := createTestConfigPool(t)
			pool1.Name = "A Agent Pool"
			require.NoError(t, repo.Create(ctx, pool1))

			pool2 := createTestConfigPool(t)
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
				pool := createTestConfigPool(t)
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
			pool := createTestConfigPool(t)
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
			pool := createTestConfigPool(t)
			require.NoError(t, repo.Create(ctx, pool))

			err := repo.Delete(ctx, pool.ID)

			require.NoError(t, err)

			found, err := repo.Get(ctx, pool.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("FindByTypeAndParticipant", func(t *testing.T) {
		t.Run("success - global scope", func(t *testing.T) {
			uniqueType := fmt.Sprintf("unique-type-%s", uuid.New().String())
			pool := createTestConfigPool(t)
			pool.Type = uniqueType
			require.NoError(t, repo.Create(ctx, pool))

			found, err := repo.FindByTypeAndParticipant(ctx, uniqueType, nil)

			require.NoError(t, err)
			assert.Equal(t, pool.ID, found.ID)
			assert.Equal(t, uniqueType, found.Type)
		})

		t.Run("success - participant scope", func(t *testing.T) {
			uniqueType := fmt.Sprintf("unique-type-%s", uuid.New().String())
			pool := createTestConfigPool(t)
			pool.Type = uniqueType
			pool.ParticipantID = &scopedParticipant.ID
			require.NoError(t, repo.Create(ctx, pool))

			// Global lookup must NOT find a participant-owned pool.
			missing, err := repo.FindByTypeAndParticipant(ctx, uniqueType, nil)
			assert.Nil(t, missing)
			assert.IsType(t, domain.NotFoundError{}, err)

			// Owning-participant lookup finds it.
			found, err := repo.FindByTypeAndParticipant(ctx, uniqueType, &scopedParticipant.ID)
			require.NoError(t, err)
			assert.Equal(t, pool.ID, found.ID)
			assert.Equal(t, &scopedParticipant.ID, found.ParticipantID)
		})

		t.Run("two participants may share the same type", func(t *testing.T) {
			sharedType := fmt.Sprintf("shared-type-%s", uuid.New().String())
			other := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, other))

			poolA := createTestConfigPool(t)
			poolA.Type = sharedType
			poolA.ParticipantID = &scopedParticipant.ID
			require.NoError(t, repo.Create(ctx, poolA))

			poolB := createTestConfigPool(t)
			poolB.Type = sharedType
			poolB.ParticipantID = &other.ID
			require.NoError(t, repo.Create(ctx, poolB))

			gotA, err := repo.FindByTypeAndParticipant(ctx, sharedType, &scopedParticipant.ID)
			require.NoError(t, err)
			assert.Equal(t, poolA.ID, gotA.ID)

			gotB, err := repo.FindByTypeAndParticipant(ctx, sharedType, &other.ID)
			require.NoError(t, err)
			assert.Equal(t, poolB.ID, gotB.ID)
		})

		t.Run("not found", func(t *testing.T) {
			found, err := repo.FindByTypeAndParticipant(ctx, "nonexistent_type", nil)

			assert.Nil(t, found)
			assert.IsType(t, domain.NotFoundError{}, err)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		cases := []struct {
			name            string
			participantID   *properties.UUID
			expectAdminOnly bool
		}{
			{name: "global pool returns admin-only scope", participantID: nil, expectAdminOnly: true},
			{name: "participant-owned pool returns default scope", participantID: &scopedParticipant.ID, expectAdminOnly: false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				pool := createTestConfigPool(t)
				pool.ParticipantID = tc.participantID
				require.NoError(t, repo.Create(ctx, pool))

				scope, err := repo.AuthScope(ctx, pool.ID)
				require.NoError(t, err)

				if tc.expectAdminOnly {
					_, ok := scope.(authz.AdminOnlyObjectScope)
					require.True(t, ok, "expected AdminOnlyObjectScope for nil participant_id")
					return
				}
				def, ok := scope.(*authz.DefaultObjectScope)
				require.True(t, ok, "expected *DefaultObjectScope for set participant_id, got %T", scope)
				require.NotNil(t, def.ParticipantID)
				assert.Equal(t, *tc.participantID, *def.ParticipantID)
			})
		}
	})
}
