package database

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
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

	t.Run("create", func(t *testing.T) {
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
			found, err := repo.Get(ctx, serviceType.ID)
			require.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)
		})

		t.Run("success - with property schema", func(t *testing.T) {
			ctx := context.Background()

			// Setup - ServiceType with property schema
			serviceType := createTestServiceType(t)
			propertySchema := &schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"instanceName": {
						Type:     "string",
						Required: true,
					},
					"ipAddress": {
						Type: "string",
					},
					"diskSize": {
						Type: "integer",
					},
				},
			}
			serviceType.PropertySchema = propertySchema

			// Execute
			err := repo.Create(ctx, serviceType)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, serviceType.ID)

			// Verify property schema persisted correctly
			found, err := repo.Get(ctx, serviceType.ID)
			require.NoError(t, err)
			require.NotNil(t, found.PropertySchema)

			// Verify properties exist and basic types are preserved
			assert.Contains(t, found.PropertySchema.Properties, "instanceName")
			assert.Contains(t, found.PropertySchema.Properties, "ipAddress")
			assert.Contains(t, found.PropertySchema.Properties, "diskSize")
			assert.Equal(t, "string", found.PropertySchema.Properties["instanceName"].Type)
			assert.Equal(t, "integer", found.PropertySchema.Properties["diskSize"].Type)
		})

		t.Run("success - with nested property schema", func(t *testing.T) {
			ctx := context.Background()

			// Setup - ServiceType with nested property schema
			serviceType := createTestServiceType(t)
			propertySchema := &schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"config": {
						Type: "object",
						Properties: map[string]schema.PropertyDefinition{
							"name": {
								Type: "string",
							},
							"port": {
								Type: "integer",
							},
						},
					},
				},
			}
			serviceType.PropertySchema = propertySchema

			// Execute
			err := repo.Create(ctx, serviceType)

			// Assert
			require.NoError(t, err)

			// Verify nested properties persisted correctly
			found, err := repo.Get(ctx, serviceType.ID)
			require.NoError(t, err)

			config := found.PropertySchema.Properties["config"]
			assert.Equal(t, "object", config.Type)
			assert.Contains(t, config.Properties, "name")
			assert.Contains(t, config.Properties, "port")
			assert.Equal(t, "string", config.Properties["name"].Type)
			assert.Equal(t, "integer", config.Properties["port"].Type)
		})

		t.Run("success - with lifecycle schema", func(t *testing.T) {
			ctx := context.Background()

			// Setup - ServiceType with lifecycle schema
			serviceType := createTestServiceType(t)
			lifecycle := &domain.LifecycleSchema{
				States: []domain.LifecycleState{
					{Name: "New"},
					{Name: "Starting"},
					{Name: "Started"},
					{Name: "Stopping"},
					{Name: "Stopped"},
					{Name: "Failed"},
					{Name: "Deleted"},
				},
				Actions: []domain.LifecycleAction{
					{
						Name: "create",
						Transitions: []domain.LifecycleTransition{
							{From: "New", To: "Stopped"},
						},
					},
					{
						Name: "start",
						Transitions: []domain.LifecycleTransition{
							{From: "Stopped", To: "Starting"},
							{From: "Starting", To: "Started"},
							{From: "Starting", To: "Failed", OnError: true, OnErrorRegexp: "quota.*exceeded"},
							{From: "Starting", To: "Stopped", OnError: true},
						},
					},
					{
						Name: "stop",
						Transitions: []domain.LifecycleTransition{
							{From: "Started", To: "Stopping"},
							{From: "Stopping", To: "Stopped"},
						},
					},
					{
						Name: "delete",
						Transitions: []domain.LifecycleTransition{
							{From: "Stopped", To: "Deleted"},
							{From: "Failed", To: "Deleted"},
						},
					},
				},
				InitialState:   "New",
				TerminalStates: []string{"Deleted"},
				RunningStates:  []string{"Started"},
			}
			serviceType.LifecycleSchema = lifecycle

			// Execute
			err := repo.Create(ctx, serviceType)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, serviceType.ID)

			// Verify lifecycle schema persisted correctly
			found, err := repo.Get(ctx, serviceType.ID)
			require.NoError(t, err)
			require.NotNil(t, found.LifecycleSchema)

			// Verify basic lifecycle properties
			assert.Equal(t, "New", found.LifecycleSchema.InitialState)
			assert.Equal(t, []string{"Deleted"}, found.LifecycleSchema.TerminalStates)
			assert.Equal(t, []string{"Started"}, found.LifecycleSchema.RunningStates)
			assert.Len(t, found.LifecycleSchema.States, 7)
			assert.Len(t, found.LifecycleSchema.Actions, 4)

			// Verify states
			stateNames := make([]string, len(found.LifecycleSchema.States))
			for i, state := range found.LifecycleSchema.States {
				stateNames[i] = state.Name
			}
			assert.Contains(t, stateNames, "New")
			assert.Contains(t, stateNames, "Started")
			assert.Contains(t, stateNames, "Stopped")
			assert.Contains(t, stateNames, "Failed")

			// Verify start action with error transitions
			var startAction *domain.LifecycleAction
			for i, action := range found.LifecycleSchema.Actions {
				if action.Name == "start" {
					startAction = &found.LifecycleSchema.Actions[i]
					break
				}
			}
			require.NotNil(t, startAction)
			assert.Len(t, startAction.Transitions, 4)

			// Verify error transitions
			errorTransitions := []domain.LifecycleTransition{}
			for _, transition := range startAction.Transitions {
				if transition.OnError {
					errorTransitions = append(errorTransitions, transition)
				}
			}
			assert.Len(t, errorTransitions, 2)

			// Verify error transition with regexp
			var quotaTransition *domain.LifecycleTransition
			for i, transition := range errorTransitions {
				if transition.OnErrorRegexp != "" {
					quotaTransition = &errorTransitions[i]
					break
				}
			}
			require.NotNil(t, quotaTransition)
			assert.Equal(t, "quota.*exceeded", quotaTransition.OnErrorRegexp)
			assert.Equal(t, "Starting", quotaTransition.From)
			assert.Equal(t, "Failed", quotaTransition.To)
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			// Execute
			found, err := repo.Get(ctx, serviceType.ID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, serviceType.Name, found.Name)
		})

		t.Run("not found", func(t *testing.T) {
			ctx := context.Background()

			// Execute
			found, err := repo.Get(ctx, uuid.New())

			// Assert
			assert.Nil(t, found)
			assert.ErrorAs(t, err, &domain.NotFoundError{})
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

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, len(result.Items), 0)
			assert.Greater(t, result.TotalItems, int64(2))
		})

		t.Run("success - list with name filter", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Filters:  map[string][]string{"name": {serviceType.Name}},
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.Equal(t, serviceType.Name, result.Items[0].Name)
		})

		t.Run("success - list with sorting", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType1 := createTestServiceType(t)
			serviceType1.Name = "A Service Type"
			require.NoError(t, repo.Create(ctx, serviceType1))

			serviceType2 := createTestServiceType(t)
			serviceType2.Name = "B Service Type"
			require.NoError(t, repo.Create(ctx, serviceType2))

			page := &domain.PageReq{
				Page:     1,
				PageSize: 10,
				Sort:     true,
				SortBy:   "name",
				SortAsc:  false, // Descending order
			}

			// Execute
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Items), 2)
			// Verify descending order
			for i := 1; i < len(result.Items); i++ {
				assert.GreaterOrEqual(t, result.Items[i-1].Name, result.Items[i].Name)
			}
		})

		t.Run("success - list with pagination", func(t *testing.T) {
			ctx := context.Background()

			// Setup - Create multiple service types
			for i := 0; i < 5; i++ {
				serviceType := createTestServiceType(t)
				require.NoError(t, repo.Create(ctx, serviceType))
			}

			page := &domain.PageReq{
				Page:     1,
				PageSize: 2,
			}

			// Execute first page
			result, err := repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert first page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.False(t, result.HasPrev)
			assert.Greater(t, result.TotalItems, int64(2))

			// Execute second page
			page.Page = 2
			result, err = repo.List(ctx, &auth.IdentityScope{}, page)

			// Assert second page
			require.NoError(t, err)
			assert.Len(t, result.Items, 2)
			assert.True(t, result.HasNext)
			assert.True(t, result.HasPrev)

			// Verify total count matches
			count, err := repo.Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, result.TotalItems, count)
		})
	})

	t.Run("Count", func(t *testing.T) {
		t.Run("success - count all", func(t *testing.T) {
			ctx := context.Background()

			// Setup
			serviceType1 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType1))
			serviceType2 := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType2))

			// Execute
			count, err := repo.Count(ctx)

			// Assert
			require.NoError(t, err)
			assert.Greater(t, count, int64(1))
		})
	})

	t.Run("AuthScope", func(t *testing.T) {
		t.Run("success - returns empty auth scope", func(t *testing.T) {
			ctx := context.Background()

			// Create a service type
			serviceType := createTestServiceType(t)
			require.NoError(t, repo.Create(ctx, serviceType))

			// Execute with existing service type ID
			scope, err := repo.AuthScope(ctx, serviceType.ID)
			require.NoError(t, err)
			assert.Equal(t, &auth.AllwaysMatchObjectScope{}, scope, "Should return empty auth scope for service types")
		})
	})

}
