package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"fulcrumproject.org/core/internal/api"
	"fulcrumproject.org/core/internal/config"
	"fulcrumproject.org/core/internal/database"
	"fulcrumproject.org/core/internal/domain"
	"fulcrumproject.org/core/internal/logging"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Builder().LoadFile(configPath).WithEnv().Build()
	if err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Setup structured logger
	logger := logging.NewLogger(cfg)
	slog.SetDefault(logger)

	// Initialize database
	db, err := database.NewConnection(&cfg.DBConfig)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	// Seed with basic data if empty
	if err := database.Seed(db); err != nil {
		slog.Error("Failed to seed the database", "error", err)
		os.Exit(1)
	}

	// Initialize the store
	store := database.NewGormStore(db)

	// Initialize commanders
	// Initialize audit commander first since it's needed by other commanders
	auditEntryCmd := domain.NewAuditEntryCommander(store)

	serviceCmd := domain.NewServiceCommander(store, auditEntryCmd)
	serviceGroupCmd := domain.NewServiceGroupCommander(store, auditEntryCmd)
	participantCmd := domain.NewParticipantCommander(store, auditEntryCmd)
	jobCmd := domain.NewJobCommander(store, auditEntryCmd)
	metricEntryCmd := domain.NewMetricEntryCommander(store)
	metricTypeCmd := domain.NewMetricTypeCommander(store, auditEntryCmd)
	agentCmd := domain.NewAgentCommander(store, auditEntryCmd)
	tokenCmd := domain.NewTokenCommander(store, auditEntryCmd)

	// Initialize auth and authorizer
	auth := database.NewTokenAuthenticator(store)
	authz := domain.NewDefaultRuleAuthorizer()

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(store.AgentTypeRepo(), authz)
	serviceTypeHandler := api.NewServiceTypeHandler(store.ServiceTypeRepo(), authz)
	participantHandler := api.NewParticipantHandler(store.ParticipantRepo(), participantCmd, authz)
	agentHandler := api.NewAgentHandler(store.AgentRepo(), agentCmd, authz)
	serviceGroupHandler := api.NewServiceGroupHandler(store.ServiceGroupRepo(), serviceGroupCmd, authz)
	serviceHandler := api.NewServiceHandler(store.ServiceRepo(), store.AgentRepo(), store.ServiceGroupRepo(), serviceCmd, authz)
	jobHandler := api.NewJobHandler(store.JobRepo(), jobCmd, authz)
	metricTypeHandler := api.NewMetricTypeHandler(store.MetricTypeRepo(), metricTypeCmd, authz)
	metricEntryHandler := api.NewMetricEntryHandler(store.MetricEntryRepo(), store.ServiceRepo(), metricEntryCmd, authz)
	auditEntryHandler := api.NewAuditEntryHandler(store.AuditEntryRepo(), auditEntryCmd, authz)
	tokenHandler := api.NewTokenHandler(store.TokenRepo(), tokenCmd, store.AgentRepo(), authz)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(
		middleware.RequestID,
		middleware.RequestLogger(&logging.SlogFormatter{Logger: logger}),
		middleware.RealIP,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)

	authMiddleware := api.Auth(auth)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Route("/agent-types", agentTypeHandler.Routes())
		r.Route("/service-types", serviceTypeHandler.Routes())
		r.Route("/participants", participantHandler.Routes())
		r.Route("/agents", agentHandler.Routes())
		r.Route("/service-groups", serviceGroupHandler.Routes())
		r.Route("/services", serviceHandler.Routes())
		r.Route("/metric-types", metricTypeHandler.Routes())
		r.Route("/metric-entries", metricEntryHandler.Routes())
		r.Route("/audit-entries", auditEntryHandler.Routes())
		r.Route("/jobs", jobHandler.Routes())
		r.Route("/tokens", tokenHandler.Routes())
	})

	// Setup background job maintenance worker
	// go JobMainenanceTask(&cfg.JobConfig, store, serviceCmd)

	// Setup background worker to mark inactive agents as disconnected
	go DisconnectUnhealthyAgentsTask(&cfg.AgentConfig, store)

	// Start server
	slog.Info("Server starting", "port", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

// TODO move to proper worker
func DisconnectUnhealthyAgentsTask(cfg *config.AgentConfig, store domain.Store) {
	ticker := time.NewTicker(cfg.HealthTimeout)
	for range ticker.C {
		ctx := context.Background()
		slog.Info("Checking agents health")
		disconnectedCount, err := store.AgentRepo().MarkInactiveAgentsAsDisconnected(ctx, cfg.HealthTimeout)
		if err != nil {
			slog.Error("Error marking inactive agents as disconnected", "error", err)
		} else if disconnectedCount > 0 {
			slog.Info("Marked inactive agents as disconnected", "count", disconnectedCount)
		}
	}
}

// TODO move to proper worker
func JobMainenanceTask(cfg *config.JobConfig, store domain.Store, serviceCmd domain.ServiceCommander) {
	ticker := time.NewTicker(cfg.Maintenance)
	for range ticker.C {
		ctx := context.Background()

		// Fail timeout jobs an services
		slog.Info("Checking timeout jobs")
		failedCount, err := serviceCmd.FailTimeoutServicesAndJobs(ctx, cfg.Timeout)
		if err != nil {
			slog.Error("Failed to timeout jobs and services", "error", err)
		} else {
			slog.Info("Timeout jobs processed", "failed_count", failedCount)
		}

		// Delete completed/failed old jobs
		slog.Info("Deleting old jobs")
		deletedCount, err := store.JobRepo().DeleteOldCompletedJobs(ctx, cfg.Retention)
		if err != nil {
			slog.Error("Failed to delete old jobs", "error", err)
		} else {
			slog.Info("Old jobs deleted", "count", deletedCount)
		}
	}
}
