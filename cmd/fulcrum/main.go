package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"fulcrumproject.org/core/internal/api"
	"fulcrumproject.org/core/internal/config"
	"fulcrumproject.org/core/internal/database"
	"fulcrumproject.org/core/internal/domain"
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
			log.Fatalf("Failed to load configuration from file: %v", err)
		}
		log.Printf("Loaded configuration from %s", *configPath)
	}
	// Override with environment variables
	if err := cfg.LoadFromEnv(); err != nil {
		log.Fatalf("Failed to load configuration from environment: %v", err)
	}

	// Initialize database
	db, err := database.NewConnection(&cfg.DBConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Seed with basic data if empty
	if err := database.Seed(db); err != nil {
		log.Fatalf("Failed to seed the database: %v", err)
	}

	// Initialize the store
	store := database.NewStore(db)

	// Initialize commanders
	serviceCmd := domain.NewServiceCommander(store)
	serviceGroupCmd := domain.NewServiceGroupCommander(store)
	providerCmd := domain.NewProviderCommander(store)
	jobCmd := domain.NewJobCommander(store)
	metricEntryCmd := domain.NewMetricEntryCommander(store)
	metricTypeCmd := domain.NewMetricTypeCommander(store)
	auditEntryCmd := domain.NewAuditEntryCommander(store)
	agentCmd := domain.NewAgentCommander(store)

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

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)
	// TODO refactor with global auth
	agentAuthMiddleware := api.AgentAuthMiddleware(store.AgentRepo())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/providers", providerHandler.Routes())
		r.Route("/agent-types", agentTypeHandler.Routes())
		r.Route("/service-types", serviceTypeHandler.Routes())
		r.Route("/agents", agentHandler.Routes(agentAuthMiddleware))
		r.Route("/service-groups", serviceGroupHandler.Routes())
		r.Route("/services", serviceHandler.Routes())
		r.Route("/metric-types", metricTypeHandler.Routes())
		r.Route("/metric-entries", metricEntryHandler.Routes(agentAuthMiddleware))
		r.Route("/audit-entries", auditEntryHandler.Routes())
		r.Route("/jobs", jobHandler.Routes(agentAuthMiddleware))
	})

	// Setup background job maintenance worker
	go JobMainenanceTask(&cfg.JobConfig, store, serviceCmd)

	// Setup background worker to mark inactive agents as disconnected
	go DisconnectUnhealthyAgentsTask(&cfg.AgentConfig, store)

	// Start server
	log.Printf("Server starting on port %d...", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func DisconnectUnhealthyAgentsTask(cfg *config.AgentConfig, store domain.Store) {
	ticker := time.NewTicker(cfg.HealthTimeout)
	for range ticker.C {
		ctx := context.Background()
		log.Println("Checking agents health ...")
		disconnectedCount, err := store.AgentRepo().MarkInactiveAgentsAsDisconnected(ctx, cfg.HealthTimeout)
		if err != nil {
			log.Printf("Error marking inactive agents as disconnected: %v", err)
		} else if disconnectedCount > 0 {
			log.Printf("Marked %d inactive agents as disconnected", disconnectedCount)
		}
	}
}

func JobMainenanceTask(cfg *config.JobConfig, store domain.Store, serviceCmd *domain.ServiceCommander) {
	ticker := time.NewTicker(cfg.Maintenance)
	for range ticker.C {
		ctx := context.Background()

		// Fail timeout jobs an services
		log.Println("Checking timeout jobs ...")
		failedCount, err := serviceCmd.FailTimeoutServicesAndJobs(ctx, cfg.Timeout)
		if err != nil {
			log.Printf("Failed to timeout jobs and services: %v", err)
		} else {
			log.Printf("Done fail %d timeout jobs.", failedCount)
		}

		// Delete completed/failed old jobs
		log.Println("Deleting old jobs ...")
		deletedCount, err := store.JobRepo().DeleteOldCompletedJobs(ctx, cfg.Retention)
		if err != nil {
			log.Printf("Failed delete old jobs: %v", err)
		} else {
			log.Printf("Deleted %d old jobs.", deletedCount)
		}
	}
}
