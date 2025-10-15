package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceTypeHandler struct {
	querier   domain.ServiceTypeQuerier
	commander domain.ServiceTypeCommander
	authz     auth.Authorizer
}

func NewServiceTypeHandler(
	querier domain.ServiceTypeQuerier,
	commander domain.ServiceTypeCommander,
	authz auth.Authorizer,
) *ServiceTypeHandler {
	return &ServiceTypeHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all service type routes registered
func (h *ServiceTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServiceType, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServiceTypeToRes))

		// Create endpoint - admin only
		r.With(
			middlewares.DecodeBody[CreateServiceTypeReq](),
			middlewares.AuthzSimple(authz.ObjectTypeServiceType, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServiceTypeToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using service type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ServiceTypeToRes))

			// Update endpoint - admin only
			r.With(
				middlewares.DecodeBody[UpdateServiceTypeReq](),
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServiceTypeToRes))

			// Delete endpoint - admin only
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))

			// Validate endpoint - authorize using service type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Post("/{id}/validate", h.Validate)
		})
	}
}

// CreateServiceTypeReq represents the request body for creating service types
type CreateServiceTypeReq struct {
	Name            string                  `json:"name"`
	PropertySchema  *domain.ServiceSchema   `json:"propertySchema,omitempty"`
	LifecycleSchema *domain.LifecycleSchema `json:"lifecycleSchema,omitempty"`
}

// UpdateServiceTypeReq represents the request body for updating service types
type UpdateServiceTypeReq struct {
	Name            *string                 `json:"name"`
	PropertySchema  *domain.ServiceSchema   `json:"propertySchema,omitempty"`
	LifecycleSchema *domain.LifecycleSchema `json:"lifecycleSchema,omitempty"`
}

// ServiceTypeRes represents the response body for service type operations
type ServiceTypeRes struct {
	ID              properties.UUID         `json:"id"`
	Name            string                  `json:"name"`
	PropertySchema  *domain.ServiceSchema   `json:"propertySchema,omitempty"`
	LifecycleSchema *domain.LifecycleSchema `json:"lifecycleSchema,omitempty"`
	CreatedAt       JSONUTCTime             `json:"createdAt"`
	UpdatedAt       JSONUTCTime             `json:"updatedAt"`
}

// ServiceTypeToRes converts a domain.ServiceType to a ServiceTypeResponse
func ServiceTypeToRes(st *domain.ServiceType) *ServiceTypeRes {
	return &ServiceTypeRes{
		ID:              st.ID,
		Name:            st.Name,
		PropertySchema:  st.PropertySchema,
		LifecycleSchema: st.LifecycleSchema,
		CreatedAt:       JSONUTCTime(st.CreatedAt),
		UpdatedAt:       JSONUTCTime(st.UpdatedAt),
	}
}

// ValidateReq represents the request body for property validation
type ValidateReq struct {
	GroupID    properties.UUID `json:"groupId"`
	Properties map[string]any  `json:"properties"`
}

// ValidateRes represents the response body for property validation
type ValidateRes struct {
	Valid  bool                           `json:"valid"`
	Errors []domain.ValidationErrorDetail `json:"errors,omitempty"`
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
	params := &domain.ServicePropertyValidationParams{
		ServiceTypeID: serviceType.ID,
		GroupID:       req.GroupID,
		Properties:    req.Properties,
	}
	_, err = h.commander.ValidateServiceProperties(r.Context(), params)

	if err != nil {
		validationErrors, ok := err.(domain.ValidationError)
		if !ok {
			render.Render(w, r, ErrDomain(err))
			return
		}
		response := ValidateRes{
			Valid:  false,
			Errors: validationErrors.Errors,
		}
		render.JSON(w, r, response)
		return
	}

	response := ValidateRes{
		Valid:  true,
		Errors: []domain.ValidationErrorDetail{},
	}
	render.JSON(w, r, response)
}

// Adapter functions that convert request structs to commander method calls

func (h *ServiceTypeHandler) Create(ctx context.Context, req *CreateServiceTypeReq) (*domain.ServiceType, error) {
	params := domain.CreateServiceTypeParams{
		Name:            req.Name,
		PropertySchema:  req.PropertySchema,
		LifecycleSchema: req.LifecycleSchema,
	}
	return h.commander.Create(ctx, params)
}

func (h *ServiceTypeHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServiceTypeReq) (*domain.ServiceType, error) {
	params := domain.UpdateServiceTypeParams{
		ID:              id,
		Name:            req.Name,
		PropertySchema:  req.PropertySchema,
		LifecycleSchema: req.LifecycleSchema,
	}
	return h.commander.Update(ctx, params)
}
