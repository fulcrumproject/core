package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestMetricTypeRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewMetricTypeRepository(testDB.DB)

	t.Run("Create", func(t *testing.T) {
		metricType := &domain.MetricType{
			Name:       "CPU Usage",
			EntityType: domain.MetricEntityTypeService,
		}

		err := repo.Create(context.Background(), metricType)
		require.NoError(t, err)
		assert.NotEmpty(t, metricType.ID)
		assert.NotZero(t, metricType.CreatedAt)
		assert.NotZero(t, metricType.UpdatedAt)
	})

	t.Run("FindByID", func(t *testing.T) {
		// Create a metric type
		metricType := &domain.MetricType{
			Name:       "Memory Usage",
			EntityType: domain.MetricEntityTypeAgent,
		}
		err := repo.Create(context.Background(), metricType)
		require.NoError(t, err)

		// Find the metric type
		found, err := repo.FindByID(context.Background(), metricType.ID)
		require.NoError(t, err)
		assert.Equal(t, metricType.ID, found.ID)
		assert.Equal(t, metricType.Name, found.Name)
		assert.Equal(t, metricType.EntityType, found.EntityType)
	})

	t.Run("FindByID_NotFound", func(t *testing.T) {
		found, err := repo.FindByID(context.Background(), domain.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Save", func(t *testing.T) {
		// Create a metric type
		metricType := &domain.MetricType{
			Name:       "Network Traffic",
			EntityType: domain.MetricEntityTypeService,
		}
		err := repo.Create(context.Background(), metricType)
		require.NoError(t, err)

		// Update the metric type
		metricType.Name = "Network Bandwidth"
		metricType.EntityType = domain.MetricEntityTypeResource

		err = repo.Save(context.Background(), metricType)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.FindByID(context.Background(), metricType.ID)
		require.NoError(t, err)
		assert.Equal(t, "Network Bandwidth", found.Name)
		assert.Equal(t, domain.MetricEntityTypeResource, found.EntityType)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a metric type
		metricType := &domain.MetricType{
			Name:       "Disk Usage",
			EntityType: domain.MetricEntityTypeResource,
		}
		err := repo.Create(context.Background(), metricType)
		require.NoError(t, err)

		// Delete the metric type
		err = repo.Delete(context.Background(), metricType.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByID(context.Background(), metricType.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple metric types
		metricTypes := []*domain.MetricType{
			{Name: "CPU Usage", EntityType: domain.MetricEntityTypeService},
			{Name: "Memory Usage", EntityType: domain.MetricEntityTypeAgent},
			{Name: "Disk Space", EntityType: domain.MetricEntityTypeResource},
		}
		for _, metricType := range metricTypes {
			err := repo.Create(context.Background(), metricType)
			require.NoError(t, err)
		}

		// Test listing with pagination
		result, err := repo.List(context.Background(), nil, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 3)

		// Test listing with name filter
		result, err = repo.List(context.Background(), &domain.SimpleFilter{Field: "name", Value: "CPU Usage"}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 1)
		assert.Equal(t, "CPU Usage", result.Items[0].Name)
	})
}
