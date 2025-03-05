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
				AuthorityType: "agent",
				AuthorityID:   "test-agent-id",
				Type:          "status_change",
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

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 3)
		})

		t.Run("success - list with authority type filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"authorityType": {"agent"}},
			}

			// Execute
			result, err := repo.List(ctx, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for _, item := range result.Items {
				assert.Equal(t, "agent", item.AuthorityType)
			}
		})

		t.Run("success - list with type filter", func(t *testing.T) {
			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"type": {"status_change"}},
			}

			// Execute
			result, err := repo.List(ctx, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			for _, item := range result.Items {
				assert.Equal(t, "status_change", item.Type)
			}
		})

		t.Run("success - list with sorting by created_at", func(t *testing.T) {
			// Create entries with different timestamps
			for i := 0; i < 3; i++ {
				entry := &domain.AuditEntry{
					AuthorityType: "agent",
					AuthorityID:   "agent-sort",
					Type:          "test_sort",
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
			result, err := repo.List(ctx, page)

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
					AuthorityType: "agent",
					AuthorityID:   "agent-page",
					Type:          "test_pagination",
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
			result, err := repo.List(ctx, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})
	})
}
