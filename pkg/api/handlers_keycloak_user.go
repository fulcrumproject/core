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

// KeycloakUserRes is the full DTO for get/create/update responses.
type KeycloakUserRes struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	FirstName     string   `json:"firstName"`
	LastName      string   `json:"lastName"`
	Enabled       bool     `json:"enabled"`
	Roles         []string `json:"roles"`
	ParticipantID string   `json:"participantId,omitempty"`
	AgentID       string   `json:"agentId,omitempty"`
}

type KeycloakUserHandler struct {
	querier   domain.KeycloakUserQuerier
	commander domain.KeycloakUserCommander
	authz     authz.Authorizer
}

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
		).Get("/", keycloakUserHandlerList(h.querier))

		r.With(
			middlewares.DecodeBody[CreateKeycloakUserReq](),
			middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionCreate, h.authz),
		).Post("/", keycloakUserHandlerCreate(h.commander))

		r.Route("/{id}", func(r chi.Router) {
			r.With(
				middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionRead, h.authz),
			).Get("/", keycloakUserHandlerGet(h.querier))

			r.With(
				middlewares.DecodeBody[UpdateKeycloakUserReq](),
				middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionUpdate, h.authz),
			).Patch("/", keycloakUserHandlerUpdate(h.commander))

			r.With(
				middlewares.AuthzSimple(authz.ObjectTypeKeycloakUser, authz.ActionDelete, h.authz),
			).Delete("/", keycloakUserHandlerDelete(h.commander))
		})
	}
}

func parsePageRequestKeycloakUser(r *http.Request) (*domain.KeycloakUserListParams, error) {
	pag, err := ParsePageRequest(r)
	if err != nil {
		return nil, err
	}
	return &domain.KeycloakUserListParams{
		Page:     pag.Page,
		PageSize: pag.PageSize,
		Search:   r.URL.Query().Get("search"),
	}, nil
}

func keycloakUserHandlerList(querier domain.KeycloakUserQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params, err := parsePageRequestKeycloakUser(r)
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		result, err := querier.List(r.Context(), *params)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, NewPageResponse(result, func(item *domain.KeycloakUserListItem) *domain.KeycloakUserListItem {
			return item
		}))
	}
}

type CreateKeycloakUserReq struct {
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Password      string    `json:"password"`
	Enabled       bool      `json:"enabled"`
	Role          auth.Role `json:"role"`
	ParticipantID string    `json:"participantId,omitempty"`
	AgentID       string    `json:"agentId,omitempty"`
}

func keycloakUserToRes(user *domain.KeycloakUser) *KeycloakUserRes {
	return &KeycloakUserRes{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Enabled:       user.Enabled,
		Roles:         user.Roles,
		ParticipantID: user.ParticipantID,
		AgentID:       user.AgentID,
	}
}

func keycloakUserHandlerCreate(commander domain.KeycloakUserCommander) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := middlewares.MustGetBody[CreateKeycloakUserReq](r.Context())

		user, err := commander.Create(r.Context(), domain.CreateKeycloakUserParams(req))
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, keycloakUserToRes(user))
	}
}

func keycloakUserHandlerGet(querier domain.KeycloakUserQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		user, err := querier.Get(r.Context(), id)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		render.JSON(w, r, keycloakUserToRes(user))
	}
}

type UpdateKeycloakUserReq struct {
	Email     *string `json:"email,omitempty"`
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Password  *string `json:"password,omitempty"`
}

func keycloakUserHandlerUpdate(commander domain.KeycloakUserCommander) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		req := middlewares.MustGetBody[UpdateKeycloakUserReq](r.Context())

		user, err := commander.Update(r.Context(), id, domain.UpdateKeycloakUserParams(req))
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, keycloakUserToRes(user))
	}
}

func keycloakUserHandlerDelete(commander domain.KeycloakUserCommander) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := commander.Delete(r.Context(), id); err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
