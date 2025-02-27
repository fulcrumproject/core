package database

import (
	"fmt"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
)

func createTestServiceType(t *testing.T) *domain.ServiceType {
	t.Helper()
	randomSuffix := uuid.New().String()

	return &domain.ServiceType{
		Name: fmt.Sprintf("Test Service Type %s", randomSuffix),
	}
}

func createTestProvider(t *testing.T, state domain.ProviderState) *domain.Provider {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Provider{
		Name:        fmt.Sprintf("Test Provider %s", randomSuffix),
		State:       state,
		CountryCode: "US",
		Attributes:  domain.Attributes{"key": []string{"value"}},
	}
}
func createTestAgentType(t *testing.T) *domain.AgentType {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.AgentType{
		Name: fmt.Sprintf("Test Agent Type %s", randomSuffix),
	}
}

func createTestAgent(t *testing.T, providerID, agentTypeID domain.UUID, state domain.AgentState) *domain.Agent {
	t.Helper()
	return createTestAgentWithStatusUpdate(t, providerID, agentTypeID, state, time.Now())
}

// Helper function to create a test agent with a specific LastStatusUpdate time
func createTestAgentWithStatusUpdate(t *testing.T, providerID, agentTypeID domain.UUID, state domain.AgentState, lastUpdate time.Time) *domain.Agent {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Agent{
		Name:             fmt.Sprintf("Test Agent %s", randomSuffix),
		State:            state,
		TokenHash:        fmt.Sprintf("token-hash-%s", randomSuffix),
		CountryCode:      "US",
		Attributes:       domain.Attributes{"key": []string{"value"}},
		Properties:       map[string]interface{}{"prop": "value"},
		ProviderID:       providerID,
		AgentTypeID:      agentTypeID,
		LastStatusUpdate: lastUpdate,
	}
}

func createTestServiceGroup(t *testing.T) *domain.ServiceGroup {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ServiceGroup{
		Name: fmt.Sprintf("Test ServiceGroup %s", randomSuffix),
	}
}

func createTestService(t *testing.T, serviceTypeID, serviceGroupID, agentID domain.UUID) *domain.Service {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Service{
		Name:          fmt.Sprintf("Test Service %s", randomSuffix),
		ServiceTypeID: serviceTypeID,
		GroupID:       serviceGroupID,
		State:         domain.ServiceNew,
		AgentID:       agentID,
		Attributes:    domain.Attributes{},
		Resources:     domain.JSON{},
	}
}

func createTestMetricType(t *testing.T) *domain.MetricType {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.MetricType{
		Name:       fmt.Sprintf("Test MetricType %s", randomSuffix),
		EntityType: domain.MetricEntityTypeService,
	}
}
