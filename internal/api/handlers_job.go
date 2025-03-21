package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"fulcrumproject.org/core/internal/domain"
)

// JobHandler handles HTTP requests for jobs
type JobHandler struct {
	querier   domain.JobQuerier
	commander domain.JobCommander
	authz     domain.Authorizer
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(
	querier domain.JobQuerier,
	commander domain.JobCommander,
	authz domain.Authorizer,
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
		// Admin routes
		r.Get("/", h.handleList)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
		})

		// Agent job polling route
		r.Get("/pending", h.handleGetPendingJobs)

		// Agent job action routes - all require agent authentication and UUID validation
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Post("/{id}/claim", h.handleClaimJob)
			r.Post("/{id}/complete", h.handleCompleteJob)
			r.Post("/{id}/fail", h.handleFailJob) // For agents to mark a job as failed
		})
	}
}

// handleList handles GET /jobs
func (h *JobHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectJob, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	page, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), id.Scope(), page)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, jobToResponse))
}

// handleGet handles GET /jobs/{id}
func (h *JobHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	job, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, jobToResponse(job))
}

// handleGetPendingJobs handles GET /jobs/pending
func (h *JobHandler) handleGetPendingJobs(w http.ResponseWriter, r *http.Request) {
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectJob, domain.ActionListPending, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
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
	agentID, err := MustGetAgentID(r.Context())
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	// Get pending jobs for this agent
	jobs, err := h.querier.GetPendingJobsForAgent(r.Context(), agentID, limit)
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
	// Get job ID from URL
	jobID := MustGetID(r)
	_, err := h.authorize(r.Context(), jobID, domain.ActionClaim)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	// Claim job for this agent
	if err := h.commander.Claim(r.Context(), jobID); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCompleteJob handles POST /jobs/{id}/complete
func (h *JobHandler) handleCompleteJob(w http.ResponseWriter, r *http.Request) {
	// Get job ID from URL
	jobID := MustGetID(r)
	// Authorize
	_, err := h.authorize(r.Context(), jobID, domain.ActionComplete)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	// Parse request body
	var req struct {
		Resources  *domain.JSON `json:"resources"`
		ExternalID *string      `json:"externalID"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	// Complete the job
	if err := h.commander.Complete(r.Context(), jobID, req.Resources, req.ExternalID); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleFailJob handles POST /jobs/{id}/fail
func (h *JobHandler) handleFailJob(w http.ResponseWriter, r *http.Request) {
	// Get job ID from URL
	jobID := MustGetID(r)
	_, err := h.authorize(r.Context(), jobID, domain.ActionFail)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	// Parse request body
	var p struct {
		ErrorMessage string `json:"errorMessage"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	// Fail the job
	if err := h.commander.Fail(r.Context(), jobID, p.ErrorMessage); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// JobResponse represents the response for a job
type JobResponse struct {
	ID           domain.UUID          `json:"id"`
	ProviderID   domain.UUID          `json:"providerId"`
	BrokerID     domain.UUID          `json:"brokerId"`
	AgentID      domain.UUID          `json:"agentId"`
	ServiceID    domain.UUID          `json:"serviceId"`
	Action       domain.ServiceAction `json:"action"`
	State        domain.JobState      `json:"state"`
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
		BrokerID:     job.BrokerID,
		ServiceID:    job.ServiceID,
		Action:       job.Action,
		State:        job.State,
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

func (h *JobHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectJob, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
