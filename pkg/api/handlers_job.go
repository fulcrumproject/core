package api

import (
	"context"
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

type CompleteJobReq struct {
	AgentData       *properties.JSON `json:"agentData"`
	AgentInstanceID *string          `json:"agentInstanceId"`
	Properties      *properties.JSON `json:"properties,omitempty"`
}

type FailJobReq struct {
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
		).Get("/", List(h.querier, JobToRes))

		// Agent job polling - requires agent identity
		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
			middlewares.AuthzSimple(authz.ObjectTypeJob, authz.ActionListPending, h.authz),
		).Get("/pending", h.Pending)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get job - authorize using job's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, JobToRes))

			// Agent actions - require agent identity and authorize from job ID
			r.With(
				middlewares.MustHaveRoles(auth.RoleAgent),
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionClaim, h.authz, h.querier.AuthScope),
			).Post("/{id}/claim", CommandWithoutBody(h.commander.Claim))

			r.With(
				middlewares.MustHaveRoles(auth.RoleAgent),
				middlewares.DecodeBody[CompleteJobReq](),
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionComplete, h.authz, h.querier.AuthScope),
			).Post("/{id}/complete", Command(h.Complete))

			r.With(
				middlewares.MustHaveRoles(auth.RoleAgent),
				middlewares.DecodeBody[FailJobReq](),
				middlewares.AuthzFromID(authz.ObjectTypeJob, authz.ActionFail, h.authz, h.querier.AuthScope),
			).Post("/{id}/fail", Command(h.Fail))
		})
	}
}

// Pending handles GET /jobs/pending
func (h *JobHandler) Pending(w http.ResponseWriter, r *http.Request) {
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
	jobResponses := make([]*JobRes, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = JobToRes(job)
	}

	render.JSON(w, r, jobResponses)
}

// Adapter functions for standard handlers
func (h *JobHandler) Complete(ctx context.Context, id properties.UUID, req *CompleteJobReq) error {
	// Convert properties from JSON to map if provided
	var properties map[string]any
	if req.Properties != nil {
		properties = *req.Properties
	}

	params := domain.CompleteJobParams{
		JobID:           id,
		AgentData:       req.AgentData,
		AgentInstanceID: req.AgentInstanceID,
		Properties:      properties,
	}
	return h.commander.Complete(ctx, params)
}

func (h *JobHandler) Fail(ctx context.Context, id properties.UUID, req *FailJobReq) error {
	params := domain.FailJobParams{
		JobID:        id,
		ErrorMessage: req.ErrorMessage,
	}
	return h.commander.Fail(ctx, params)
}

// JobRes represents the response for a job
type JobRes struct {
	ID           properties.UUID  `json:"id"`
	ProviderID   properties.UUID  `json:"providerId"`
	ConsumerID   properties.UUID  `json:"consumerId"`
	AgentID      properties.UUID  `json:"agentId"`
	ServiceID    properties.UUID  `json:"serviceId"`
	Action       string           `json:"action"`
	Params       *properties.JSON `json:"params,omitempty"`
	Status       domain.JobStatus `json:"status"`
	Priority     int              `json:"priority"`
	ErrorMessage string           `json:"errorMessage,omitempty"`
	ClaimedAt    *JSONUTCTime     `json:"claimedAt,omitempty"`
	CompletedAt  *JSONUTCTime     `json:"completedAt,omitempty"`
	CreatedAt    JSONUTCTime      `json:"createdAt"`
	UpdatedAt    JSONUTCTime      `json:"updatedAt"`
	Service      *ServiceRes      `json:"service,omitempty"`
}

// JobToRes converts a job entity to a response
func JobToRes(job *domain.Job) *JobRes {
	resp := &JobRes{
		ID:           job.ID,
		AgentID:      job.AgentID,
		ProviderID:   job.ProviderID,
		ConsumerID:   job.ConsumerID,
		ServiceID:    job.ServiceID,
		Action:       job.Action,
		Params:       job.Params,
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
		resp.Service = ServiceToRes(job.Service)
	}
	return resp
}
