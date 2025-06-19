package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceTypeHandler struct {
	querier domain.ServiceTypeQuerier
	authz   auth.Authorizer
}

func NewServiceTypeHandler(
	querier domain.ServiceTypeQuerier,
	authz auth.Authorizer,
) *ServiceTypeHandler {
	return &ServiceTypeHandler{
		querier: querier,
		authz:   authz,
	}
}

// Routes returns the router with all service type routes registered
func (h *ServiceTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServiceType, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using service type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Validate endpoint - authorize using service type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Post("/{id}/validate", h.handleValidate)
		})
	}
}

func (h *ServiceTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	serviceType, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceTypeToResponse(serviceType))
}

func (h *ServiceTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, serviceTypeToResponse))
}

// ServiceTypeResponse represents the response body for service type operations
type ServiceTypeResponse struct {
	ID             properties.UUID      `json:"id"`
	Name           string               `json:"name"`
	PropertySchema *schema.CustomSchema `json:"propertySchema,omitempty"`
	CreatedAt      JSONUTCTime          `json:"createdAt"`
	UpdatedAt      JSONUTCTime          `json:"updatedAt"`
}

// serviceTypeToResponse converts a domain.ServiceType to a ServiceTypeResponse
func serviceTypeToResponse(st *domain.ServiceType) *ServiceTypeResponse {
	return &ServiceTypeResponse{
		ID:             st.ID,
		Name:           st.Name,
		PropertySchema: st.PropertySchema,
		CreatedAt:      JSONUTCTime(st.CreatedAt),
		UpdatedAt:      JSONUTCTime(st.UpdatedAt),
	}
}

// ValidateRequest represents the request body for property validation
type ValidateRequest struct {
	Properties map[string]any `json:"properties"`
}

// ValidateResponse represents the response body for property validation
type ValidateResponse struct {
	Valid  bool                     `json:"valid"`
	Errors []schema.ValidationError `json:"errors,omitempty"`
}

func (h *ServiceTypeHandler) handleValidate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	// Get the service type
	serviceType, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	// Check if service type has a property schema
	if serviceType.PropertySchema == nil {
		render.Render(w, r, ErrInvalidRequest(errors.New("service type has no property schema defined")))
		return
	}

	// Parse request body
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Validate properties against schema
	validationErrors := schema.Validate(req.Properties, *serviceType.PropertySchema)

	response := ValidateResponse{
		Valid:  len(validationErrors) == 0,
		Errors: validationErrors,
	}

	render.JSON(w, r, response)
}
