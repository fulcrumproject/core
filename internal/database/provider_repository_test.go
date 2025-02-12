package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

func TestProviderRepository_Integration(t *testing.T) {
	// Setup
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewProviderRepository(testDB.DB)
	ctx := context.Background()

	// Ensure clean state before each test
	testDB.TruncateTables(t)

	t.Run("CRUD operations", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create
			provider, err := domain.NewProvider(
				"AWS CRUD",
				"US",
				domain.Attributes{
					"region":     {"us-east-1", "us-west-2"},
					"tier":       {"enterprise"},
					"available":  {"true"},
					"max_agents": {"100"},
				},
			)
			assert.NoError(t, err)

			err = repo.Create(ctx, provider)
			assert.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, provider.ID)

			// Read
			found, err := repo.FindByID(ctx, provider.ID)
			assert.NoError(t, err)
			assert.Equal(t, provider.Name, found.Name)
			assert.Equal(t, provider.CountryCode, found.CountryCode)
			assert.Equal(t, domain.ProviderEnabled, found.State) // Default state is enabled

			attrs, err := found.GetAttributes()
			assert.NoError(t, err)
			assert.Equal(t, []string{"us-east-1", "us-west-2"}, attrs["region"])
			assert.Equal(t, []string{"enterprise"}, attrs["tier"])
			assert.Equal(t, []string{"true"}, attrs["available"])
			assert.Equal(t, []string{"100"}, attrs["max_agents"])

			// Update
			provider.Name = "AWS CRUD Updated"
			err = repo.Update(ctx, provider)
			assert.NoError(t, err)

			found, err = repo.FindByID(ctx, provider.ID)
			assert.NoError(t, err)
			assert.Equal(t, "AWS CRUD Updated", found.Name)

			// Delete
			err = repo.Delete(ctx, provider.ID)
			assert.NoError(t, err)

			_, err = repo.FindByID(ctx, provider.ID)
			assert.Equal(t, domain.ErrNotFound, err)

			return nil
		})
	})

	t.Run("List providers", func(t *testing.T) {
		// Ensure clean state before test
		testDB.TruncateTables(t)

		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create multiple providers
			provider1, err := domain.NewProvider("AWS List 1", "US", domain.Attributes{})
			assert.NoError(t, err)
			err = repo.Create(ctx, provider1)
			assert.NoError(t, err)

			provider2, err := domain.NewProvider("AWS List 2", "US", domain.Attributes{})
			assert.NoError(t, err)
			err = repo.Create(ctx, provider2)
			assert.NoError(t, err)

			// List all
			providers, err := repo.List(ctx, nil)
			assert.NoError(t, err)
			assert.Len(t, providers, 2)

			// List with filter
			providers, err = repo.List(ctx, map[string]interface{}{"name": "AWS List 1"})
			assert.NoError(t, err)
			assert.Len(t, providers, 1)
			assert.Equal(t, "AWS List 1", providers[0].Name)

			return nil
		})
	})

	t.Run("Find by country code", func(t *testing.T) {
		// Ensure clean state before test
		testDB.TruncateTables(t)

		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create providers in different countries
			provider1, err := domain.NewProvider("AWS US", "US", domain.Attributes{})
			assert.NoError(t, err)
			err = repo.Create(ctx, provider1)
			assert.NoError(t, err)

			provider2, err := domain.NewProvider("AWS UK", "GB", domain.Attributes{})
			assert.NoError(t, err)
			err = repo.Create(ctx, provider2)
			assert.NoError(t, err)

			// Find by country code
			providers, err := repo.FindByCountryCode(ctx, "US")
			assert.NoError(t, err)
			assert.Len(t, providers, 1)
			assert.Equal(t, "US", providers[0].CountryCode)

			return nil
		})
	})

	t.Run("Update state", func(t *testing.T) {
		// Ensure clean state before test
		testDB.TruncateTables(t)

		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create provider
			provider, err := domain.NewProvider("AWS State", "US", domain.Attributes{})
			assert.NoError(t, err)
			err = repo.Create(ctx, provider)
			assert.NoError(t, err)

			// Update state
			err = repo.UpdateState(ctx, provider.ID, domain.ProviderDisabled)
			assert.NoError(t, err)

			// Verify state change
			found, err := repo.FindByID(ctx, provider.ID)
			assert.NoError(t, err)
			assert.Equal(t, domain.ProviderDisabled, found.State)

			return nil
		})
	})

	t.Run("Not found cases", func(t *testing.T) {
		// Ensure clean state before test
		testDB.TruncateTables(t)

		nonExistentID := uuid.New()

		// FindByID
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// Update
		provider := &domain.Provider{
			BaseEntity:  domain.BaseEntity{ID: nonExistentID},
			Name:        "Non-existent",
			CountryCode: "US",
		}
		err = repo.Update(ctx, provider)
		assert.Equal(t, domain.ErrNotFound, err)

		// Delete
		err = repo.Delete(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// UpdateState
		err = repo.UpdateState(ctx, nonExistentID, domain.ProviderDisabled)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}
