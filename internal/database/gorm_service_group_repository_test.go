package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestServiceGroupRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceGroupRepository(testDB.DB)

	t.Run("Create", func(t *testing.T) {
		serviceGroup := &domain.ServiceGroup{
			Name: "Test Group",
		}

		err := repo.Create(context.Background(), serviceGroup)
		require.NoError(t, err)
		assert.NotEmpty(t, serviceGroup.ID)
		assert.NotZero(t, serviceGroup.CreatedAt)
		assert.NotZero(t, serviceGroup.UpdatedAt)
	})

	t.Run("FindByID", func(t *testing.T) {
		// Create a service group
		serviceGroup := &domain.ServiceGroup{
			Name: "Test Group",
		}
		err := repo.Create(context.Background(), serviceGroup)
		require.NoError(t, err)

		// Find the service group
		found, err := repo.FindByID(context.Background(), serviceGroup.ID)
		require.NoError(t, err)
		assert.Equal(t, serviceGroup.ID, found.ID)
		assert.Equal(t, serviceGroup.Name, found.Name)
	})

	t.Run("FindByID_NotFound", func(t *testing.T) {
		found, err := repo.FindByID(context.Background(), domain.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a service group
		serviceGroup := &domain.ServiceGroup{
			Name: "Test Group",
		}
		err := repo.Create(context.Background(), serviceGroup)
		require.NoError(t, err)

		// Update the service group
		serviceGroup.Name = "Updated Group"
		err = repo.Save(context.Background(), serviceGroup)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.FindByID(context.Background(), serviceGroup.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Group", found.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a service group
		serviceGroup := &domain.ServiceGroup{
			Name: "Test Group",
		}
		err := repo.Create(context.Background(), serviceGroup)
		require.NoError(t, err)

		// Delete the service group
		err = repo.Delete(context.Background(), serviceGroup.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByID(context.Background(), serviceGroup.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple service groups
		groups := []*domain.ServiceGroup{
			{Name: "Group A"},
			{Name: "Group B"},
			{Name: "Group C"},
		}
		for _, group := range groups {
			err := repo.Create(context.Background(), group)
			require.NoError(t, err)
		}

		// Test listing with pagination
		result, err := repo.List(context.Background(), nil, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 2)

		// Test listing with filter
		result, err = repo.List(context.Background(), &domain.SimpleFilter{Field: "name", Value: "Group A"}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "Group A", result.Items[0].Name)
	})

	t.Run("Count", func(t *testing.T) {
		// Create multiple service groups
		groups := []*domain.ServiceGroup{
			{Name: "Count A"},
			{Name: "Count B"},
			{Name: "Count C"},
		}
		for _, group := range groups {
			err := repo.Create(context.Background(), group)
			require.NoError(t, err)
		}

		// Test count without filter
		count, err := repo.Count(context.Background(), nil)
		require.NoError(t, err)
		assert.Greater(t, count, int64(2))

		// Test count with filter
		count, err = repo.Count(context.Background(), &domain.SimpleFilter{Field: "name", Value: "Count A"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}
