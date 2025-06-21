package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTypeRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instances
	repo := NewAgentTypeRepository(tdb.DB)
	serviceTypeRepo := NewServiceTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			agentType := createTestAgentType(t)
			agentType.ServiceTypes = []domain.ServiceType{*serviceType}

			// Execute
			err := repo.Create(ctx, agentType)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, agentType.ID)

			// Verify in database
			found, err := repo.Get(ctx, agentType.ID)
			require.NoError(t, err)
			assert.Equal(t, agentType.Name, found.Name)
			assert.NotEmpty(t, found.ServiceTypes)
			assert.Equal(t, serviceType.ID, found.ServiceTypes[0].ID)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			agentType := createTestAgentType(t)
			agentType.ServiceTypes = []domain.ServiceType{*serviceType}
			require.NoError(t, repo.Create(ctx, agentType))

			// Execute
			found, err := repo.Get(ctx, agentType.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, agentType.Name, found.Name)
			assert.NotEmpty(t, found.ServiceTypes)
			assert.Equal(t, serviceType.ID, found.ServiceTypes[0].ID)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.Get(ctx, uuid.New())

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			agentType1 := createTestAgentType(t)
			agentType1.ServiceTypes = []domain.ServiceType{*serviceType}
			require.NoError(t, repo.Create(ctx, agentType1))

			agentType2 := createTestAgentType(t)
			agentType2.ServiceTypes = []domain.ServiceType{*serviceType}
			require.NoError(t, repo.Create(ctx, agentType2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			// Verify ServiceTypes are preloaded
			assert.NotEmpty(t, result.Items[0].ServiceTypes)
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			agentType := createTestAgentType(t)
			agentType.ServiceTypes = []domain.ServiceType{*serviceType}
			require.NoError(t, repo.Create(ctx, agentType))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {agentType.Name}},
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, agentType.Name, result.Items[0].Name)
			assert.NotEmpty(t, result.Items[0].ServiceTypes)
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			agentType1 := createTestAgentType(t)
			agentType1.Name = "A Agent Type"
			agentType1.ServiceTypes = []domain.ServiceType{*serviceType}
			require.NoError(t, repo.Create(ctx, agentType1))

			agentType2 := createTestAgentType(t)
			agentType2.Name = "B Agent Type"
			agentType2.ServiceTypes = []domain.ServiceType{*serviceType}
			require.NoError(t, repo.Create(ctx, agentType2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
			// Verify ServiceTypes are preloaded
			assert.NotEmpty(t, result.Items[0].ServiceTypes)
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, serviceTypeRepo.Create(ctx, serviceType))

			// Create multiple agent types
			for i := 0; i < 5; i++ {
				agentType := createTestAgentType(t)
				agentType.ServiceTypes = []domain.ServiceType{*serviceType}
				require.NoError(t, repo.Create(ctx, agentType))
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))
			// Verify ServiceTypes are preloaded
			assert.NotEmpty(t, result.Items[0].ServiceTypes)

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)

			// Verify total count matches
			count, err := repo.Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, result.TotalItems, count)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns empty auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			agentType := createTestAgentType(t)
			require.NoError(t, repo.Create(ctx, agentType))

			// Execute with existing agent type ID
			scope, err := repo.AuthScope(ctx, agentType.ID)

			// Assert - should return empty scope for any ID since agent types are global resources
			require.NoError(t, err)
			assert.NotNil(t, scope)
			assert.Equal(t, &auth.AllwaysMatchObjectScope{}, scope, "Should return empty auth scope for agent types")
		})
	})

}
