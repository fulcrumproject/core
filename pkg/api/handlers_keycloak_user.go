package api

import (
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// KeycloakUserRes is the response body for keycloak user get, create, and update operations.
type KeycloakUserRes struct {
	ID            string      `json:"id"`
	Username      string      `json:"username"`
	Email         string      `json:"email"`
	EmailVerified bool        `json:"emailVerified"`
	FirstName     string      `json:"firstName"`
	LastName      string      `json:"lastName"`
	Enabled       bool        `json:"enabled"`
	Roles         []auth.Role `json:"roles"`
	ParticipantID string      `json:"participantId,omitempty"`
	AgentID       string      `json:"agentId,omitempty"`
}

// KeycloakUserListItemRes is the response body for keycloak user list items.
type KeycloakUserListItemRes struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// CreateKeycloakUserReq is the request body for creating a keycloak user.
type CreateKeycloakUserReq struct {
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Password      string    `json:"password"`
	Enabled       bool      `json:"enabled"`
	Role          auth.Role `json:"role"`
	ParticipantID string    `json:"participantId,omitempty"`
	AgentID       string    `json:"agentId,omitempty"`
}

// UpdateKeycloakUserReq is the request body for updating a keycloak user.
type UpdateKeycloakUserReq struct {
	Email         *string    `json:"email,omitempty"`
	FirstName     *string    `json:"firstName,omitempty"`
	LastName      *string    `json:"lastName,omitempty"`
	Enabled       *bool      `json:"enabled,omitempty"`
	Password      *string    `json:"password,omitempty"`
	Role          *auth.Role `json:"role,omitempty"`
	ParticipantID *string    `json:"participantId,omitempty"`
	AgentID       *string    `json:"agentId,omitempty"`
}

// KeycloakUserHandler handles HTTP requests for keycloak user operations.
type KeycloakUserHandler struct {
	querier   domain.KeycloakUserQuerier
	commander domain.KeycloakUserCommander
	authz     authz.Authorizer
}

// NewKeycloakUserHandler creates a new KeycloakUserHandler.
func NewKeycloakUserHandler(
	querier domain.KeycloakUserQuerier,
	commander domain.KeycloakUserCommander,
	authz authz.Authorizer,
) *KeycloakUserHandler {
	return &KeycloakUserHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *KeycloakUserHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionRead, h.authz),
		).Get("/", h.List)

		r.With(
			middlewares.DecodeBody[CreateKeycloakUserReq](),
			middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionCreate, h.authz),
		).Post("/", h.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.With(
				middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionRead, h.authz),
			).Get("/", h.Get)

			r.With(
				middlewares.DecodeBody[UpdateKeycloakUserReq](),
				middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionUpdate, h.authz),
			).Patch("/", h.Update)

			r.With(
				middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionDelete, h.authz),
			).Delete("/", h.Delete)
		})
	}
}

func parsePageRequestKeycloakUser(r *http.Request) (*domain.KeycloakUserListParams, error) {
	pag, err := ParsePageRequest(r)
	if err != nil {
		return nil, err
	}

	q := r.URL.Query()

	return &domain.KeycloakUserListParams{
		Page:      pag.Page,
		PageSize:  pag.PageSize,
		Email:     q.Get("email"),
		FirstName: q.Get("firstName"),
		LastName:  q.Get("lastName"),
	}, nil
}

func (h *KeycloakUserHandler) List(w http.ResponseWriter, r *http.Request) {
	params, err := parsePageRequestKeycloakUser(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), *params)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, KeycloakUserListItemToRes))
}

func (h *KeycloakUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := middlewares.MustGetBody[CreateKeycloakUserReq](r.Context())

	user, err := h.commander.Create(r.Context(), domain.CreateKeycloakUserParams{
		Username:      req.Username,
		Email:         req.Email,
		EmailVerified: req.EmailVerified,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Password:      req.Password,
		Enabled:       req.Enabled,
		Role:          req.Role,
		ParticipantID: req.ParticipantID,
		AgentID:       req.AgentID,
	})
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, KeycloakUserToRes(user))
}

func (h *KeycloakUserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, KeycloakUserToRes(user))
}

func (h *KeycloakUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	req := middlewares.MustGetBody[UpdateKeycloakUserReq](r.Context())

	user, err := h.commander.Update(r.Context(), id, domain.UpdateKeycloakUserParams{
		Email:         req.Email,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Enabled:       req.Enabled,
		Password:      req.Password,
		Role:          req.Role,
		ParticipantID: req.ParticipantID,
		AgentID:       req.AgentID,
	})
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, KeycloakUserToRes(user))
}

func (h *KeycloakUserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// KeycloakUserToRes converts a domain.KeycloakUser to a KeycloakUserRes.
func KeycloakUserToRes(user *domain.KeycloakUser) *KeycloakUserRes {
	return &KeycloakUserRes{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Enabled:       user.Enabled,
		Roles:         user.Roles,
		ParticipantID: user.ParticipantID,
		AgentID:       user.AgentID,
	}
}

// KeycloakUserListItemToRes converts a domain.KeycloakUserListItem to a KeycloakUserListItemRes.
func KeycloakUserListItemToRes(item *domain.KeycloakUserListItem) *KeycloakUserListItemRes {
	return &KeycloakUserListItemRes{
		ID:        item.ID,
		Username:  item.Username,
		Email:     item.Email,
		FirstName: item.FirstName,
		LastName:  item.LastName,
	}
}
