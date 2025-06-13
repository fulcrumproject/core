package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGormStore(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	store := NewGormStore(testDB.DB)

	// Test that store is not nil
	assert.NotNil(t, store, "Store should not be nil")

	// Test all repo methods return non-nil repositories
	assert.NotNil(t, store.ParticipantRepo(), "ParticipantRepo should not be nil")
	assert.NotNil(t, store.TokenRepo(), "TokenRepo should not be nil")
	assert.NotNil(t, store.AgentTypeRepo(), "AgentTypeRepo should not be nil")
	assert.NotNil(t, store.AgentRepo(), "AgentRepo should not be nil")
	assert.NotNil(t, store.ServiceTypeRepo(), "ServiceTypeRepo should not be nil")
	assert.NotNil(t, store.ServiceGroupRepo(), "ServiceGroupRepo should not be nil")
	assert.NotNil(t, store.ServiceRepo(), "ServiceRepo should not be nil")
	assert.NotNil(t, store.JobRepo(), "JobRepo should not be nil")
	assert.NotNil(t, store.EventRepo(), "EventRepo should not be nil")
	assert.NotNil(t, store.MetricTypeRepo(), "MetricTypeRepo should not be nil")
	assert.NotNil(t, store.MetricEntryRepo(), "MetricEntryRepo should not be nil")
}

func TestGormStore_Atomic(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	store := NewGormStore(testDB.DB)

	// Test Atomic method
	err := store.Atomic(context.Background(), func(txStore domain.Store) error {
		// Verify transaction store is not nil
		assert.NotNil(t, txStore, "Transaction store should not be nil")

		// Verify all repositories in the transaction store are not nil
		assert.NotNil(t, txStore.ParticipantRepo(), "Transaction ParticipantRepo should not be nil")
		assert.NotNil(t, txStore.TokenRepo(), "Transaction TokenRepo should not be nil")
		assert.NotNil(t, txStore.AgentTypeRepo(), "Transaction AgentTypeRepo should not be nil")
		assert.NotNil(t, txStore.AgentRepo(), "Transaction AgentRepo should not be nil")
		assert.NotNil(t, txStore.ServiceTypeRepo(), "Transaction ServiceTypeRepo should not be nil")
		assert.NotNil(t, txStore.ServiceGroupRepo(), "Transaction ServiceGroupRepo should not be nil")
		assert.NotNil(t, txStore.ServiceRepo(), "Transaction ServiceRepo should not be nil")
		assert.NotNil(t, txStore.JobRepo(), "Transaction JobRepo should not be nil")
		assert.NotNil(t, txStore.EventRepo(), "Transaction EventRepo should not be nil")
		assert.NotNil(t, txStore.MetricTypeRepo(), "Transaction MetricTypeRepo should not be nil")
		assert.NotNil(t, txStore.MetricEntryRepo(), "Transaction MetricEntryRepo should not be nil")

		return nil
	})

	assert.NoError(t, err, "Atomic transaction should not return an error")
}
