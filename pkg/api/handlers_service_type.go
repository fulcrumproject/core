package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
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
		).Get("/", List(h.querier, ServiceTypeToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using service type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier, ServiceTypeToRes))

			// Validate endpoint - authorize using service type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Post("/{id}/validate", h.Validate)
		})
	}
}

// ServiceTypeRes represents the response body for service type operations
type ServiceTypeRes struct {
	ID             properties.UUID      `json:"id"`
	Name           string               `json:"name"`
	PropertySchema *schema.CustomSchema `json:"propertySchema,omitempty"`
	CreatedAt      JSONUTCTime          `json:"createdAt"`
	UpdatedAt      JSONUTCTime          `json:"updatedAt"`
}

// ServiceTypeToRes converts a domain.ServiceType to a ServiceTypeResponse
func ServiceTypeToRes(st *domain.ServiceType) *ServiceTypeRes {
	return &ServiceTypeRes{
		ID:             st.ID,
		Name:           st.Name,
		PropertySchema: st.PropertySchema,
		CreatedAt:      JSONUTCTime(st.CreatedAt),
		UpdatedAt:      JSONUTCTime(st.UpdatedAt),
	}
}

// ValidateReq represents the request body for property validation
type ValidateReq struct {
	Properties map[string]any `json:"properties"`
}

// ValidateRes represents the response body for property validation
type ValidateRes struct {
	Valid  bool                     `json:"valid"`
	Errors []schema.ValidationError `json:"errors,omitempty"`
}

func (h *ServiceTypeHandler) Validate(w http.ResponseWriter, r *http.Request) {
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
	var req ValidateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Validate properties against schema
	validationErrors := schema.Validate(req.Properties, *serviceType.PropertySchema)

	response := ValidateRes{
		Valid:  len(validationErrors) == 0,
		Errors: validationErrors,
	}

	render.JSON(w, r, response)
}
