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
	cfg := config.DefaultConfig()
	var err error
	if configPath != nil && *configPath != "" {
		// Load from file if specified
		cfg, err = config.LoadFromFile(*configPath)
		if err != nil {
			fmt.Printf("Failed to load configuration from file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded configuration from %s\n", *configPath)
	}
	// Override with environment variables
	if err := cfg.LoadFromEnv(); err != nil {
		fmt.Printf("Failed to load configuration from environment: %v\n", err)
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
	store := database.NewStore(db)

	// Initialize authenticator and authorizer
	authenticator := database.NewTokenAuthenticator(store)

	// Initialize commanders
	serviceCmd := domain.NewServiceCommander(store)
	serviceGroupCmd := domain.NewServiceGroupCommander(store)
	providerCmd := domain.NewProviderCommander(store)
	jobCmd := domain.NewJobCommander(store)
	metricEntryCmd := domain.NewMetricEntryCommander(store)
	metricTypeCmd := domain.NewMetricTypeCommander(store)
	auditEntryCmd := domain.NewAuditEntryCommander(store)
	agentCmd := domain.NewAgentCommander(store)
	brokerCmd := domain.NewBrokerCommander(store)
	tokenCmd := domain.NewTokenCommander(store)

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(store.AgentTypeRepo())
	serviceTypeHandler := api.NewServiceTypeHandler(store.ServiceTypeRepo())
	providerHandler := api.NewProviderHandler(store.ProviderRepo(), providerCmd)
	agentHandler := api.NewAgentHandler(store.AgentRepo(), agentCmd)
	serviceGroupHandler := api.NewServiceGroupHandler(store.ServiceGroupRepo(), serviceGroupCmd)
	serviceHandler := api.NewServiceHandler(store.ServiceRepo(), serviceCmd)
	jobHandler := api.NewJobHandler(store.JobRepo(), jobCmd)
	metricTypeHandler := api.NewMetricTypeHandler(store.MetricTypeRepo(), metricTypeCmd)
	metricEntryHandler := api.NewMetricEntryHandler(store.MetricEntryRepo(), metricEntryCmd)
	auditEntryHandler := api.NewAuditEntryHandler(store.AuditEntryRepo(), auditEntryCmd)
	brokerHandler := api.NewBrokerHandler(store.BrokerRepo(), brokerCmd)
	tokenHandler := api.NewTokenHandler(store.TokenRepo(), tokenCmd)

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

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	// Create auth middleware
	authz := api.AuthzMiddleware(authenticator, domain.NewDefaultRuleAuthorizer())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Register API endpoints
		r.Route("/agent-types", agentTypeHandler.Routes(authz))
		r.Route("/service-types", serviceTypeHandler.Routes(authz))
		r.Route("/providers", providerHandler.Routes(authz))
		r.Route("/agents", agentHandler.Routes(authz))
		r.Route("/service-groups", serviceGroupHandler.Routes(authz))
		r.Route("/services", serviceHandler.Routes(authz))
		r.Route("/metric-types", metricTypeHandler.Routes(authz))
		r.Route("/metric-entries", metricEntryHandler.Routes(authz))
		r.Route("/audit-entries", auditEntryHandler.Routes(authz))
		r.Route("/jobs", jobHandler.Routes(authz))
		r.Route("/brokers", brokerHandler.Routes(authz))
		r.Route("/tokens", tokenHandler.Routes(authz))
	})

	// Setup background job maintenance worker
	go JobMainenanceTask(&cfg.JobConfig, store, serviceCmd)

	// Setup background worker to mark inactive agents as disconnected
	go DisconnectUnhealthyAgentsTask(&cfg.AgentConfig, store)

	// Start server
	slog.Info("Server starting", "port", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

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
