package health

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

// Res represents the HTTP response for health endpoints
type Res struct {
	Status string `json:"status"`
}

// Handler provides HTTP handlers for health endpoints
type Handler struct {
	checker Checker
}

// NewHandler creates a new health handler
func NewHandler(checker Checker) *Handler {
	return &Handler{
		checker: checker,
	}
}

// HealthHandler handles GET /healthz requests
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result := h.checker.CheckHealth(ctx)

	response := Res{
		Status: string(result.Status),
	}

	if result.Status == StatusUP {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	render.JSON(w, r, response)
}

// ReadinessHandler handles GET /ready requests
func (h *Handler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result := h.checker.CheckReadiness(ctx)

	response := Res{
		Status: string(result.Status),
	}

	if result.Status == StatusUP {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	render.JSON(w, r, response)
}
