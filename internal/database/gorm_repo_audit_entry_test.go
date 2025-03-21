package database

import (
	"context"
	"testing"
	"time"

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
		t.Run("success", func(t *testing.T) {
			// Setup
			auditEntry := &domain.AuditEntry{
				AuthorityType: domain.AuthorityTypeAgent,
				AuthorityID:   "test-agent-id",
				EventType:     domain.EventTypeStatusChange,
				Properties: domain.JSON{
					"old_status": "new",
					"new_status": "active",
				},
			}

			// Execute
			err := repo.Create(ctx, auditEntry)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, auditEntry.ID)
			assert.NotZero(t, auditEntry.CreatedAt)
			assert.NotZero(t, auditEntry.UpdatedAt)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			// Create multiple audit entries
			entries := []struct {
				authorityType domain.AuthorityType
				authorityID   string
				entryType     domain.EventType
			}{
				{domain.AuthorityTypeAgent, "agent-1", domain.EventTypeStatusChange},
				{domain.AuthorityTypeAdmin, "service-1", domain.EventTypeConfigUpdate},
				{domain.AuthorityTypeAgent, "agent-2", domain.EventTypeStatusChange},
			}

			for _, e := range entries {
				entry := &domain.AuditEntry{
					AuthorityType: e.authorityType,
					AuthorityID:   e.authorityID,
					EventType:     e.entryType,
					Properties: domain.JSON{
						"test": "data",
					},
				}
				err := repo.Create(ctx, entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
		})

		t.Run("success - list with authority type filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"authorityType": {string(domain.AuthorityTypeAgent)}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for _, item := range result.Items {
				assert.Equal(t, domain.AuthorityTypeAgent, item.AuthorityType)
			}
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"eventType": {string(domain.EventTypeStatusChange)}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for _, item := range result.Items {
				assert.Equal(t, domain.EventTypeStatusChange, item.EventType)
			}
		})

		t.Run("success - list with sorting by created_at", func(t *testing.T) {
			// Create entries with different timestamps
			for i := 0; i < 3; i++ {
				entry := &domain.AuditEntry{
					AuthorityType: domain.AuthorityTypeAgent,
					AuthorityID:   "agent-sort",
					EventType:     domain.EventTypeUpdated,
					Properties:    domain.JSON{"test": "data"},
				}
				err := repo.Create(ctx, entry)
				require.NoError(t, err)
				time.Sleep(time.Millisecond * 100) // Ensure different timestamps
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "createdAt",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.True(t, result.Items[i-1].CreatedAt.After(result.Items[i].CreatedAt))
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			// Create multiple audit entries
			for i := 0; i < 5; i++ {
				entry := &domain.AuditEntry{
					AuthorityType: domain.AuthorityTypeAgent,
					AuthorityID:   "agent-page",
					EventType:     domain.EventTypeCreated,
					Properties:    domain.JSON{"test": "data"},
				}
				err := repo.Create(ctx, entry)
				require.NoError(t, err)
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, &domain.EmptyAuthScope, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})
}
