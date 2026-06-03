package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfrastructureRepository(t *testing.T) {
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	infraRepo := NewInfrastructureRepository(tdb.DB)
	infraTypeRepo := NewInfrastructureTypeRepository(tdb.DB)
	participantRepo := NewParticipantRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success with preloads", func(t *testing.T) {
			ctx := context.Background()

			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			infraType := createTestInfrastructureType(t)
			require.NoError(t, infraTypeRepo.Create(ctx, infraType))

			infra := createTestInfrastructure(t, participant.ID, infraType.ID)
			require.NoError(t, infraRepo.Create(ctx, infra))
			assert.NotEmpty(t, infra.ID)

			found, err := infraRepo.Get(ctx, infra.ID)
			require.NoError(t, err)
			assert.Equal(t, infra.Name, found.Name)
			assert.Equal(t, infra.ProviderID, found.ProviderID)
			assert.Equal(t, infra.InfrastructureTypeID, found.InfrastructureTypeID)
			assert.NotNil(t, found.Provider, "Provider should be preloaded")
			assert.NotNil(t, found.InfrastructureType, "InfrastructureType should be preloaded")
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()
			found, err := infraRepo.Get(ctx, uuid.New())
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("filter by providerId", func(t *testing.T) {
			ctx := context.Background()

			provA := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, provA))
			provB := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, provB))

			infraType := createTestInfrastructureType(t)
			require.NoError(t, infraTypeRepo.Create(ctx, infraType))

			infraA := createTestInfrastructure(t, provA.ID, infraType.ID)
			require.NoError(t, infraRepo.Create(ctx, infraA))
			infraB := createTestInfrastructure(t, provB.ID, infraType.ID)
			require.NoError(t, infraRepo.Create(ctx, infraB))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"providerId": {provA.ID.String()}},
			}
			result, err := infraRepo.List(ctx, &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, infraA.ID, result.Items[0].ID)
		})

		t.Run("filter by infrastructureTypeId", func(t *testing.T) {
			ctx := context.Background()

			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))

			itA := createTestInfrastructureType(t)
			require.NoError(t, infraTypeRepo.Create(ctx, itA))
			itB := createTestInfrastructureType(t)
			require.NoError(t, infraTypeRepo.Create(ctx, itB))

			infraOfA := createTestInfrastructure(t, participant.ID, itA.ID)
			require.NoError(t, infraRepo.Create(ctx, infraOfA))
			infraOfB := createTestInfrastructure(t, participant.ID, itB.ID)
			require.NoError(t, infraRepo.Create(ctx, infraOfB))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"infrastructureTypeId": {itA.ID.String()}},
			}
			result, err := infraRepo.List(ctx, &auth.IdentityScope{}, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, infraOfA.ID, result.Items[0].ID)
		})
	})

	t.Run("Save", func(t *testing.T) {
		ctx := context.Background()

		participant := createTestParticipant(t, domain.ParticipantEnabled)
		require.NoError(t, participantRepo.Create(ctx, participant))
		infraType := createTestInfrastructureType(t)
		require.NoError(t, infraTypeRepo.Create(ctx, infraType))

		infra := createTestInfrastructure(t, participant.ID, infraType.ID)
		require.NoError(t, infraRepo.Create(ctx, infra))

		infra.Name = "Renamed infra"
		require.NoError(t, infraRepo.Save(ctx, infra))

		found, err := infraRepo.Get(ctx, infra.ID)
		require.NoError(t, err)
		assert.Equal(t, "Renamed infra", found.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		ctx := context.Background()

		participant := createTestParticipant(t, domain.ParticipantEnabled)
		require.NoError(t, participantRepo.Create(ctx, participant))
		infraType := createTestInfrastructureType(t)
		require.NoError(t, infraTypeRepo.Create(ctx, infraType))

		infra := createTestInfrastructure(t, participant.ID, infraType.ID)
		require.NoError(t, infraRepo.Create(ctx, infra))

		require.NoError(t, infraRepo.Delete(ctx, infra.ID))

		_, err := infraRepo.Get(ctx, infra.ID)
		assert.ErrorAs(t, err, &domain.NotFoundError{})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("returns provider + agent coordinates", func(t *testing.T) {
			ctx := context.Background()

			participant := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, participant))
			infraType := createTestInfrastructureType(t)
			require.NoError(t, infraTypeRepo.Create(ctx, infraType))

			infra := createTestInfrastructure(t, participant.ID, infraType.ID)
			require.NoError(t, infraRepo.Create(ctx, infra))

			scope, err := infraRepo.AuthScope(ctx, infra.ID)
			require.NoError(t, err)
			require.NotNil(t, scope)

			ds, ok := scope.(*authz.DefaultObjectScope)
			require.True(t, ok, "expected *authz.DefaultObjectScope, got %T", scope)
			require.NotNil(t, ds.ProviderID, "ProviderID should be populated from provider_id")
			assert.Equal(t, participant.ID, *ds.ProviderID)
			require.NotNil(t, ds.AgentID, "AgentID coordinate doubles as the self-reference for Infrastructure")
			assert.Equal(t, infra.ID, *ds.AgentID)
		})

		t.Run("returns not-found error for missing row", func(t *testing.T) {
			ctx := context.Background()
			_, err := infraRepo.AuthScope(ctx, properties.UUID(uuid.New()))
			require.Error(t, err)
		})
	})
}
