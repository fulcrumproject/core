package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

func TestAgentRepository_Integration(t *testing.T) {
	// Setup
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewAgentRepository(testDB.DB)
	ctx := context.Background()

	// Helper function to create a provider
	createProvider := func(t *testing.T, name string) *domain.Provider {
		provider, err := domain.NewProvider(name, "US", domain.Attributes{})
		assert.NoError(t, err)
		err = NewProviderRepository(testDB.DB).Create(ctx, provider)
		assert.NoError(t, err)
		return provider
	}

	// Helper function to create an agent type
	createAgentType := func(t *testing.T, name string) *domain.AgentType {
		agentType, err := domain.NewAgentType(name)
		assert.NoError(t, err)
		err = NewAgentTypeRepository(testDB.DB).Create(ctx, agentType)
		assert.NoError(t, err)
		return agentType
	}

	t.Run("CRUD operations", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			provider := createProvider(t, "AWS-1")
			agentType := createAgentType(t, "VM Runner-1")

			// Create
			agent, err := domain.NewAgent(
				"Test Agent",
				"US",
				domain.Attributes{
					"zone":     {"zone1", "zone2"},
					"capacity": {"high"},
				},
				domain.JSON{
					"maxConnections": float64(100),
					"timeout":        float64(30),
				},
				provider.ID,
				agentType.ID,
			)
			assert.NoError(t, err)

			err = repo.Create(ctx, agent)
			assert.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, agent.ID)

			// Read
			found, err := repo.FindByID(ctx, agent.ID)
			assert.NoError(t, err)
			assert.Equal(t, agent.Name, found.Name)
			assert.Equal(t, agent.CountryCode, found.CountryCode)
			assert.Equal(t, domain.AgentNew, found.State)

			attrs, err := found.GetAttributes()
			assert.NoError(t, err)
			assert.Equal(t, []string{"zone1", "zone2"}, attrs["zone"])
			assert.Equal(t, []string{"high"}, attrs["capacity"])

			props, err := found.GetProperties()
			assert.NoError(t, err)
			assert.Equal(t, float64(100), props["maxConnections"])
			assert.Equal(t, float64(30), props["timeout"])

			// Update
			agent.Name = "Test Agent Updated"
			err = repo.Update(ctx, agent)
			assert.NoError(t, err)

			found, err = repo.FindByID(ctx, agent.ID)
			assert.NoError(t, err)
			assert.Equal(t, "Test Agent Updated", found.Name)

			// Delete
			err = repo.Delete(ctx, agent.ID)
			assert.NoError(t, err)

			_, err = repo.FindByID(ctx, agent.ID)
			assert.Equal(t, domain.ErrNotFound, err)

			return nil
		})
	})

	t.Run("List agents", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			provider := createProvider(t, "AWS-2")
			agentType := createAgentType(t, "VM Runner-2")

			// Create multiple agents
			agent1, err := domain.NewAgent("Agent 1", "US", domain.Attributes{}, domain.JSON{}, provider.ID, agentType.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent1)
			assert.NoError(t, err)

			agent2, err := domain.NewAgent("Agent 2", "US", domain.Attributes{}, domain.JSON{}, provider.ID, agentType.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent2)
			assert.NoError(t, err)

			// List all
			agents, err := repo.List(ctx, nil)
			assert.NoError(t, err)
			assert.Len(t, agents, 2)

			// List with filter
			agents, err = repo.List(ctx, map[string]interface{}{"name": "Agent 1"})
			assert.NoError(t, err)
			assert.Len(t, agents, 1)
			assert.Equal(t, "Agent 1", agents[0].Name)

			return nil
		})
	})

	t.Run("Find by provider", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			provider1 := createProvider(t, "AWS-3")
			provider2 := createProvider(t, "AWS-4")
			agentType := createAgentType(t, "VM Runner-3")

			// Create agents for different providers
			agent1, err := domain.NewAgent("Agent 1", "US", domain.Attributes{}, domain.JSON{}, provider1.ID, agentType.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent1)
			assert.NoError(t, err)

			agent2, err := domain.NewAgent("Agent 2", "US", domain.Attributes{}, domain.JSON{}, provider2.ID, agentType.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent2)
			assert.NoError(t, err)

			// Find by provider
			agents, err := repo.FindByProvider(ctx, provider1.ID)
			assert.NoError(t, err)
			assert.Len(t, agents, 1)
			assert.Equal(t, provider1.ID, agents[0].ProviderID)

			return nil
		})
	})

	t.Run("Find by agent type", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			provider := createProvider(t, "AWS-5")
			agentType1 := createAgentType(t, "VM Runner-4")
			agentType2 := createAgentType(t, "VM Runner-5")

			// Create agents for different agent types
			agent1, err := domain.NewAgent("Agent 1", "US", domain.Attributes{}, domain.JSON{}, provider.ID, agentType1.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent1)
			assert.NoError(t, err)

			agent2, err := domain.NewAgent("Agent 2", "US", domain.Attributes{}, domain.JSON{}, provider.ID, agentType2.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent2)
			assert.NoError(t, err)

			// Find by agent type
			agents, err := repo.FindByAgentType(ctx, agentType1.ID)
			assert.NoError(t, err)
			assert.Len(t, agents, 1)
			assert.Equal(t, agentType1.ID, agents[0].AgentTypeID)

			return nil
		})
	})

	t.Run("Update state", func(t *testing.T) {
		testDB.RunWithinTransaction(t, func(ctx context.Context, tx *gorm.DB) error {
			provider := createProvider(t, "AWS-6")
			agentType := createAgentType(t, "VM Runner-6")

			// Create agent
			agent, err := domain.NewAgent("Test Agent", "US", domain.Attributes{}, domain.JSON{}, provider.ID, agentType.ID)
			assert.NoError(t, err)
			err = repo.Create(ctx, agent)
			assert.NoError(t, err)

			// Update state
			err = repo.UpdateState(ctx, agent.ID, domain.AgentConnected)
			assert.NoError(t, err)

			// Verify state change
			found, err := repo.FindByID(ctx, agent.ID)
			assert.NoError(t, err)
			assert.Equal(t, domain.AgentConnected, found.State)

			return nil
		})
	})

	t.Run("Not found cases", func(t *testing.T) {
		nonExistentID := uuid.New()

		// FindByID
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// Update
		provider := createProvider(t, "AWS-7")
		agentType := createAgentType(t, "VM Runner-7")
		agent := &domain.Agent{
			BaseEntity:  domain.BaseEntity{ID: nonExistentID},
			Name:        "Non-existent",
			CountryCode: "US",
			State:       domain.AgentDisabled,
			TokenHash:   "dummy-token-hash",
			ProviderID:  provider.ID,
			AgentTypeID: agentType.ID,
		}
		err = repo.Update(ctx, agent)
		assert.Equal(t, domain.ErrNotFound, err)

		// Delete
		err = repo.Delete(ctx, nonExistentID)
		assert.Equal(t, domain.ErrNotFound, err)

		// UpdateState
		err = repo.UpdateState(ctx, nonExistentID, domain.AgentConnected)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}
