package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

func TestAgentTypeRepository_Integration(t *testing.T) {
	// Setup
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewAgentTypeRepository(testDB.DB)
	ctx := context.Background()

	// Helper function to create a service type
	createServiceType := func(t *testing.T, name string) *domain.ServiceType {
		serviceType, err := domain.NewServiceType(
			name,
			domain.JSON{
				"cpu":    float64(4),
				"memory": "8GB",
			},
		)
		assert.NoError(t, err)
		err = NewServiceTypeRepository(testDB.DB).Create(ctx, serviceType)
		assert.NoError(t, err)
		return serviceType
	}

	t.Run("CRUD operations", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create
			agentType, err := domain.NewAgentType("VM Runner CRUD")
			assert.NoError(t, err)

			err = repo.Create(ctx, agentType)
			assert.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, agentType.ID)

			// Read
			found, err := repo.FindByID(ctx, agentType.ID)
			assert.NoError(t, err)
			assert.Equal(t, agentType.Name, found.Name)

			// Update
			agentType.Name = "VM Runner CRUD Updated"
			err = repo.Update(ctx, agentType)
			assert.NoError(t, err)

			found, err = repo.FindByID(ctx, agentType.ID)
			assert.NoError(t, err)
			assert.Equal(t, "VM Runner CRUD Updated", found.Name)

			// Delete
			err = repo.Delete(ctx, agentType.ID)
			assert.NoError(t, err)

			_, err = repo.FindByID(ctx, agentType.ID)
			assert.Equal(t, domain.ErrNotFound, err)

			return nil
		})
	})

	t.Run("List agent types", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create multiple agent types
			agentType1, err := domain.NewAgentType("VM Runner List 1")
			assert.NoError(t, err)
			err = repo.Create(ctx, agentType1)
			assert.NoError(t, err)

			agentType2, err := domain.NewAgentType("VM Runner List 2")
			assert.NoError(t, err)
			err = repo.Create(ctx, agentType2)
			assert.NoError(t, err)

			// List all
			agentTypes, err := repo.List(ctx, nil)
			assert.NoError(t, err)
			assert.Len(t, agentTypes, 2)

			// List with filter
			agentTypes, err = repo.List(ctx, map[string]interface{}{"name": "VM Runner List 1"})
			assert.NoError(t, err)
			assert.Len(t, agentTypes, 1)
			assert.Equal(t, "VM Runner List 1", agentTypes[0].Name)

			return nil
		})
	})

	t.Run("Service type associations", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			agentType, err := domain.NewAgentType("VM Runner Service")
			assert.NoError(t, err)
			err = repo.Create(ctx, agentType)
			assert.NoError(t, err)

			serviceType1 := createServiceType(t, "VM Service 1")
			serviceType2 := createServiceType(t, "VM Service 2")

			// Add service types
			err = repo.AddServiceType(ctx, agentType.ID, serviceType1.ID)
			assert.NoError(t, err)
			err = repo.AddServiceType(ctx, agentType.ID, serviceType2.ID)
			assert.NoError(t, err)

			// Find by service type
			agentTypes, err := repo.FindByServiceType(ctx, serviceType1.ID)
			assert.NoError(t, err)
			assert.Len(t, agentTypes, 1)
			assert.Equal(t, agentType.ID, agentTypes[0].ID)

			// Remove service type
			err = repo.RemoveServiceType(ctx, agentType.ID, serviceType1.ID)
			assert.NoError(t, err)

			// Verify removal
			agentTypes, err = repo.FindByServiceType(ctx, serviceType1.ID)
			assert.NoError(t, err)
			assert.Len(t, agentTypes, 0)

			// Service type 2 should still be associated
			agentTypes, err = repo.FindByServiceType(ctx, serviceType2.ID)
			assert.NoError(t, err)
			assert.Len(t, agentTypes, 1)

			return nil
		})
	})

	t.Run("Not found cases", func(t *testing.T) {
		nonExistentID := uuid.New()

		// FindByID
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// Update
		agentType := &domain.AgentType{
			BaseEntity: domain.BaseEntity{ID: nonExistentID},
			Name:       "Non-existent",
		}
		err = repo.Update(ctx, agentType)
		assert.Equal(t, domain.ErrNotFound, err)

		// Delete
		err = repo.Delete(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// Service type associations
		serviceType := createServiceType(t, "VM Service Not Found")
		err = repo.AddServiceType(ctx, nonExistentID, serviceType.ID)
		assert.Equal(t, domain.ErrNotFound, err)
		err = repo.RemoveServiceType(ctx, nonExistentID, serviceType.ID)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}
