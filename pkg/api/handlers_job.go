package api

import (
	"net/http"
	"strconv"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
)

type CompleteJobRequest struct {
	Resources  *properties.JSON `json:"resources"`
	ExternalID *string          `json:"externalID"`
}

type FailJobRequest struct {
	ErrorMessage string `json:"errorMessage"`
}

// JobHandler handles HTTP requests for jobs
type JobHandler struct {
	querier   domain.JobQuerier
	commander domain.JobCommander
	authz     auth.Authorizer
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(
	querier domain.JobQuerier,
	commander domain.JobCommander,
	authz auth.Authorizer,
) *JobHandler {
	return &JobHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router for job endpoints
func (h *JobHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List jobs - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeJob, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Agent job polling - requires agent identity
		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
			middlewares.AuthzSimple(authz.ObjectTypeJob, authz.ActionListPending, h.authz),
		).Get("/pending", h.handleGetPendingJobs)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get job - authorize using job's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Agent actions - require agent identity and authorize from job ID
			r.With(
				middlewares.MustHaveRoles(auth.RoleAgent),
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionComplete, h.authz, h.querier.AuthScope),
			).Post("/{id}/claim", h.handleClaimJob)

			r.With(
				middlewares.MustHaveRoles(auth.RoleAgent),
				middlewares.DecodeBody[CompleteJobRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionComplete, h.authz, h.querier.AuthScope),
			).Post("/{id}/complete", h.handleCompleteJob)

			r.With(
				middlewares.MustHaveRoles(auth.RoleAgent),
				middlewares.DecodeBody[FailJobRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionFail, h.authz, h.querier.AuthScope),
			).Post("/{id}/fail", h.handleFailJob)
		})
	}
}

// handleList handles GET /jobs
func (h *JobHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())
	page, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, page)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, jobToResponse))
}

// handleGet handles GET /jobs/{id}
func (h *JobHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	job, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, jobToResponse(job))
}

// handleGetPendingJobs handles GET /jobs/pending
func (h *JobHandler) handleGetPendingJobs(w http.ResponseWriter, r *http.Request) {
	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get agent ID from context
	agentID := auth.MustGetIdentity(r.Context()).Scope.AgentID

	// Get pending jobs for this agent
	jobs, err := h.querier.GetPendingJobsForAgent(r.Context(), *agentID, limit)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	// Convert to response
	jobResponses := make([]*JobResponse, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = jobToResponse(job)
	}

	render.JSON(w, r, jobResponses)
}

// handleClaimJob handles POST /jobs/{id}/claim
func (h *JobHandler) handleClaimJob(w http.ResponseWriter, r *http.Request) {
	jobID := middlewares.MustGetID(r.Context())

	// Claim job for this agent
	if err := h.commander.Claim(r.Context(), jobID); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleCompleteJob handles POST /jobs/{id}/complete
func (h *JobHandler) handleCompleteJob(w http.ResponseWriter, r *http.Request) {
	jobID := middlewares.MustGetID(r.Context())
	req := middlewares.MustGetBody[CompleteJobRequest](r.Context())

	// Complete the job
	if err := h.commander.Complete(r.Context(), jobID, req.Resources, req.ExternalID); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleFailJob handles POST /jobs/{id}/fail
func (h *JobHandler) handleFailJob(w http.ResponseWriter, r *http.Request) {
	jobID := middlewares.MustGetID(r.Context())
	req := middlewares.MustGetBody[FailJobRequest](r.Context())

	// Fail the job
	if err := h.commander.Fail(r.Context(), jobID, req.ErrorMessage); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// JobResponse represents the response for a job
type JobResponse struct {
	ID           properties.UUID      `json:"id"`
	ProviderID   properties.UUID      `json:"providerId"`
	ConsumerID   properties.UUID      `json:"consumerId"`
	AgentID      properties.UUID      `json:"agentId"`
	ServiceID    properties.UUID      `json:"serviceId"`
	Action       domain.ServiceAction `json:"action"`
	Status       domain.JobStatus     `json:"status"`
	Priority     int                  `json:"priority"`
	ErrorMessage string               `json:"errorMessage,omitempty"`
	ClaimedAt    *JSONUTCTime         `json:"claimedAt,omitempty"`
	CompletedAt  *JSONUTCTime         `json:"completedAt,omitempty"`
	CreatedAt    JSONUTCTime          `json:"createdAt"`
	UpdatedAt    JSONUTCTime          `json:"updatedAt"`
	Service      *ServiceResponse     `json:"service,omitempty"`
}

// jobToResponse converts a job entity to a response
func jobToResponse(job *domain.Job) *JobResponse {
	resp := &JobResponse{
		ID:           job.ID,
		AgentID:      job.AgentID,
		ProviderID:   job.ProviderID,
		ConsumerID:   job.ConsumerID,
		ServiceID:    job.ServiceID,
		Action:       job.Action,
		Status:       job.Status,
		Priority:     job.Priority,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    JSONUTCTime(job.CreatedAt),
		UpdatedAt:    JSONUTCTime(job.UpdatedAt),
	}
	if job.ClaimedAt != nil {
		resp.ClaimedAt = (*JSONUTCTime)(job.ClaimedAt)
	}
	if job.CompletedAt != nil {
		resp.CompletedAt = (*JSONUTCTime)(job.CompletedAt)
	}
	if job.Service != nil {
		resp.Service = serviceToResponse(job.Service)
	}
	return resp
}
