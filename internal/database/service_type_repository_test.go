package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

func TestServiceTypeRepository_Integration(t *testing.T) {
	// Setup
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewServiceTypeRepository(testDB.DB)
	ctx := context.Background()

	// Helper function to create an agent type
	createAgentType := func(t *testing.T) *domain.AgentType {
		agentType, err := domain.NewAgentType("VM Runner")
		assert.NoError(t, err)
		err = NewAgentTypeRepository(testDB.DB).Create(ctx, agentType)
		assert.NoError(t, err)
		return agentType
	}

	t.Run("CRUD operations", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create
			serviceType, err := domain.NewServiceType(
				"VM Service CRUD",
				domain.JSON{
					"cpu":    float64(4),
					"memory": "8GB",
					"disk": map[string]interface{}{
						"size": "100GB",
						"type": "SSD",
					},
				},
			)
			assert.NoError(t, err)

			err = repo.Create(ctx, serviceType)
			assert.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, serviceType.ID)

			// Read
			found, err := repo.FindByID(ctx, serviceType.ID)
			assert.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)

			defs, err := found.GetResourceDefinitions()
			assert.NoError(t, err)
			assert.Equal(t, float64(4), defs["cpu"])
			assert.Equal(t, "8GB", defs["memory"])
			diskDefs := defs["disk"].(map[string]interface{})
			assert.Equal(t, "100GB", diskDefs["size"])
			assert.Equal(t, "SSD", diskDefs["type"])

			// Update
			serviceType.Name = "VM Service CRUD Updated"
			err = repo.Update(ctx, serviceType)
			assert.NoError(t, err)

			found, err = repo.FindByID(ctx, serviceType.ID)
			assert.NoError(t, err)
			assert.Equal(t, "VM Service CRUD Updated", found.Name)

			// Delete
			err = repo.Delete(ctx, serviceType.ID)
			assert.NoError(t, err)

			_, err = repo.FindByID(ctx, serviceType.ID)
			assert.Equal(t, domain.ErrNotFound, err)

			return nil
		})
	})

	t.Run("List service types", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create multiple service types
			serviceType1, err := domain.NewServiceType("VM Service List 1", domain.JSON{})
			assert.NoError(t, err)
			err = repo.Create(ctx, serviceType1)
			assert.NoError(t, err)

			serviceType2, err := domain.NewServiceType("VM Service List 2", domain.JSON{})
			assert.NoError(t, err)
			err = repo.Create(ctx, serviceType2)
			assert.NoError(t, err)

			// List all
			serviceTypes, err := repo.List(ctx, nil)
			assert.NoError(t, err)
			assert.Len(t, serviceTypes, 2)

			// List with filter
			serviceTypes, err = repo.List(ctx, map[string]interface{}{"name": "VM Service List 1"})
			assert.NoError(t, err)
			assert.Len(t, serviceTypes, 1)
			assert.Equal(t, "VM Service List 1", serviceTypes[0].Name)

			return nil
		})
	})

	t.Run("Find by agent type", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			agentType := createAgentType(t)

			// Create service types
			serviceType1, err := domain.NewServiceType("VM Service Agent 1", domain.JSON{})
			assert.NoError(t, err)
			err = repo.Create(ctx, serviceType1)
			assert.NoError(t, err)

			serviceType2, err := domain.NewServiceType("VM Service Agent 2", domain.JSON{})
			assert.NoError(t, err)
			err = repo.Create(ctx, serviceType2)
			assert.NoError(t, err)

			// Associate service type with agent type
			agentTypeRepo := NewAgentTypeRepository(testDB.DB)
			err = agentTypeRepo.AddServiceType(ctx, agentType.ID, serviceType1.ID)
			assert.NoError(t, err)

			// Find by agent type
			serviceTypes, err := repo.FindByAgentType(ctx, agentType.ID)
			assert.NoError(t, err)
			assert.Len(t, serviceTypes, 1)
			assert.Equal(t, serviceType1.ID, serviceTypes[0].ID)

			return nil
		})
	})

	t.Run("Update resource definitions", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			// Create service type
			serviceType, err := domain.NewServiceType(
				"VM Service Resource",
				domain.JSON{
					"cpu":    float64(4),
					"memory": "8GB",
				},
			)
			assert.NoError(t, err)
			err = repo.Create(ctx, serviceType)
			assert.NoError(t, err)

			// Update resource definitions
			newDefs := domain.JSON{
				"cpu":    float64(8),
				"memory": "16GB",
				"disk":   "500GB",
			}
			err = repo.UpdateResourceDefinitions(ctx, serviceType.ID, newDefs)
			assert.NoError(t, err)

			// Verify update
			found, err := repo.FindByID(ctx, serviceType.ID)
			assert.NoError(t, err)

			defs, err := found.GetResourceDefinitions()
			assert.NoError(t, err)
			assert.Equal(t, float64(8), defs["cpu"])
			assert.Equal(t, "16GB", defs["memory"])
			assert.Equal(t, "500GB", defs["disk"])

			return nil
		})
	})

	t.Run("Not found cases", func(t *testing.T) {
		nonExistentID := uuid.New()

		// FindByID
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// Update
		serviceType := &domain.ServiceType{
			BaseEntity: domain.BaseEntity{ID: nonExistentID},
			Name:       "Non-existent",
		}
		err = repo.Update(ctx, serviceType)
		assert.Equal(t, domain.ErrNotFound, err)

		// Delete
		err = repo.Delete(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// Update resource definitions
		err = repo.UpdateResourceDefinitions(ctx, nonExistentID, domain.JSON{})
		assert.Equal(t, domain.ErrNotFound, err)
	})
}
