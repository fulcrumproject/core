package domain

import (
	"context"
	"slices"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
)

// KeycloakUser represents a user managed in Keycloak.
type KeycloakUser struct {
	ID            string
	Username      string
	FirstName     string
	LastName      string
	Email         string
	EmailVerified bool
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

// KeycloakUserListParams defines the filtering and pagination parameters for listing keycloak users.
type KeycloakUserListParams struct {
	Email     string
	FirstName string
	LastName  string
	Page      int // converted to "first" = (Page-1) * PageSize
	PageSize  int // maps to "max"
}

// KeycloakRole represents a Keycloak realm role.
type KeycloakRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// KeycloakUserQuerier defines the read operations for keycloak users.
type KeycloakUserQuerier interface {
	Get(ctx context.Context, id string) (*KeycloakUser, error)
	List(ctx context.Context, params KeycloakUserListParams) (*PageRes[KeycloakUserListItem], error)
}

// KeycloakAdminClient defines the interface for Keycloak admin operations.
// Implemented by keycloak.AdminClient.
type KeycloakAdminClient interface {
	KeycloakUserQuerier
	Create(ctx context.Context, params CreateKeycloakUserParams) (*KeycloakUser, error)
	Update(ctx context.Context, id string, params UpdateKeycloakUserParams) (*KeycloakUser, error)
	Delete(ctx context.Context, id string) error
}

// CreateKeycloakUserParams defines the parameters for creating a keycloak user.
type CreateKeycloakUserParams struct {
	Username      string
	Email         string
	EmailVerified bool
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

// UpdateKeycloakUserParams defines the parameters for updating a keycloak user.
type UpdateKeycloakUserParams struct {
	Email         *string
	FirstName     *string
	LastName      *string
	Enabled       *bool
	Password      *string
	Role          *auth.Role
	ParticipantID *string
	AgentID       *string
}

// KeycloakUserCommander defines the write operations for keycloak users.
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

// NewKeycloakUserCommander creates a new KeycloakUserCommander.
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
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if err := c.validateCreateRoleAttributes(ctx, params); err != nil {
		return nil, err
	}
	return c.adminClient.Create(ctx, params)
}

func (c *keycloakUserCommander) Update(ctx context.Context, id string, params UpdateKeycloakUserParams) (*KeycloakUser, error) {
	if id == "" {
		return nil, NewInvalidInputErrorf("keycloak user id is required")
	}
	if params.Role != nil {
		if err := c.validateRoleChange(ctx, &params); err != nil {
			return nil, err
		}
	} else if params.ParticipantID != nil || params.AgentID != nil {
		if err := c.validateAttributeOnlyUpdate(ctx, id, &params); err != nil {
			return nil, err
		}
	}
	return c.adminClient.Update(ctx, id, params)
}

func (c *keycloakUserCommander) Delete(ctx context.Context, id string) error {
	if id == "" {
		return NewInvalidInputErrorf("keycloak user id is required")
	}
	return c.adminClient.Delete(ctx, id)
}

func (c *keycloakUserCommander) validateCreateRoleAttributes(ctx context.Context, params CreateKeycloakUserParams) error {
	switch params.Role {
	case auth.RoleParticipant:
		if params.ParticipantID == "" {
			return NewInvalidInputErrorf("participantId is required for role participant")
		}
		return c.validateEntityExists(ctx, params.ParticipantID, "participant", c.participantQuerier.Exists)
	case auth.RoleAgent:
		if params.AgentID == "" {
			return NewInvalidInputErrorf("agentId is required for role agent")
		}
		return c.validateEntityExists(ctx, params.AgentID, "agent", c.agentQuerier.Exists)
	}
	return nil
}

func (c *keycloakUserCommander) validateRoleChange(ctx context.Context, params *UpdateKeycloakUserParams) error {
	if err := params.Role.Validate(); err != nil {
		return NewInvalidInputErrorf("invalid role: %s", *params.Role)
	}
	switch *params.Role {
	case auth.RoleParticipant:
		if params.ParticipantID == nil || *params.ParticipantID == "" {
			return NewInvalidInputErrorf("participantId is required for role participant")
		}
		if err := c.validateEntityExists(ctx, *params.ParticipantID, "participant", c.participantQuerier.Exists); err != nil {
			return err
		}
		params.AgentID = helpers.StringPtr("")
	case auth.RoleAgent:
		if params.AgentID == nil || *params.AgentID == "" {
			return NewInvalidInputErrorf("agentId is required for role agent")
		}
		if err := c.validateEntityExists(ctx, *params.AgentID, "agent", c.agentQuerier.Exists); err != nil {
			return err
		}
		params.ParticipantID = helpers.StringPtr("")
	case auth.RoleAdmin:
		params.ParticipantID = helpers.StringPtr("")
		params.AgentID = helpers.StringPtr("")
	}
	return nil
}

func (c *keycloakUserCommander) validateAttributeOnlyUpdate(ctx context.Context, id string, params *UpdateKeycloakUserParams) error {
	currentUser, err := c.adminClient.Get(ctx, id)
	if err != nil {
		return err
	}
	if params.ParticipantID != nil && *params.ParticipantID != "" {
		if !slices.Contains(currentUser.Roles, string(auth.RoleParticipant)) {
			return NewInvalidInputErrorf("participantId can only be set on users with role participant")
		}
		if err := c.validateEntityExists(ctx, *params.ParticipantID, "participant", c.participantQuerier.Exists); err != nil {
			return err
		}
	}
	if params.AgentID != nil && *params.AgentID != "" {
		if !slices.Contains(currentUser.Roles, string(auth.RoleAgent)) {
			return NewInvalidInputErrorf("agentId can only be set on users with role agent")
		}
		if err := c.validateEntityExists(ctx, *params.AgentID, "agent", c.agentQuerier.Exists); err != nil {
			return err
		}
	}
	return nil
}

// validateEntityExists checks that an entity with the given ID exists using the provided existsFn.
func (c *keycloakUserCommander) validateEntityExists(ctx context.Context, rawID string, entityName string, existsFn func(context.Context, properties.UUID) (bool, error)) error {
	id, err := properties.ParseUUID(rawID)
	if err != nil {
		return NewInvalidInputErrorf("invalid %s id: %s", entityName, rawID)
	}
	exists, err := existsFn(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return NewInvalidInputErrorf("%s with id %s not found", entityName, rawID)
	}
	return nil
}
