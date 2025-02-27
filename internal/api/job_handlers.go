package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"fulcrumproject.org/core/internal/domain"
)

// JobHandler handles HTTP requests for jobs
type JobHandler struct {
	repo      domain.JobRepository
	agentRepo domain.AgentRepository
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(repo domain.JobRepository, agentRepo domain.AgentRepository) *JobHandler {
	return &JobHandler{
		repo:      repo,
		agentRepo: agentRepo,
	}
}

// Routes returns the router for job endpoints
func (h *JobHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Apply content type middleware
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Agent authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(AgentAuthMiddleware(h.agentRepo))

		r.Get("/pending", h.handleGetPendingJobs)     // For agents to poll for jobs
		r.Post("/{id}/claim", h.handleClaimJob)       // For agents to claim a job
		r.Post("/{id}/complete", h.handleCompleteJob) // For agents to mark a job as completed
		r.Post("/{id}/fail", h.handleFailJob)         // For agents to mark a job as failed
	})

	// Admin routes
	r.Get("/", h.handleList)
	r.Get("/{id}", h.handleGet)

	return r
}

// JobResponse represents the response for a job
type JobResponse struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	State        string           `json:"state"`
	AgentID      string           `json:"agentId"`
	ServiceID    string           `json:"serviceId"`
	Priority     int              `json:"priority"`
	RequestData  domain.JSON      `json:"requestData,omitempty"`
	ResultData   domain.JSON      `json:"resultData,omitempty"`
	ErrorMessage string           `json:"errorMessage,omitempty"`
	ClaimedAt    *JSONUTCTime     `json:"claimedAt,omitempty"`
	CompletedAt  *JSONUTCTime     `json:"completedAt,omitempty"`
	CreatedAt    JSONUTCTime      `json:"createdAt"`
	UpdatedAt    JSONUTCTime      `json:"updatedAt"`
	Agent        *AgentResponse   `json:"agent,omitempty"`
	Service      *ServiceResponse `json:"service,omitempty"`
}

// CompleteJobRequest represents a request to complete a job
type CompleteJobRequest struct {
	ResultData domain.JSON `json:"resultData"`
}

// FailJobRequest represents a request to fail a job
type FailJobRequest struct {
	ErrorMessage string `json:"errorMessage"`
}

// jobToResponse converts a job entity to a response
func jobToResponse(job *domain.Job) *JobResponse {
	resp := &JobResponse{
		ID:           job.ID.String(),
		Type:         string(job.Type),
		State:        string(job.State),
		AgentID:      job.AgentID.String(),
		ServiceID:    job.ServiceID.String(),
		Priority:     job.Priority,
		RequestData:  job.RequestData,
		ResultData:   job.ResultData,
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

	if job.Agent != nil {
		resp.Agent = agentToResponse(job.Agent)
	}

	if job.Service != nil {
		resp.Service = serviceToResponse(job.Service)
	}

	return resp
}

// handleList handles GET /jobs
func (h *JobHandler) handleList(w http.ResponseWriter, r *http.Request) {
	page, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.repo.List(r.Context(), page)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	jobs := result.Items
	jobResponses := make([]*JobResponse, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = jobToResponse(&job)
	}

	render.JSON(w, r, NewPageResponse(result, jobToResponse))
}

// handleGet handles GET /jobs/{id}
func (h *JobHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	job, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, jobToResponse(job))
}

// handleGetPendingJobs handles GET /jobs/pending
func (h *JobHandler) handleGetPendingJobs(w http.ResponseWriter, r *http.Request) {
	// Get authenticated agent from context (set by middleware)
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
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

	// Get pending jobs for this agent
	jobs, err := h.repo.GetPendingJobsForAgent(r.Context(), agent.ID, limit)
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
	// Get authenticated agent from context (set by middleware)
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}

	// Parse job ID from URL
	jobID, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Claim job for this agent
	if err := h.repo.ClaimJob(r.Context(), jobID, agent.ID); err != nil {
		if _, ok := err.(domain.NotFoundError); ok {
			render.Render(w, r, ErrNotFound())
			return
		}
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleCompleteJob handles POST /jobs/{id}/complete
func (h *JobHandler) handleCompleteJob(w http.ResponseWriter, r *http.Request) {
	// Get authenticated agent from context (set by middleware)
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}

	// Parse job ID from URL
	jobID, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Parse request body
	var req CompleteJobRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Complete the job
	if err := h.repo.CompleteJob(r.Context(), jobID, req.ResultData); err != nil {
		if _, ok := err.(domain.NotFoundError); ok {
			render.Render(w, r, ErrNotFound())
			return
		}
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleFailJob handles POST /jobs/{id}/fail
func (h *JobHandler) handleFailJob(w http.ResponseWriter, r *http.Request) {
	// Get authenticated agent from context (set by middleware)
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}

	// Parse job ID from URL
	jobID, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Parse request body
	var req FailJobRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Fail the job
	if err := h.repo.FailJob(r.Context(), jobID, req.ErrorMessage); err != nil {
		if _, ok := err.(domain.NotFoundError); ok {
			render.Render(w, r, ErrNotFound())
			return
		}
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
