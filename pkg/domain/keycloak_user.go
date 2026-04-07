package domain

import (
	"context"
	"log/slog"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

type KeycloakUser struct {
	ID            string
	Username      string
	FirstName     string
	LastName      string
	Email         string
	Enabled       bool
	Roles         []string
	ParticipantID string
	AgentID       string
}

// KeycloakUserListItem is a slim representation for list responses.
// The Keycloak GET /users endpoint doesn't populate realmRoles,
// and fetching roles per-user is too expensive for a list.
type KeycloakUserListItem struct {
	ID        string
	Username  string
	Email     string
	FirstName string
	LastName  string
}

type KeycloakUserListParams struct {
	Search   string //maps to keycloak search contained in username, first or last name, or email.
	Page     int    // converted to "first" = (Page-1) * PageSize
	PageSize int    // maps to "max"
}

// KeycloakRole represents a Keycloak realm role.
type KeycloakRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type KeycloakUserQuerier interface {
	Get(ctx context.Context, id string) (*KeycloakUser, error)
	List(ctx context.Context, params KeycloakUserListParams) (*PageRes[KeycloakUserListItem], error)
}

// KeycloakAdminClient defines the interface for Keycloak admin operations.
// Implemented by keycloak.AdminClient.
type KeycloakAdminClient interface {
	KeycloakUserQuerier
	CreateUser(ctx context.Context, user *KeycloakUser) (string, error)
	UpdateUser(ctx context.Context, id string, params UpdateKeycloakUserParams) (*KeycloakUser, error)
	DeleteUser(ctx context.Context, id string) error
	SetPassword(ctx context.Context, id string, password string, temporary bool) error
	GetRealmRoles(ctx context.Context) ([]KeycloakRole, error)
	AssignRealmRoles(ctx context.Context, id string, roles []KeycloakRole) error
	RemoveRealmRoles(ctx context.Context, id string, roles []KeycloakRole) error
}

type CreateKeycloakUserParams struct {
	Username      string
	Email         string
	FirstName     string
	LastName      string
	Password      string
	Enabled       bool
	Role          auth.Role
	ParticipantID string // required if role is "participant"
	AgentID       string // required if role is "agent"
}

func (p *CreateKeycloakUserParams) Validate() error {
	if p.Username == "" {
		return NewInvalidInputErrorf("username is required")
	}
	if p.Email == "" {
		return NewInvalidInputErrorf("email is required")
	}
	if p.FirstName == "" {
		return NewInvalidInputErrorf("first name is required")
	}
	if p.LastName == "" {
		return NewInvalidInputErrorf("last name is required")
	}
	if p.Password == "" {
		return NewInvalidInputErrorf("password is required")
	}
	if err := p.Role.Validate(); err != nil {
		return NewInvalidInputErrorf("invalid role: %s", p.Role)
	}
	return nil
}

type UpdateKeycloakUserParams struct {
	Email     *string
	FirstName *string
	LastName  *string
	Enabled   *bool
	Password  *string
}

type KeycloakUserCommander interface {
	Create(ctx context.Context, params CreateKeycloakUserParams) (*KeycloakUser, error)
	Update(ctx context.Context, id string, params UpdateKeycloakUserParams) (*KeycloakUser, error)
	Delete(ctx context.Context, id string) error
}

type keycloakUserCommander struct {
	adminClient        KeycloakAdminClient
	participantQuerier ParticipantQuerier
	agentQuerier       AgentQuerier
}

func NewKeycloakUserCommander(
	adminClient KeycloakAdminClient,
	participantQuerier ParticipantQuerier,
	agentQuerier AgentQuerier,
) KeycloakUserCommander {
	return &keycloakUserCommander{
		adminClient:        adminClient,
		participantQuerier: participantQuerier,
		agentQuerier:       agentQuerier,
	}
}

func (c *keycloakUserCommander) Create(ctx context.Context, params CreateKeycloakUserParams) (*KeycloakUser, error) {
	// Validate required fields
	if err := params.Validate(); err != nil {
		return nil, err
	}

	if params.Role == auth.RoleParticipant {
		if params.ParticipantID == "" {
			return nil, NewInvalidInputErrorf("participantId is required for role participant")
		}
		if err := c.validateParticipantID(ctx, params.ParticipantID); err != nil {
			return nil, err
		}
	}

	if params.Role == auth.RoleAgent {
		if params.AgentID == "" {
			return nil, NewInvalidInputErrorf("agentId is required for role agent")
		}
		if err := c.validateAgentID(ctx, params.AgentID); err != nil {
			return nil, err
		}
	}

	// Create user in Keycloak
	userID, err := c.adminClient.CreateUser(ctx, &KeycloakUser{
		Username:      params.Username,
		Email:         params.Email,
		FirstName:     params.FirstName,
		LastName:      params.LastName,
		Enabled:       params.Enabled,
		ParticipantID: params.ParticipantID,
		AgentID:       params.AgentID,
	})
	if err != nil {
		return nil, err
	}

	// Set password
	if err := c.adminClient.SetPassword(ctx, userID, params.Password, false); err != nil {
		c.compensatingDelete(ctx, userID)
		return nil, err
	}

	// Assign realm role
	realmRoles, err := c.adminClient.GetRealmRoles(ctx)
	if err != nil {
		c.compensatingDelete(ctx, userID)
		return nil, err
	}
	var targetRole *KeycloakRole
	for _, r := range realmRoles {
		if r.Name == string(params.Role) {
			targetRole = &r
			break
		}
	}
	if targetRole == nil {
		c.compensatingDelete(ctx, userID)
		return nil, NewInvalidInputErrorf("realm role %s not found in Keycloak", params.Role)
	}
	if err := c.adminClient.AssignRealmRoles(ctx, userID, []KeycloakRole{*targetRole}); err != nil {
		c.compensatingDelete(ctx, userID)
		return nil, err
	}

	return c.adminClient.Get(ctx, userID)
}

func (c *keycloakUserCommander) Update(ctx context.Context, id string, params UpdateKeycloakUserParams) (*KeycloakUser, error) {
	if id == "" {
		return nil, NewInvalidInputErrorf("keycloak user id is required")
	}

	user, err := c.adminClient.UpdateUser(ctx, id, UpdateKeycloakUserParams{
		Email:     params.Email,
		FirstName: params.FirstName,
		LastName:  params.LastName,
		Enabled:   params.Enabled,
	})
	if err != nil {
		return nil, err
	}

	if params.Password != nil {
		if err := c.adminClient.SetPassword(ctx, id, *params.Password, false); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (c *keycloakUserCommander) Delete(ctx context.Context, id string) error {
	if id == "" {
		return NewInvalidInputErrorf("keycloak user id is required")
	}
	return c.adminClient.DeleteUser(ctx, id)
}

// compensatingDelete attempts to clean up a partially created user.
func (c *keycloakUserCommander) compensatingDelete(ctx context.Context, userID string) {
	if err := c.adminClient.DeleteUser(ctx, userID); err != nil {
		slog.Error("failed compensating delete of keycloak user", "userID", userID, "error", err)
	}
}

// validateParticipantID checks that the participant exists in the local DB.
func (c *keycloakUserCommander) validateParticipantID(ctx context.Context, participantID string) error {
	id, err := properties.ParseUUID(participantID)
	if err != nil {
		return NewInvalidInputErrorf("invalid participant id: %s", participantID)
	}
	exists, err := c.participantQuerier.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return NewInvalidInputErrorf("participant with id %s not found", participantID)
	}
	return nil
}

// validateAgentID checks that the agent exists in the local DB.
func (c *keycloakUserCommander) validateAgentID(ctx context.Context, agentID string) error {
	id, err := properties.ParseUUID(agentID)
	if err != nil {
		return NewInvalidInputErrorf("invalid agent id: %s", agentID)
	}
	exists, err := c.agentQuerier.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return NewInvalidInputErrorf("agent with id %s not found", agentID)
	}
	return nil
}
