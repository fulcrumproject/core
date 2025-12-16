package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParticipantRepository(t *testing.T) {
	tdb := NewTestDB(t)
	defer tdb.Cleanup(t)
	// Create repository
	repo := NewParticipantRepository(tdb.DB)
	t.Run("create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			participant := createTestParticipant(t, domain.ParticipantEnabled)

			err := repo.Create(ctx, participant)

			require.NoError(t, err)
			
			found, err := repo.Get(ctx, participant.ID)

			require.NoError(t, err)
			assert.Equal(t, participant.ID, found.ID)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			participant := createTestParticipant(t, domain.ParticipantEnabled)

			err := repo.Create(ctx, participant)

			require.NoError(t, err)

			found, err := repo.Get(ctx, participant.ID)
			
			require.NoError(t, err)

			assert.Equal(t, participant.ID, found.ID)
			assert.Equal(t, participant.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			found, err := repo.Get(ctx, properties.NewUUID())

			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant1 := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, repo.Create(ctx, participant1))
			participant2 := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, repo.Create(ctx, participant2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
		})
		t.Run("success - list with participant scope filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup - Create two participants
			participant1 := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, repo.Create(ctx, participant1))
			participant2 := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, repo.Create(ctx, participant2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			// Execute with participant1 scope
			scope := &auth.IdentityScope{
				ParticipantID: &participant1.ID,
			}
			result, err := repo.List(ctx, scope, page)

			// Assert - should only return participant1
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, participant1.ID, result.Items[0].ID)
		})
	})
	t.Run("Save", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, repo.Create(ctx, participant))

			// Read
			participant, err := repo.Get(ctx, participant.ID)
			require.NoError(t, err)

			// Update participant
			participant.Name = "Updated Participant"
			participant.Status = domain.ParticipantDisabled

			// Execute
			err = repo.Save(ctx, participant)

			// Assert
			require.NoError(t, err)

			// Verify in database
			updated, err := repo.Get(ctx, participant.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated Participant", updated.Name)
			assert.Equal(t, domain.ParticipantDisabled, updated.Status)
		})
	})
	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, repo.Create(ctx, participant))

			// Execute
			err := repo.Delete(ctx, participant.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := repo.Get(ctx, participant.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})
}