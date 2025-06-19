package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEvent_Validate(t *testing.T) {
	entry := Event{
		InitiatorType: InitiatorTypeUser,
		InitiatorID:   uuid.New().String(),
		Type:          EventTypeAgentCreated,
		Payload:       properties.JSON{"key": "value"},
	}

	// Since Validate() always returns nil, this just confirms it doesn't panic
	err := entry.Validate()
	assert.NoError(t, err)
}

func TestNewEvent(t *testing.T) {
	// Create a test agent to use with WithAgent
	agent := &Agent{
		BaseEntity: BaseEntity{ID: uuid.New()},
		Name:       "test-agent",
		ProviderID: uuid.New(),
	}

	// Setup context with mock auth identity
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := auth.Identity{ID: identityID, Role: auth.RoleAdmin}
	ctx := auth.WithIdentity(baseCtx, &identity)

	entry, err := NewEvent(
		EventTypeAgentCreated,
		WithInitiatorCtx(ctx),
		WithAgent(agent),
	)

	assert.NoError(t, err)
	assert.Equal(t, InitiatorTypeUser, entry.InitiatorType)
	assert.Equal(t, identityID.String(), entry.InitiatorID)
	assert.Equal(t, EventTypeAgentCreated, entry.Type)
	assert.Equal(t, agent.ID, *entry.EntityID)
	assert.Equal(t, agent.ProviderID, *entry.ProviderID)
	assert.Equal(t, agent.ID, *entry.AgentID)
}

func TestNewEventWithDiff(t *testing.T) {
	type testEntity struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	beforeEntity := testEntity{
		Name:  "before",
		Value: 10,
	}

	afterEntity := testEntity{
		Name:  "after",
		Value: 20,
	}

	// Create a test agent to use with WithAgent
	agent := &Agent{
		BaseEntity: BaseEntity{ID: uuid.New()},
		Name:       "test-agent",
		ProviderID: uuid.New(),
	}

	// Setup context with mock auth identity
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := auth.Identity{ID: identityID, Role: auth.RoleParticipant}
	ctx := auth.WithIdentity(baseCtx, &identity)

	entry, err := NewEvent(
		EventTypeAgentUpdated,
		WithInitiatorCtx(ctx),
		WithDiff(beforeEntity, afterEntity),
		WithAgent(agent),
	)

	assert.NoError(t, err)
	assert.Equal(t, InitiatorTypeUser, entry.InitiatorType)
	assert.Equal(t, identityID.String(), entry.InitiatorID)
	assert.Equal(t, EventTypeAgentUpdated, entry.Type)
	assert.Equal(t, agent.ID, *entry.EntityID)
	assert.Equal(t, agent.ProviderID, *entry.ProviderID)
	assert.Equal(t, agent.ID, *entry.AgentID)

	// Check that diff was generated
	assert.Contains(t, entry.Payload, "diff")
	assert.NotNil(t, entry.Payload["diff"])

	// The diff should be a properties.JSON patch with operations
	diff, ok := entry.Payload["diff"]
	assert.True(t, ok)
	assert.NotNil(t, diff)
}

func TestNewEventWithDiff_ErrorHandling(t *testing.T) {
	type testEntity struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	beforeEntity := testEntity{
		Name:  "before",
		Value: 10,
	}

	afterEntity := testEntity{
		Name:  "after",
		Value: 20,
	}

	// Unmarshalable entity that will cause json.Marshal to fail
	type badEntity struct {
		BadField func() `json:"bad_field"`
	}

	badEntityInstance := badEntity{
		BadField: func() {},
	}

	// Setup context with mock auth identity
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := auth.Identity{ID: identityID, Role: auth.RoleAdmin}
	ctx := auth.WithIdentity(baseCtx, &identity)

	// Create a test metric type to use with WithMetricType
	metricType := &MetricType{
		BaseEntity: BaseEntity{ID: uuid.New()},
		Name:       "test-metric",
	}

	tests := []struct {
		name       string
		before     any
		after      any
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			name:    "Success",
			before:  beforeEntity,
			after:   afterEntity,
			wantErr: false,
		},
		{
			name:    "Marshal before entity error",
			before:  badEntityInstance,
			after:   afterEntity,
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to marshal 'before' entity")
			},
		},
		{
			name:    "Marshal after entity error",
			before:  beforeEntity,
			after:   badEntityInstance,
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to marshal 'after' entity")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewEvent(
				EventTypeAgentUpdated,
				WithInitiatorCtx(ctx),
				WithDiff(tt.before, tt.after),
				WithMetricType(metricType),
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, entry)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry)
				assert.NotNil(t, entry.Payload)
				assert.Contains(t, entry.Payload, "diff")
			}
		})
	}
}

func TestEvent_TableName(t *testing.T) {
	eventEntry := Event{}
	assert.Equal(t, "events", eventEntry.TableName())
}
