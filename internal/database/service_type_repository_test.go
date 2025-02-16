package database

import (
	"context"
	"fmt"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceTypeRepository(t *testing.T) {
	// Setup test database
	tdb := NewTestDB(t)
	t.Logf("Temp test DB name %s", tdb.DBName)
	defer tdb.Cleanup(t)

	// Create repository instance
	repo := NewServiceTypeRepository(tdb.DB)

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)

			// Execute
			err := repo.Create(ctx, serviceType)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, serviceType.ID)

			// Verify in database
			found, err := repo.FindByID(ctx, serviceType.ID)
			require.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			// Execute
			found, err := repo.FindByID(ctx, serviceType.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.FindByID(ctx, domain.UUID(uuid.New()))

			// Assert
			assert.Nil(t, found)
			assert.ErrorIs(t, err, domain.ErrNotFound)
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType1 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType1))
			serviceType2 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType2))

			// Execute
			serviceTypes, err := repo.List(ctx, nil)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(serviceTypes), 0)
		})

		t.Run("success - list with filters", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			filters := map[string]interface{}{
				"name": serviceType.Name,
			}

			// Execute
			serviceTypes, err := repo.List(ctx, filters)

			// Assert
			require.NoError(t, err)
			require.Len(t, serviceTypes, 1)
			assert.Equal(t, serviceType.Name, serviceTypes[0].Name)
		})
	})
}

func createTestServiceType(t *testing.T) *domain.ServiceType {
	t.Helper()
	randomSuffix := uuid.New().String()

	return &domain.ServiceType{
		Name:       fmt.Sprintf("Test Service Type %s", randomSuffix),
		AgentTypes: []domain.AgentType{}, // Empty slice for now, can be populated if needed
	}
}
