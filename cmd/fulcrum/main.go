package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/config"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/health"
	"github.com/fulcrumproject/core/pkg/keycloak"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/utils/confbuilder"
	"github.com/fulcrumproject/utils/logging"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := confbuilder.New(config.Default).
		EnvPrefix(config.EnvPrefix).
		EnvFiles(".env").
		File(configPath).
		Build()
	if err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Setup structured logger
	logger := logging.NewLogger(&cfg.LogConfig)
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
	serviceCmd := domain.NewServiceCommander(store)
	serviceGroupCmd := domain.NewServiceGroupCommander(store)
	participantCmd := domain.NewParticipantCommander(store)
	jobCmd := domain.NewJobCommander(store)
	metricEntryCmd := domain.NewMetricEntryCommander(store)
	metricTypeCmd := domain.NewMetricTypeCommander(store)
	agentCmd := domain.NewAgentCommander(store)
	tokenCmd := domain.NewTokenCommander(store)

	// Initialize authenticators
	authenticators := []auth.Authenticator{}

	for _, authType := range cfg.Authenticators {
		switch strings.TrimSpace(authType) {
		case "token":
			tokenAuth := database.NewTokenAuthenticator(store)
			authenticators = append(authenticators, tokenAuth)
			slog.Info("Token authentication enabled")
		case "oauth":
			ctx := context.Background()
			oauthAuth, err := keycloak.NewAuthenticator(ctx, &cfg.OAuthConfig)
			if err != nil {
				slog.Error("Failed to initialize OAuth authenticator", "error", err)
				os.Exit(1)
			}
			authenticators = append(authenticators, oauthAuth)
			slog.Info("OAuth authentication enabled", "issuer", cfg.OAuthConfig.GetIssuer())
		default:
			slog.Warn("Unknown authenticator type in config", "type", authType)
		}
	}

	if len(authenticators) == 0 {
		slog.Warn("No authenticators enabled in configuration. API will be unprotected.")
		// Optionally, you might want to exit or use a no-op authenticator
	}

	ath := auth.NewCompositeAuthenticator(authenticators...)

	athz := auth.NewRuleBasedAuthorizer(authz.Rules)

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(store.AgentTypeRepo(), athz)
	serviceTypeHandler := api.NewServiceTypeHandler(store.ServiceTypeRepo(), athz)
	participantHandler := api.NewParticipantHandler(store.ParticipantRepo(), participantCmd, athz)
	agentHandler := api.NewAgentHandler(store.AgentRepo(), agentCmd, athz)
	serviceGroupHandler := api.NewServiceGroupHandler(store.ServiceGroupRepo(), serviceGroupCmd, athz)
	serviceHandler := api.NewServiceHandler(store.ServiceRepo(), store.AgentRepo(), store.ServiceGroupRepo(), serviceCmd, athz)
	jobHandler := api.NewJobHandler(store.JobRepo(), jobCmd, athz)
	metricTypeHandler := api.NewMetricTypeHandler(store.MetricTypeRepo(), metricTypeCmd, athz)
	metricEntryHandler := api.NewMetricEntryHandler(store.MetricEntryRepo(), store.ServiceRepo(), metricEntryCmd, athz)
	eventSubscriptionCmd := domain.NewEventSubscriptionCommander(store)
	eventHandler := api.NewEventHandler(store.EventRepo(), eventSubscriptionCmd, athz)
	tokenHandler := api.NewTokenHandler(store.TokenRepo(), tokenCmd, store.AgentRepo(), athz)

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

	authMiddleware := middlewares.Auth(ath)

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
		r.Route("/events", eventHandler.Routes())
		r.Route("/jobs", jobHandler.Routes())
		r.Route("/tokens", tokenHandler.Routes())
	})

	// Setup background job maintenance worker
	go JobMaintenanceTask(&cfg.JobConfig, store, serviceCmd)

	// Setup background worker to mark inactive agents as disconnected
	go DisconnectUnhealthyAgentsTask(&cfg.AgentConfig, store)

	// Initialize health checker and handlers
	healthDeps := &health.PrimaryDependencies{
		DB:             db,
		Authenticators: authenticators,
	}
	healthChecker := health.NewHealthChecker(healthDeps)
	healthHandler := health.NewHandler(healthChecker)

	// Setup health router
	healthRouter := chi.NewRouter()
	healthRouter.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)
	healthRouter.Get("/healthz", healthHandler.HealthHandler)
	healthRouter.Get("/ready", healthHandler.ReadinessHandler)

	// Start health server in a goroutine
	go func() {
		slog.Info("Health server starting", "port", cfg.HealthPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HealthPort), healthRouter); err != nil {
			slog.Error("Failed to start health server", "error", err)
			os.Exit(1)
		}
	}()

	// Start main API server
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
func JobMaintenanceTask(cfg *config.JobConfig, store domain.Store, serviceCmd domain.ServiceCommander) {
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
