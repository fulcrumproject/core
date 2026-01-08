package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceOptionTypeRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceOptionTypeRepository(testDB.DB)

	t.Run("Create", func(t *testing.T) {
		optionType := &domain.ServiceOptionType{
			Name:        "Operating System",
			Type:        "os",
			Description: "Available operating systems for VMs",
		}

		err := repo.Create(context.Background(), optionType)
		require.NoError(t, err)
		assert.NotEmpty(t, optionType.ID)
		assert.NotZero(t, optionType.CreatedAt)
		assert.NotZero(t, optionType.UpdatedAt)
	})

	t.Run("Get", func(t *testing.T) {
		// Create a service option type
		optionType := &domain.ServiceOptionType{
			Name:        "Machine Type",
			Type:        "machine_type",
			Description: "Available machine types",
		}
		err := repo.Create(context.Background(), optionType)
		require.NoError(t, err)

		// Get the service option type
		found, err := repo.Get(context.Background(), optionType.ID)
		require.NoError(t, err)
		assert.Equal(t, optionType.ID, found.ID)
		assert.Equal(t, optionType.Name, found.Name)
		assert.Equal(t, optionType.Type, found.Type)
		assert.Equal(t, optionType.Description, found.Description)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		found, err := repo.Get(context.Background(), properties.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("FindByType", func(t *testing.T) {
		// Create a service option type with a unique type
		optionType := &domain.ServiceOptionType{
			Name:        "Disk Type",
			Type:        "disk_type",
			Description: "Available disk types",
		}
		err := repo.Create(context.Background(), optionType)
		require.NoError(t, err)

		// Find the service option type by type
		found, err := repo.FindByType(context.Background(), "disk_type")
		require.NoError(t, err)
		assert.Equal(t, optionType.ID, found.ID)
		assert.Equal(t, optionType.Type, found.Type)
	})

	t.Run("FindByType_NotFound", func(t *testing.T) {
		found, err := repo.FindByType(context.Background(), "nonexistent_type")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a service option type
		optionType := &domain.ServiceOptionType{
			Name:        "Region",
			Type:        "region",
			Description: "Available regions",
		}
		err := repo.Create(context.Background(), optionType)
		require.NoError(t, err)

		// Update the service option type
		optionType.Name = "Geographic Region"
		optionType.Description = "Available geographic regions for deployment"

		err = repo.Save(context.Background(), optionType)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.Get(context.Background(), optionType.ID)
		require.NoError(t, err)
		assert.Equal(t, "Geographic Region", found.Name)
		assert.Equal(t, "Available geographic regions for deployment", found.Description)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a service option type
		optionType := &domain.ServiceOptionType{
			Name:        "Zone",
			Type:        "zone",
			Description: "Available zones",
		}
		err := repo.Create(context.Background(), optionType)
		require.NoError(t, err)

		// Delete the service option type
		err = repo.Delete(context.Background(), optionType.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.Get(context.Background(), optionType.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple service option types
			optionTypes := []*domain.ServiceOptionType{
				{Name: "List OS", Type: "list_os", Description: "OS options"},
				{Name: "List Machine", Type: "list_machine", Description: "Machine options"},
				{Name: "List Disk", Type: "list_disk", Description: "Disk options"},
			}
			for _, optionType := range optionTypes {
				err := repo.Create(context.Background(), optionType)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			// Create multiple service option types
			for i := 0; i < 5; i++ {
				optionType := &domain.ServiceOptionType{
					Name:        "Paginated Option",
					Type:        properties.NewUUID().String(),
					Description: "Test pagination",
				}
				err := repo.Create(context.Background(), optionType)
				require.NoError(t, err)
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
			}

			// First page
			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns empty auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a service option type
			optionType := &domain.ServiceOptionType{
				Name:        "Scope Test",
				Type:        "scope_test",
				Description: "Test auth scope",
			}
			require.NoError(t, repo.Create(ctx, optionType))

			// Execute with existing ID
			scope, err := repo.AuthScope(ctx, optionType.ID)
			require.NoError(t, err)
			assert.Equal(t, &authz.AllwaysMatchObjectScope{}, scope, "Should return empty auth scope for service option types")
		})
	})
}
