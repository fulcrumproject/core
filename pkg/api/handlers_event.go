package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fulcrumproject.org/core/pkg/authz"
	"fulcrumproject.org/core/pkg/domain"
	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

const (
	// Event lease configuration constants
	DefaultLeaseDurationSeconds = 300  // 5 minutes
	MaxLeaseDurationSeconds     = 3600 // 1 hour
	MinLeaseDurationSeconds     = 30   // 30 seconds
	DefaultEventLimit           = 100  // default number of events to fetch
	MaxEventLimit               = 1000 // maximum number of events to fetch
	MinEventLimit               = 1    // minimum number of events to fetch
)

type EventHandler struct {
	querier                    domain.EventQuerier
	eventSubscriptionCommander domain.EventSubscriptionCommander
	authz                      auth.Authorizer
}

func NewEventHandler(
	querier domain.EventQuerier,
	eventSubscriptionCommander domain.EventSubscriptionCommander,
	authz auth.Authorizer,
) *EventHandler {
	return &EventHandler{
		querier:                    querier,
		eventSubscriptionCommander: eventSubscriptionCommander,
		authz:                      authz,
	}
}

// Routes returns the router with all event entry routes registered
func (h *EventHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeEvent, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Event consumption endpoint with leasing - requires admin role
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeEvent, authz.ActionLease, h.authz),
		).Post("/lease", h.handleLease)

		// Event acknowledgement endpoint - requires admin role
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeEvent, authz.ActionAck, h.authz),
		).Post("/ack", h.handleAcknowledge)
	}
}

func (h *EventHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	render.JSON(w, r, NewPageResponse(result, eventToResponse))
}

// EventResponse represents the response body for event entry operations
type EventResponse struct {
	ID             properties.UUID      `json:"id"`
	SequenceNumber int64                `json:"sequenceNumber"`
	InitiatorType  domain.InitiatorType `json:"initiatorType"`
	InitiatorID    string               `json:"initiatorId"`
	Type           domain.EventType     `json:"type"`
	Properties     properties.JSON      `json:"properties"`
	ProviderID     *properties.UUID     `json:"providerId,omitempty"`
	AgentID        *properties.UUID     `json:"agentId,omitempty"`
	ConsumerID     *properties.UUID     `json:"consumerId,omitempty"`
	CreatedAt      JSONUTCTime          `json:"createdAt"`
	UpdatedAt      JSONUTCTime          `json:"updatedAt"`
}

// eventToResponse converts a domain.Event to an EventResponse
func eventToResponse(ae *domain.Event) *EventResponse {
	return &EventResponse{
		ID:             ae.ID,
		SequenceNumber: ae.SequenceNumber,
		InitiatorType:  ae.InitiatorType,
		InitiatorID:    ae.InitiatorID,
		Type:           ae.Type,
		Properties:     ae.Payload,
		ProviderID:     ae.ProviderID,
		AgentID:        ae.AgentID,
		ConsumerID:     ae.ConsumerID,
		CreatedAt:      JSONUTCTime(ae.CreatedAt),
		UpdatedAt:      JSONUTCTime(ae.UpdatedAt),
	}
}

// EventLeaseRequest represents the request body for event lease operations
type EventLeaseRequest struct {
	SubscriberID         string `json:"subscriberId" validate:"required"`
	InstanceID           string `json:"instanceId" validate:"required"`
	LeaseDurationSeconds *int   `json:"leaseDurationSeconds,omitempty"`
	Limit                *int   `json:"limit,omitempty"`
}

// Bind implements the render.Binder interface for EventLeaseRequest
func (req *EventLeaseRequest) Bind(r *http.Request) error {
	if req.SubscriberID == "" {
		return fmt.Errorf("subscriberId is required")
	}
	if req.InstanceID == "" {
		return fmt.Errorf("instanceId is required")
	}
	return nil
}

// EventLeaseResponse represents the response body for event lease operations
type EventLeaseResponse struct {
	Events                     []EventResponse `json:"events"`
	LeaseExpiresAt             JSONUTCTime     `json:"leaseExpiresAt"`
	LastEventSequenceProcessed int64           `json:"lastEventSequenceProcessed"`
}

