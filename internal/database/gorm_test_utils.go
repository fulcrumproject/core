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

func createTestParticipant(t *testing.T, status domain.ParticipantStatus) *domain.Participant {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Participant{
		Name:        fmt.Sprintf("Test Participant %s", randomSuffix),
		Status:      status,
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

func createTestAgent(t *testing.T, participantID, agentTypeID domain.UUID, status domain.AgentStatus) *domain.Agent {
	t.Helper()
	return createTestAgentWithStatusUpdate(t, participantID, agentTypeID, status, time.Now())
}

// Helper function to create a test agent with a specific LastStatusUpdate time
func createTestAgentWithStatusUpdate(t *testing.T, participantID, agentTypeID domain.UUID, status domain.AgentStatus, lastUpdate time.Time) *domain.Agent {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Agent{
		Name:             fmt.Sprintf("Test Agent %s", randomSuffix),
		Status:           status,
		CountryCode:      "US",
		Attributes:       domain.Attributes{"key": []string{"value"}},
		ProviderID:       participantID,
		AgentTypeID:      agentTypeID,
		LastStatusUpdate: lastUpdate,
	}
}

func createTestServiceGroup(t *testing.T, participantID domain.UUID) *domain.ServiceGroup {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ServiceGroup{
		Name:       fmt.Sprintf("Test ServiceGroup %s", randomSuffix),
		ConsumerID: participantID,
	}
}

func createTestService(t *testing.T, serviceTypeID, serviceGroupID, agentID, providerParticipantID, consumerParticipantID domain.UUID) *domain.Service {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Service{
		Name:              fmt.Sprintf("Test Service %s", randomSuffix),
		ServiceTypeID:     serviceTypeID,
		GroupID:           serviceGroupID,
		CurrentStatus:     domain.ServiceStarted,
		ProviderID:        providerParticipantID,
		ConsumerID:        consumerParticipantID,
		AgentID:           agentID,
		CurrentProperties: &(domain.JSON{}),
		Resources:         &(domain.JSON{}),
	}
}

func createTestToken(t *testing.T, role domain.AuthRole, scopeID *domain.UUID) *domain.Token {
	t.Helper()
	randomSuffix := uuid.New().String()
	token := &domain.Token{
		Name:     fmt.Sprintf("Test Token %s", randomSuffix),
		Role:     role,
		ExpireAt: time.Now().Add(24 * time.Hour), // Expires in 24 hours
	}

	// Set the specific scope ID field based on role
	if scopeID != nil {
		switch role {
		case domain.RoleParticipant:
			token.ParticipantID = scopeID
		case domain.RoleAgent:
			token.AgentID = scopeID
		}
	}
	err := token.GenerateTokenValue()
	if err != nil {
		t.Fatalf("Failed to generate token value: %v", err)
	}
	return token
}

// createTestMetricTypeForEntity creates a test metric type for a specific entity type
func createTestMetricTypeForEntity(t *testing.T, entityType domain.MetricEntityType) *domain.MetricType {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.MetricType{
		Name:       fmt.Sprintf("Test MetricType %s for %s", randomSuffix, entityType),
		EntityType: entityType,
	}
}

// createTestMetricEntry creates a test metric entry with all required relationships
func createTestMetricEntry(t *testing.T, agentID, serviceID, typeID, providerParticipantID, consumerParticipantID domain.UUID) *domain.MetricEntry {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.MetricEntry{
		AgentID:    agentID,
		ServiceID:  serviceID,
		ResourceID: fmt.Sprintf("resource-%s", randomSuffix),
		ProviderID: providerParticipantID,
		ConsumerID: consumerParticipantID,
		Value:      42.0,
		TypeID:     typeID,
	}
}
