package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestAuditEntryRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewAuditEntryRepository(testDB.DB)

	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		auditEntry := &domain.AuditEntry{
			AuthorityType: "agent",
			AuthorityID:   "test-agent-id",
			Type:          "status_change",
			Properties: domain.JSON{
				"old_status": "new",
				"new_status": "active",
			},
		}

		err := repo.Create(ctx, auditEntry)
		require.NoError(t, err)
		assert.NotEmpty(t, auditEntry.ID)
		assert.NotZero(t, auditEntry.CreatedAt)
		assert.NotZero(t, auditEntry.UpdatedAt)
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple audit entries
		entries := []struct {
			authorityType string
			authorityID   string
			entryType     string
		}{
			{"agent", "agent-1", "status_change"},
			{"service", "service-1", "config_update"},
			{"agent", "agent-2", "status_change"},
		}

		for _, e := range entries {
			entry := &domain.AuditEntry{
				AuthorityType: e.authorityType,
				AuthorityID:   e.authorityID,
				Type:          e.entryType,
				Properties: domain.JSON{
					"test": "data",
				},
			}
			err := repo.Create(ctx, entry)
			require.NoError(t, err)
		}

		// Test listing with pagination
		result, err := repo.List(ctx, nil, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 3)

		// Test listing with authorityType filter
		result, err = repo.List(ctx, &domain.SimpleFilter{Field: "authorityType", Value: "agent"}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 2)
		for _, item := range result.Items {
			assert.Equal(t, "agent", item.AuthorityType)
		}

		// Test listing with type filter
		result, err = repo.List(ctx, &domain.SimpleFilter{Field: "type", Value: "status_change"}, nil, &domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Items), 2)
		for _, item := range result.Items {
			assert.Equal(t, "status_change", item.Type)
		}
	})
}
