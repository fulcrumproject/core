package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fulcrumproject/core/pkg/domain"
)

func TestMetricTypeRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewMetricTypeRepository(testDB.DB)

	t.Run("create", func(t *testing.T) {
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

	t.Run("Get", func(t *testing.T) {
		// Create a metric type
		metricType := &domain.MetricType{
			Name:       "Memory Usage",
			EntityType: domain.MetricEntityTypeAgent,
		}
		err := repo.Create(context.Background(), metricType)
		require.NoError(t, err)

		// Find the metric type
		found, err := repo.Get(context.Background(), metricType.ID)
		require.NoError(t, err)
		assert.Equal(t, metricType.ID, found.ID)
		assert.Equal(t, metricType.Name, found.Name)
		assert.Equal(t, metricType.EntityType, found.EntityType)
	})

	t.Run("FindByName", func(t *testing.T) {
		// Create a metric type with a unique name
		metricType := &domain.MetricType{
			Name:       "Disk IO Operations",
			EntityType: domain.MetricEntityTypeService,
		}
		err := repo.Create(context.Background(), metricType)
		require.NoError(t, err)

		// Find the metric type by name
		found, err := repo.FindByName(context.Background(), metricType.Name)
		require.NoError(t, err)
		assert.Equal(t, metricType.ID, found.ID)
		assert.Equal(t, metricType.Name, found.Name)
		assert.Equal(t, metricType.EntityType, found.EntityType)
	})

	t.Run("FindByName_NotFound", func(t *testing.T) {
		// Try to find a non-existent metric type
		found, err := repo.FindByName(context.Background(), "NonExistentMetricType")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		found, err := repo.Get(context.Background(), properties.NewUUID())
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
		found, err := repo.Get(context.Background(), metricType.ID)
		require.NoError(t, err)
		assert.Equal(t, "Network Bandwidth", found.Name)
		assert.Equal(t, domain.MetricEntityTypeResource, found.EntityType)
	})

	t.Run("delete", func(t *testing.T) {
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
		found, err := repo.Get(context.Background(), metricType.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple metric types
			metricTypes := []*domain.MetricType{
				{Name: "List CPU Usage", EntityType: domain.MetricEntityTypeService},
				{Name: "List Memory Usage", EntityType: domain.MetricEntityTypeAgent},
				{Name: "List Disk Space", EntityType: domain.MetricEntityTypeResource},
			}
			for _, metricType := range metricTypes {
				err := repo.Create(context.Background(), metricType)
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

		t.Run("success - list with name substring filter", func(t *testing.T) {
			// Create metric types with names containing a common substring
			metricType1 := &domain.MetricType{
				Name:       "FilterTest Network Bandwidth",
				EntityType: domain.MetricEntityTypeService,
			}
			err := repo.Create(context.Background(), metricType1)
			require.NoError(t, err)

			metricType2 := &domain.MetricType{
				Name:       "FilterTest Network Latency",
				EntityType: domain.MetricEntityTypeService,
			}
			err = repo.Create(context.Background(), metricType2)
			require.NoError(t, err)

			other := &domain.MetricType{
				Name:       "FilterTest Disk IOPS",
				EntityType: domain.MetricEntityTypeService,
			}
			err = repo.Create(context.Background(), other)
			require.NoError(t, err)

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"FilterTest Network"}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			names := []string{result.Items[0].Name, result.Items[1].Name}
			assert.ElementsMatch(t, []string{"FilterTest Network Bandwidth", "FilterTest Network Latency"}, names)
		})

		t.Run("success - list with name filter escaping LIKE wildcards", func(t *testing.T) {
			// Create metric types with names containing LIKE wildcard characters
			percentMetric := &domain.MetricType{
				Name:       "50% CPU Usage",
				EntityType: domain.MetricEntityTypeService,
			}
			err := repo.Create(context.Background(), percentMetric)
			require.NoError(t, err)

			underscoreMetric := &domain.MetricType{
				Name:       "user_name_metric",
				EntityType: domain.MetricEntityTypeService,
			}
			err = repo.Create(context.Background(), underscoreMetric)
			require.NoError(t, err)

			backslashMetric := &domain.MetricType{
				Name:       "path\\to\\metric",
				EntityType: domain.MetricEntityTypeService,
			}
			err = repo.Create(context.Background(), backslashMetric)
			require.NoError(t, err)

			// Create a metric type that would match if wildcards weren't escaped
			otherMetric := &domain.MetricType{
				Name:       "50 CPU Usage", // This should NOT match "50% CPU Usage"
				EntityType: domain.MetricEntityTypeService,
			}
			err = repo.Create(context.Background(), otherMetric)
			require.NoError(t, err)

			// Test searching for "50% CPU Usage" - should only match the exact name
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"50% CPU Usage"}},
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, "50% CPU Usage", result.Items[0].Name)

			// Test searching for "user_name_metric" - should only match the exact name
			page = &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"user_name_metric"}},
			}

			result, err = repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, "user_name_metric", result.Items[0].Name)

			// Test searching for "path\\to\\metric" - should only match the exact name
			page = &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {"path\\to\\metric"}},
			}

			result, err = repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, "path\\to\\metric", result.Items[0].Name)
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			// Create multiple metric types
			for i := 0; i < 5; i++ {
				metricType := &domain.MetricType{
					Name:       fmt.Sprintf("Metric %d", i),
					EntityType: domain.MetricEntityTypeService,
				}
				err := repo.Create(context.Background(), metricType)
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
			assert.Greater(t, result.TotalItems, int64(2))

			// Second page
			page.Page = 2
			result, err = repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns empty auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a metric type
			metricType := &domain.MetricType{
				Name:       "Scope Test Metric",
				EntityType: domain.MetricEntityTypeService,
			}
			require.NoError(t, repo.Create(ctx, metricType))

			// Execute with existing metric type ID
			scope, err := repo.AuthScope(ctx, metricType.ID)
			require.NoError(t, err)
			assert.Equal(t, &auth.AllwaysMatchObjectScope{}, scope, "Should return empty auth scope for metric types")
		})
	})
}
