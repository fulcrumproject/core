package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGormMetricStore(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	// Create metric store
	metricStore := NewGormMetricStore(testDB.MetricDB)

	t.Run("MetricEntryRepo", func(t *testing.T) {
		repo := metricStore.MetricEntryRepo()
		assert.NotNil(t, repo)
		assert.Implements(t, (*domain.MetricEntryRepository)(nil), repo)
	})

	t.Run("GetMetricDb", func(t *testing.T) {
		db := metricStore.GetMetricDb()
		assert.NotNil(t, db)
		assert.Equal(t, testDB.MetricDB, db)
	})

	t.Run("Atomic", func(t *testing.T) {
		ctx := context.Background()
		err := metricStore.Atomic(ctx, func(txStore domain.MetricStore) error {
			// Test that we can access the repository within a transaction
			_, err := txStore.MetricEntryRepo().Count(ctx)
			return err
		})
		require.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		// Create a new store for testing close
		store := NewGormMetricStore(testDB.MetricDB)
		err := store.Close()
		require.NoError(t, err)
	})
}
