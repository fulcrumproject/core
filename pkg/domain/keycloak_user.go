package domain

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

type KeycloakUser struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	FirstName     string   `json:"firstName"`
	LastName      string   `json:"lastName"`
	Email         string   `json:"email"`
	Enabled       bool     `json:"enabled"`
	Roles         []string `json:"roles"`
	ParticipantID string   `json:"participantId"`
	AgentID       string   `json:"agentId"`
}

// KeycloakUserListItem is a slim representation for list responses.
// The Keycloak GET /users endpoint doesn't populate realmRoles,
// and fetching roles per-user is too expensive for a list.
type KeycloakUserListItem struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (k *KeycloakUser) Validate() error {
	if k.ID == "" {
		return errors.New("keycloak user id is required")
	}

	if k.Username == "" {
		return errors.New("keycloak user username is required")
	}

	if k.Email == "" {
		return errors.New("keycloak user email is required")
	}

	if k.FirstName == "" {
		return errors.New("keycloak user first name is required")
	}

	if k.LastName == "" {
		return errors.New("keycloak user last name is required")
	}

	return nil
}

type KeycloakUserListParams struct {
	Search   string //maps to keycloak search contained in username, first or last name, or email.
	Page     int    // converted to "first" = (Page-1) * PageSize
	PageSize int    // maps to "max"
}

type KeycloakUserPaginatedRes struct {
	Items      []KeycloakUserListItem `json:"items"`
	TotalItems int                    `json:"totalItems"`
}

// KeycloakRole represents a Keycloak realm role.
type KeycloakRole struct {
	ID   string
	Name string
}

// KeycloakUserCreateRequest is the data needed to create a user in Keycloak.
type KeycloakUserCreateRequest struct {
	Username   string
	Email      string
	FirstName  string
	LastName   string
	Enabled    bool
	Attributes map[string][]string
}

// KeycloakUserUpdateRequest is the data needed to update a user in Keycloak.
type KeycloakUserUpdateRequest struct {
	Email      *string
	FirstName  *string
	LastName   *string
	Enabled    *bool
	Attributes map[string][]string
}

type KeycloakUserQuerier interface {
	Get(ctx context.Context, id string) (*KeycloakUser, error)
	List(ctx context.Context, params KeycloakUserListParams) (KeycloakUserPaginatedRes, error)
}

// KeycloakAdminClient defines the interface for Keycloak admin operations.
// Implemented by keycloak.AdminClient.
type KeycloakAdminClient interface {
	KeycloakUserQuerier
	CreateUser(ctx context.Context, user KeycloakUserCreateRequest) (string, error)
	UpdateUser(ctx context.Context, id string, user KeycloakUserUpdateRequest) (*KeycloakUser, error)
	DeleteUser(ctx context.Context, id string) error
	SetPassword(ctx context.Context, id string, password string, temporary bool) error
	GetRealmRoles(ctx context.Context) ([]KeycloakRole, error)
	AssignRealmRoles(ctx context.Context, id string, roles []KeycloakRole) error
	RemoveRealmRoles(ctx context.Context, id string, roles []KeycloakRole) error
}

type CreateKeycloakUserParams struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Password      string `json:"password"`
	Enabled       bool   `json:"enabled"`
	Role          auth.Role `json:"role"`
	ParticipantID string `json:"participantId"` // required if role is "participant"
	AgentID       string `json:"agentId"`       // required if role is "agent"
}

type UpdateKeycloakUserParams struct {
	Email     *string `json:"email,omitempty"`
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Password  *string `json:"password,omitempty"`
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
	if params.Username == "" {
		return nil, NewInvalidInputErrorf("username is required")
	}
	if params.Email == "" {
		return nil, NewInvalidInputErrorf("email is required")
	}
	if params.FirstName == "" {
		return nil, NewInvalidInputErrorf("first name is required")
	}
	if params.LastName == "" {
		return nil, NewInvalidInputErrorf("last name is required")
	}
	if params.Password == "" {
		return nil, NewInvalidInputErrorf("password is required")
	}

	if err := params.Role.Validate(); err != nil {
		return nil, NewInvalidInputErrorf("invalid role: %s", params.Role)
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

	// Build attributes (participant_id, agent_id) upfront
	attrs := map[string][]string{}
	if params.ParticipantID != "" {
		attrs["participant_id"] = []string{params.ParticipantID}
	}
	if params.AgentID != "" {
		attrs["agent_id"] = []string{params.AgentID}
	}

	// Create user in Keycloak (includes attributes)
	userID, err := c.adminClient.CreateUser(ctx, KeycloakUserCreateRequest{
		Username:   params.Username,
		Email:      params.Email,
		FirstName:  params.FirstName,
		LastName:   params.LastName,
		Enabled:    params.Enabled,
		Attributes: attrs,
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

	user, err := c.adminClient.UpdateUser(ctx, id, KeycloakUserUpdateRequest{
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
