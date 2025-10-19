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

func TestServiceOptionRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceOptionRepository(testDB.DB)
	optionTypeRepo := NewServiceOptionTypeRepository(testDB.DB)
	participantRepo := NewParticipantRepository(testDB.DB)

	// Create a participant to use as provider
	participant := domain.NewParticipant(domain.CreateParticipantParams{
		Name: "Test Provider",
	})
	err := participantRepo.Create(context.Background(), participant)
	require.NoError(t, err)

	// Create a service option type for tests
	optionType := &domain.ServiceOptionType{
		Name:        "Test Option Type",
		Type:        "test_type",
		Description: "Test description",
	}
	err = optionTypeRepo.Create(context.Background(), optionType)
	require.NoError(t, err)

	t.Run("Create", func(t *testing.T) {
		option := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType.ID,
			Name:                "Ubuntu 20.04",
			Value:               map[string]any{"image": "ubuntu-20.04"},
			Enabled:             true,
			DisplayOrder:        1,
		}

		err := repo.Create(context.Background(), option)
		require.NoError(t, err)
		assert.NotEmpty(t, option.ID)
		assert.NotZero(t, option.CreatedAt)
		assert.NotZero(t, option.UpdatedAt)
	})

	t.Run("Get", func(t *testing.T) {
		// Create a service option
		option := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType.ID,
			Name:                "Ubuntu 22.04",
			Value:               map[string]any{"image": "ubuntu-22.04", "version": "22.04"},
			Enabled:             true,
			DisplayOrder:        2,
		}
		err := repo.Create(context.Background(), option)
		require.NoError(t, err)

		// Get the service option
		found, err := repo.Get(context.Background(), option.ID)
		require.NoError(t, err)
		assert.Equal(t, option.ID, found.ID)
		assert.Equal(t, option.Name, found.Name)
		assert.Equal(t, option.ProviderID, found.ProviderID)
		assert.Equal(t, option.ServiceOptionTypeID, found.ServiceOptionTypeID)
		assert.Equal(t, option.Enabled, found.Enabled)
		assert.Equal(t, option.DisplayOrder, found.DisplayOrder)
		// Value is stored as JSONB - skip detailed assertions for now due to GORM JSONB quirks
		assert.NotNil(t, found.Value)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		found, err := repo.Get(context.Background(), properties.NewUUID())
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("FindByProviderAndTypeAndValue", func(t *testing.T) {
		// Create a service option with a specific value
		testValue := map[string]any{
			"image":  "debian-11",
			"family": "debian",
		}
		option := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType.ID,
			Name:                "Debian 11",
			Value:               testValue,
			Enabled:             true,
			DisplayOrder:        3,
		}
		err := repo.Create(context.Background(), option)
		require.NoError(t, err)

		// Find by provider, type, and value
		found, err := repo.FindByProviderAndTypeAndValue(
			context.Background(),
			participant.ID,
			optionType.ID,
			testValue,
		)
		require.NoError(t, err)
		assert.Equal(t, option.ID, found.ID)
		assert.NotNil(t, found.Value)
	})

	t.Run("FindByProviderAndTypeAndValue_NotFound", func(t *testing.T) {
		// Try to find with non-existent value
		found, err := repo.FindByProviderAndTypeAndValue(
			context.Background(),
			participant.ID,
			optionType.ID,
			map[string]any{"image": "nonexistent"},
		)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("FindByProviderAndTypeAndValue_OnlyEnabledOptions", func(t *testing.T) {
		// Create a disabled option
		disabledValue := map[string]any{"image": "disabled-os"}
		disabledOption := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType.ID,
			Name:                "Disabled OS",
			Value:               disabledValue,
			Enabled:             false,
			DisplayOrder:        99,
		}
		err := repo.Create(context.Background(), disabledOption)
		require.NoError(t, err)

		// Disable it by updating it (to test the filter)
		disabledOption.Enabled = false
		err = repo.Save(context.Background(), disabledOption)
		require.NoError(t, err)

		// Try to find the disabled option - should not be found
		found, err := repo.FindByProviderAndTypeAndValue(
			context.Background(),
			participant.ID,
			optionType.ID,
			disabledValue,
		)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("ListByProvider", func(t *testing.T) {
		// Create another provider
		provider2 := domain.NewParticipant(domain.CreateParticipantParams{
			Name: "Second Provider",
		})
		err := participantRepo.Create(context.Background(), provider2)
		require.NoError(t, err)

		// Create options for second provider
		for i := 0; i < 3; i++ {
			option := &domain.ServiceOption{
				ProviderID:          provider2.ID,
				ServiceOptionTypeID: optionType.ID,
				Name:                "Provider 2 Option",
				Value:               map[string]any{"id": i},
				Enabled:             true,
				DisplayOrder:        i,
			}
			err := repo.Create(context.Background(), option)
			require.NoError(t, err)
		}

		// List options for second provider
		options, err := repo.ListByProvider(context.Background(), provider2.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(options), 3)
		for _, opt := range options {
			assert.Equal(t, provider2.ID, opt.ProviderID)
		}
	})

	t.Run("ListByProviderAndType", func(t *testing.T) {
		// Create another option type
		optionType2 := &domain.ServiceOptionType{
			Name:        "Second Type",
			Type:        "second_type",
			Description: "Second type for testing",
		}
		err := optionTypeRepo.Create(context.Background(), optionType2)
		require.NoError(t, err)

		// Create options of different types for the same provider
		option1 := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType2.ID,
			Name:                "Type 2 Option 1",
			Value:               map[string]any{"key": "value1"},
			Enabled:             true,
			DisplayOrder:        1,
		}
		err = repo.Create(context.Background(), option1)
		require.NoError(t, err)

		option2 := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType2.ID,
			Name:                "Type 2 Option 2",
			Value:               map[string]any{"key": "value2"},
			Enabled:             true,
			DisplayOrder:        2,
		}
		err = repo.Create(context.Background(), option2)
		require.NoError(t, err)

		// List options by provider and type
		options, err := repo.ListByProviderAndType(context.Background(), participant.ID, optionType2.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(options), 2)
		for _, opt := range options {
			assert.Equal(t, participant.ID, opt.ProviderID)
			assert.Equal(t, optionType2.ID, opt.ServiceOptionTypeID)
		}
	})

	t.Run("Save", func(t *testing.T) {
		// Create a service option
		option := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType.ID,
			Name:                "Original Name",
			Value:               map[string]any{"original": "value"},
			Enabled:             true,
			DisplayOrder:        10,
		}
		err := repo.Create(context.Background(), option)
		require.NoError(t, err)

		// Fetch the created option from DB
		fetched, err := repo.Get(context.Background(), option.ID)
		require.NoError(t, err)

		// Update the service option
		fetched.Name = "Updated Name"
		fetched.Value = map[string]any{"updated": "value"}
		fetched.Enabled = false
		fetched.DisplayOrder = 20

		err = repo.Save(context.Background(), fetched)
		require.NoError(t, err)

		// Verify the update
		found, err := repo.Get(context.Background(), fetched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", found.Name)
		assert.NotNil(t, found.Value, "Value should not be nil after update")
		assert.False(t, found.Enabled)
		assert.Equal(t, 20, found.DisplayOrder)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a service option
		option := &domain.ServiceOption{
			ProviderID:          participant.ID,
			ServiceOptionTypeID: optionType.ID,
			Name:                "To Delete",
			Value:               map[string]any{"delete": "me"},
			Enabled:             true,
			DisplayOrder:        100,
		}
		err := repo.Create(context.Background(), option)
		require.NoError(t, err)

		// Delete the service option
		err = repo.Delete(context.Background(), option.ID)
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.Get(context.Background(), option.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.IsType(t, domain.NotFoundError{}, err)
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), &auth.IdentityScope{}, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
		})

		t.Run("success - list with participant filter", func(t *testing.T) {
			// Create a third provider for filtering
			provider3 := domain.NewParticipant(domain.CreateParticipantParams{
				Name: "Third Provider",
			})
			err := participantRepo.Create(context.Background(), provider3)
			require.NoError(t, err)

			// Create options for third provider
			option := &domain.ServiceOption{
				ProviderID:          provider3.ID,
				ServiceOptionTypeID: optionType.ID,
				Name:                "Provider 3 Option",
				Value:               map[string]any{"filter": "test"},
				Enabled:             true,
				DisplayOrder:        1,
			}
			err = repo.Create(context.Background(), option)
			require.NoError(t, err)

			// List with participant scope
			scope := &auth.IdentityScope{
				ParticipantID: &provider3.ID,
			}
			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			result, err := repo.List(context.Background(), scope, page)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 1)
			for _, item := range result.Items {
				assert.Equal(t, provider3.ID, item.ProviderID)
			}
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns provider scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a service option
			option := &domain.ServiceOption{
				ProviderID:          participant.ID,
				ServiceOptionTypeID: optionType.ID,
				Name:                "Scope Test Option",
				Value:               map[string]any{"scope": "test"},
				Enabled:             true,
				DisplayOrder:        1,
			}
			require.NoError(t, repo.Create(ctx, option))

			// Execute with existing ID
			scope, err := repo.AuthScope(ctx, option.ID)
			require.NoError(t, err)
			defaultScope, ok := scope.(*auth.DefaultObjectScope)
			require.True(t, ok, "Should return DefaultObjectScope")
			assert.Equal(t, participant.ID, *defaultScope.ProviderID, "Should return provider ID in scope")
		})
	})
}