func (h *EventHandler) handleLease(w http.ResponseWriter, r *http.Request) {
	var req EventLeaseRequest
	if err := render.Bind(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Set defaults and enforce limits
	leaseDurationSeconds := DefaultLeaseDurationSeconds
	if req.LeaseDurationSeconds != nil {
		leaseDurationSeconds = *req.LeaseDurationSeconds
		if leaseDurationSeconds > MaxLeaseDurationSeconds {
			leaseDurationSeconds = MaxLeaseDurationSeconds
		}
		if leaseDurationSeconds < MinLeaseDurationSeconds {
			leaseDurationSeconds = MinLeaseDurationSeconds
		}
	}

	limit := DefaultEventLimit
	if req.Limit != nil {
		limit = *req.Limit
		if limit > MaxEventLimit {
			limit = MaxEventLimit
		}
		if limit < MinEventLimit {
			limit = MinEventLimit
		}
	}

	ctx := r.Context()

	// Try to acquire or renew the lease
	subscription, err := h.eventSubscriptionCommander.AcquireLease(
		ctx,
		req.SubscriberID,
		req.InstanceID,
		time.Duration(leaseDurationSeconds)*time.Second,
	)
	if err != nil {
		// Check if it's a conflict error (lease held by another instance)
		var invalidInputErr domain.InvalidInputError
		if errors.As(err, &invalidInputErr) &&
			(strings.Contains(err.Error(), "lease is already held") ||
				strings.Contains(err.Error(), "lease is not owned")) {
			render.Render(w, r, &ErrResponse{
				Err:            err,
				HTTPStatusCode: 409, // Conflict
				StatusText:     "Lease Conflict",
				ErrorText:      err.Error(),
			})
			return
		}
		render.Render(w, r, ErrDomain(err))
		return
	}

	// Fetch events starting from the last processed sequence
	events, err := h.querier.ListFromSequence(ctx, subscription.LastEventSequenceProcessed, limit)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	// Convert events to response format
	eventResponses := make([]EventResponse, len(events))
	for i, event := range events {
		eventResponses[i] = *eventToResponse(event)
	}

	response := EventLeaseResponse{
		Events:                     eventResponses,
		LeaseExpiresAt:             JSONUTCTime(*subscription.LeaseExpiresAt),
		LastEventSequenceProcessed: subscription.LastEventSequenceProcessed,
	}

	render.JSON(w, r, response)
}

// EventAckRequest represents the request body for event acknowledgement
type EventAckRequest struct {
	SubscriberID               string `json:"subscriberId"`
	InstanceID                 string `json:"instanceId"`
	LastEventSequenceProcessed int64  `json:"lastEventSequenceProcessed"`
}

// Bind implements the render.Binder interface for EventAckRequest
func (req *EventAckRequest) Bind(r *http.Request) error {
	if req.SubscriberID == "" {
		return fmt.Errorf("subscriberId is required")
	}
	if req.InstanceID == "" {
		return fmt.Errorf("instanceId is required")
	}
	if req.LastEventSequenceProcessed <= 0 {
		return fmt.Errorf("lastEventSequenceProcessed must be greater than 0")
	}
	return nil
}

// EventAckResponse represents the response body for event acknowledgement
type EventAckResponse struct {
	LastEventSequenceProcessed int64 `json:"lastEventSequenceProcessed"`
}

func (h *EventHandler) handleAcknowledge(w http.ResponseWriter, r *http.Request) {
	var req EventAckRequest
	if err := render.Bind(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	ctx := r.Context()

	// Acknowledge the events by updating progress with lease validation
	subscription, err := h.eventSubscriptionCommander.AcknowledgeEvents(
		ctx,
		req.SubscriberID,
		req.InstanceID,
		req.LastEventSequenceProcessed,
	)
	if err != nil {
		// Check if it's a conflict error (lease not held by this instance)
		var invalidInputErr domain.InvalidInputError
		if errors.As(err, &invalidInputErr) &&
			(strings.Contains(err.Error(), "no active lease") ||
				strings.Contains(err.Error(), "lease is not owned") ||
				strings.Contains(err.Error(), "cannot acknowledge sequence")) {
			render.Render(w, r, &ErrResponse{
				Err:            err,
				HTTPStatusCode: 409, // Conflict
				StatusText:     "Acknowledgement Conflict",
				ErrorText:      err.Error(),
			})
			return
		}
		render.Render(w, r, ErrDomain(err))
		return
	}

	response := EventAckResponse{
		LastEventSequenceProcessed: subscription.LastEventSequenceProcessed,
	}

	render.JSON(w, r, response)
}
