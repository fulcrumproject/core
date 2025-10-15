package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
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
		Name:   fmt.Sprintf("Test Participant %s", randomSuffix),
		Status: status,
	}
}
func createTestAgentType(t *testing.T) *domain.AgentType {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.AgentType{
		Name: fmt.Sprintf("Test Agent Type %s", randomSuffix),
	}
}

func createTestAgent(t *testing.T, participantID, agentTypeID properties.UUID, status domain.AgentStatus) *domain.Agent {
	t.Helper()
	return createTestAgentWithTags(t, participantID, agentTypeID, status, nil)
}

func createTestAgentWithTags(t *testing.T, participantID, agentTypeID properties.UUID, status domain.AgentStatus, tags []string) *domain.Agent {
	t.Helper()
	return createTestAgentWithStatusUpdateAndTags(t, participantID, agentTypeID, status, time.Now(), tags)
}

func createTestAgentWithStatusUpdate(t *testing.T, participantID, agentTypeID properties.UUID, status domain.AgentStatus, lastUpdate time.Time) *domain.Agent {
	t.Helper()
	return createTestAgentWithStatusUpdateAndTags(t, participantID, agentTypeID, status, lastUpdate, nil)
}

func createTestAgentWithStatusUpdateAndTags(t *testing.T, participantID, agentTypeID properties.UUID, status domain.AgentStatus, lastUpdate time.Time, tags []string) *domain.Agent {
	t.Helper()
	return createTestAgentWithConfig(t, participantID, agentTypeID, status, lastUpdate, tags, nil)
}

func createTestAgentWithConfig(t *testing.T, participantID, agentTypeID properties.UUID, status domain.AgentStatus, lastUpdate time.Time, tags []string, configuration *properties.JSON) *domain.Agent {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Agent{
		Name:             fmt.Sprintf("Test Agent %s", randomSuffix),
		Status:           status,
		ProviderID:       participantID,
		AgentTypeID:      agentTypeID,
		LastStatusUpdate: lastUpdate,
		Tags:             tags,
		Configuration:    configuration,
	}
}

func createTestServiceGroup(t *testing.T, participantID properties.UUID) *domain.ServiceGroup {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.ServiceGroup{
		Name:       fmt.Sprintf("Test ServiceGroup %s", randomSuffix),
		ConsumerID: participantID,
	}
}

func createTestService(t *testing.T, serviceTypeID, serviceGroupID, agentID, providerParticipantID, consumerParticipantID properties.UUID) *domain.Service {
	t.Helper()
	randomSuffix := uuid.New().String()
	return &domain.Service{
		Name:          fmt.Sprintf("Test Service %s", randomSuffix),
		ServiceTypeID: serviceTypeID,
		GroupID:       serviceGroupID,
		Status:        "Started",
		ProviderID:    providerParticipantID,
		ConsumerID:    consumerParticipantID,
		AgentID:       agentID,
		Properties:    &(properties.JSON{}),
		Resources:     &(properties.JSON{}),
	}
}

func createTestToken(t *testing.T, role auth.Role, scopeID *properties.UUID) *domain.Token {
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
		case auth.RoleParticipant:
			token.ParticipantID = scopeID
		case auth.RoleAgent:
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
func createTestMetricEntry(t *testing.T, agentID, serviceID, typeID, providerParticipantID, consumerParticipantID properties.UUID) *domain.MetricEntry {
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
