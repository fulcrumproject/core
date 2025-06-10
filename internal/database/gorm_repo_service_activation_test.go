package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fulcrumproject.org/core/internal/domain"
)

func TestServiceActivationRepository(t *testing.T) {
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)
	repo := NewServiceActivationRepository(testDB.DB)

	// Create test dependencies
	participantRepo := NewParticipantRepository(testDB.DB)
	provider := createTestParticipant(t, domain.ParticipantEnabled)
	require.NoError(t, participantRepo.Create(context.Background(), provider))

	serviceTypeRepo := NewServiceTypeRepository(testDB.DB)
	serviceType := createTestServiceType(t)
	require.NoError(t, serviceTypeRepo.Create(context.Background(), serviceType))

	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	agentType := createTestAgentType(t)
	require.NoError(t, agentTypeRepo.Create(context.Background(), agentType))

	agentRepo := NewAgentRepository(testDB.DB)
	agent1 := createTestAgent(t, provider.ID, agentType.ID, domain.AgentConnected)
	require.NoError(t, agentRepo.Create(context.Background(), agent1))
	agent2 := createTestAgent(t, provider.ID, agentType.ID, domain.AgentConnected)
	require.NoError(t, agentRepo.Create(context.Background(), agent2))

	t.Run("Create", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			tags := []string{"ISO 27001", "SOC 2 Type II"}
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, tags)

			// Execute
			err := repo.Create(ctx, serviceActivation)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, serviceActivation.ID)
			assert.NotZero(t, serviceActivation.CreatedAt)
			assert.NotZero(t, serviceActivation.UpdatedAt)

			// Verify in database
			found, err := repo.FindByID(ctx, serviceActivation.ID)
			require.NoError(t, err)
			assert.Equal(t, serviceActivation.ProviderID, found.ProviderID)
			assert.Equal(t, serviceActivation.ServiceTypeID, found.ServiceTypeID)
			assert.Equal(t, tags, []string(found.Tags))
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			tags := []string{"GDPR Compliant"}
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, tags)
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Execute
			found, err := repo.FindByID(ctx, serviceActivation.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, serviceActivation.ProviderID, found.ProviderID)
			assert.Equal(t, serviceActivation.ServiceTypeID, found.ServiceTypeID)
			assert.Equal(t, tags, []string(found.Tags))
			assert.Empty(t, found.Agents) // No agents associated yet
		})

		t.Run("success with agents", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			tags := []string{"PCI DSS"}
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, tags)
			serviceActivation.Agents = []domain.Agent{*agent1, *agent2}
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Execute
			found, err := repo.FindByID(ctx, serviceActivation.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, serviceActivation.ProviderID, found.ProviderID)
			assert.Equal(t, serviceActivation.ServiceTypeID, found.ServiceTypeID)
			assert.Equal(t, tags, []string(found.Tags))
			assert.Len(t, found.Agents, 2)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.FindByID(ctx, domain.NewUUID())

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("success - list all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			sa1 := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"ISO 27001"})
			require.NoError(t, repo.Create(ctx, sa1))
			sa2 := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"SOC 2"})
			require.NoError(t, repo.Create(ctx, sa2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
		})

		t.Run("success - list with provider_id filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"HIPAA"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"provider_id": {provider.ID.String()}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			for _, item := range result.Items {
				assert.Equal(t, provider.ID, item.ProviderID)
			}
		})

		t.Run("success - list with service_type_id filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"FedRAMP"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"service_type_id": {serviceType.ID.String()}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			for _, item := range result.Items {
				assert.Equal(t, serviceType.ID, item.ServiceTypeID)
			}
		})

		t.Run("success - list with tag filter (contains all)", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			tags := []string{"ISO 27001", "SOC 2 Type II", "GDPR Compliant"}
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, tags)
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Create another service activation with different tags
			otherTags := []string{"PCI DSS", "HIPAA"}
			otherSA := createTestServiceActivation(t, provider.ID, serviceType.ID, otherTags)
			require.NoError(t, repo.Create(ctx, otherSA))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"tag": {"ISO 27001", "SOC 2 Type II", "GDPR Compliant"}},
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, serviceActivation.ID, result.Items[0].ID)
			assert.Contains(t, []string(result.Items[0].Tags), "ISO 27001")
			assert.Contains(t, []string(result.Items[0].Tags), "SOC 2 Type II")
			assert.Contains(t, []string(result.Items[0].Tags), "GDPR Compliant")
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			sa1 := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Tag A"})
			require.NoError(t, repo.Create(ctx, sa1))

			sa2 := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Tag B"})
			require.NoError(t, repo.Create(ctx, sa2))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "created_at",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			// Verify descending order by created_at
			for i := 1; i < len(result.Items); i++ {
				assert.True(t, result.Items[i-1].CreatedAt.After(result.Items[i].CreatedAt) ||
					result.Items[i-1].CreatedAt.Equal(result.Items[i].CreatedAt))
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup - Create multiple service activations
			for i := 0; i < 5; i++ {
				sa := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Test Tag"})
				require.NoError(t, repo.Create(ctx, sa))
			}

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, &domain.EmptyAuthIdentityScope, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)
		})

		t.Run("success - list with participant authorization", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Authorized Tag"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Create another provider and service activation
			otherProvider := createTestParticipant(t, domain.ParticipantEnabled)
			require.NoError(t, participantRepo.Create(ctx, otherProvider))
			otherSA := createTestServiceActivation(t, otherProvider.ID, serviceType.ID, []string{"Other Tag"})
			require.NoError(t, repo.Create(ctx, otherSA))

			page := &domain.PageRequest{
				Page:     1,
				PageSize: 10,
			}

			authScope := &domain.AuthIdentityScope{
				ParticipantID: &provider.ID,
			}

			// Execute
			result, err := repo.List(ctx, authScope, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			for _, item := range result.Items {
				assert.Equal(t, provider.ID, item.ProviderID)
			}
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Original Tag"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Update the service activation
			serviceActivation.Tags = []string{"Updated Tag", "New Tag"}

			// Execute
			err := repo.Save(ctx, serviceActivation)

			// Assert
			require.NoError(t, err)

			// Verify in database
			found, err := repo.FindByID(ctx, serviceActivation.ID)
			require.NoError(t, err)
			assert.Equal(t, []string{"Updated Tag", "New Tag"}, []string(found.Tags))
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"To Delete"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Execute
			err := repo.Delete(ctx, serviceActivation.ID)

			// Assert
			require.NoError(t, err)

			// Verify deletion
			found, err := repo.FindByID(ctx, serviceActivation.ID)
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
		})
	})

	t.Run("Exists", func(t *testing.T) {
		t.Run("success - exists", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Exists Tag"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Execute
			exists, err := repo.Exists(ctx, serviceActivation.ID)

			// Assert
			require.NoError(t, err)
			assert.True(t, exists)
		})

		t.Run("success - does not exist", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			exists, err := repo.Exists(ctx, domain.NewUUID())

			// Assert
			require.NoError(t, err)
			assert.False(t, exists)
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns provider auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceActivation := createTestServiceActivation(t, provider.ID, serviceType.ID, []string{"Auth Tag"})
			require.NoError(t, repo.Create(ctx, serviceActivation))

			// Execute
			scope, err := repo.AuthScope(ctx, serviceActivation.ID)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, scope, "AuthScope should not return nil")
			assert.NotNil(t, scope.ProviderID, "ProviderID should not be nil")
			assert.Equal(t, provider.ID, *scope.ProviderID, "ProviderID should match the provider's ID")
			assert.Nil(t, scope.AgentID, "AgentID should be nil for service activations")
		})
	})
}
