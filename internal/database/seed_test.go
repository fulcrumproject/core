package database

import (
	"context"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSeed(t *testing.T) {
	// Create a test database
	testDB := NewTestDB(t)
	defer testDB.Cleanup(t)

	// Run the seed function
	err := Seed(testDB.DB)
	assert.NoError(t, err, "Seed function should not return an error")

	// Create the repositories to verify data
	agentTypeRepo := NewAgentTypeRepository(testDB.DB)
	serviceTypeRepo := NewServiceTypeRepository(testDB.DB)
	metricTypeRepo := NewMetricTypeRepository(testDB.DB)
	tokenRepo := NewTokenRepository(testDB.DB)
	ctx := context.Background()

	// Test that vm service type was created
	vmServiceTypeID := uuid.MustParse("0195c3c8-69e5-7806-9598-8523c01ea54f")
	exists, err := serviceTypeRepo.Exists(ctx, vmServiceTypeID)
	assert.NoError(t, err, "Error checking if service type exists")
	assert.True(t, exists, "VM service type should exist after seeding")

	vmServiceType, err := serviceTypeRepo.FindByID(ctx, vmServiceTypeID)
	assert.NoError(t, err, "Error getting service type")
	assert.Equal(t, "vm", vmServiceType.Name, "VM service type should have correct name")

	// Test that dummy agent type was created
	dummyAgentTypeID := uuid.MustParse("0195c3c6-4c7d-7e3c-b481-f276e17a7bec")
	exists, err = agentTypeRepo.Exists(ctx, dummyAgentTypeID)
	assert.NoError(t, err, "Error checking if agent type exists")
	assert.True(t, exists, "Dummy agent type should exist after seeding")

	dummyAgentType, err := agentTypeRepo.FindByID(ctx, dummyAgentTypeID)
	assert.NoError(t, err, "Error getting agent type")
	assert.Equal(t, "dummy", dummyAgentType.Name, "Dummy agent type should have correct name")

	// Test that dummy agent type has VM service type
	found := false
	for _, st := range dummyAgentType.ServiceTypes {
		if st.ID == vmServiceTypeID {
			found = true
			break
		}
	}
	assert.True(t, found, "Dummy agent type should have VM service type")

	// Test that metric types were created
	metricTypeIDs := []uuid.UUID{
		uuid.MustParse("0195c3c9-a211-753b-86d0-be343ec40df4"),
		uuid.MustParse("0195c3c9-fb93-717e-8e4d-94247359e35c"),
		uuid.MustParse("0195c3ca-2c6e-771c-8250-3a5dabaaceee"),
		uuid.MustParse("0195c3ca-6334-74fd-a230-a64bf1d4f376"),
	}

	metricTypeNames := []string{
		"vm.cpu.usage",
		"vm.memory.usage",
		"vm.disk.usage",
		"vm.network.throughput",
	}

	for i, id := range metricTypeIDs {
		exists, err := metricTypeRepo.Exists(ctx, id)
		assert.NoError(t, err, "Error checking if metric type exists")
		assert.True(t, exists, "Metric type should exist after seeding")

		mt, err := metricTypeRepo.FindByID(ctx, id)
		assert.NoError(t, err, "Error getting metric type")
		assert.Equal(t, metricTypeNames[i], mt.Name, "Metric type should have correct name")
		assert.Equal(t, domain.MetricEntityTypeResource, mt.EntityType, "Metric type should have correct entity type")
	}

	// Test that admin token was created
	adminTokenID := uuid.MustParse("0195c3cc-a21a-7474-a214-f6fd48d4609b")
	exists, err = tokenRepo.Exists(ctx, adminTokenID)
	assert.NoError(t, err, "Error checking if admin token exists")
	assert.True(t, exists, "Admin token should exist after seeding")

	adminToken, err := tokenRepo.FindByID(ctx, adminTokenID)
	assert.NoError(t, err, "Error getting admin token")
	assert.Equal(t, "Admin Test Token", adminToken.Name, "Admin token should have correct name")
	assert.Equal(t, domain.RoleAdmin, adminToken.Role, "Admin token should have admin role")
	assert.Equal(t, domain.HashTokenValue("admin-test-token"), adminToken.HashedValue, "Admin token should have correct hashed value")
}
