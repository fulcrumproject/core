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

	// Initialize repositories used also as queriers
	agentTypeRepo := database.NewAgentTypeRepository(db)
	serviceTypeRepo := database.NewServiceTypeRepository(db)
	providerRepo := database.NewProviderRepository(db)
	agentRepo := database.NewAgentRepository(db)
	serviceGroupRepo := database.NewServiceGroupRepository(db)
	serviceRepo := database.NewServiceRepository(db)
	metricTypeRepo := database.NewMetricTypeRepository(db)
	metricEntryRepo := database.NewMetricEntryRepository(db)
	auditEntryRepo := database.NewAuditEntryRepository(db)
	jobRepo := database.NewJobRepository(db)

	// Initialize commanders
	serviceCmd := domain.NewServiceCommander(serviceRepo, jobRepo)
	serviceGroupCmd := domain.NewServiceGroupCommander(serviceGroupRepo, serviceRepo)
	providerCmd := domain.NewProviderCommander(providerRepo, agentRepo)
	jobCmd := domain.NewJobCommander(jobRepo, serviceRepo)
	metricEntryCmd := domain.NewMetricEntryCommander(metricEntryRepo, serviceRepo, metricTypeRepo)
	metricTypeCmd := domain.NewMetricTypeCommander(metricTypeRepo, metricEntryRepo)
	auditEntryCmd := domain.NewAuditEntryCommander(auditEntryRepo)
	agentCmd := domain.NewAgentCommander(agentRepo, serviceRepo)

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(agentTypeRepo)
	serviceTypeHandler := api.NewServiceTypeHandler(serviceTypeRepo)
	providerHandler := api.NewProviderHandler(providerRepo, providerCmd)
	agentHandler := api.NewAgentHandler(agentRepo, agentCmd)
	serviceGroupHandler := api.NewServiceGroupHandler(serviceGroupRepo, serviceGroupCmd)
	serviceHandler := api.NewServiceHandler(serviceRepo, serviceCmd)
	jobHandler := api.NewJobHandler(jobRepo, jobCmd)
	metricTypeHandler := api.NewMetricTypeHandler(metricTypeRepo, metricTypeCmd)
	metricEntryHandler := api.NewMetricEntryHandler(metricEntryRepo, metricEntryCmd)
	auditEntryHandler := api.NewAuditEntryHandler(auditEntryRepo, auditEntryCmd)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)
	// TODO refactor with global auth
	agentAuthMiddleware := api.AgentAuthMiddleware(agentRepo)

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
	go JobMainenanceTask(&cfg.JobConfig, jobRepo)

	// Setup background worker to mark inactive agents as disconnected
	go DisconnectUnhealthyAgentsTask(&cfg.AgentConfig, agentRepo)

	// Start server
	log.Printf("Server starting on port %d...", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func DisconnectUnhealthyAgentsTask(cfg *config.AgentConfig, agentRepo domain.AgentRepository) {
	ticker := time.NewTicker(cfg.HealthTimeout)
	for range ticker.C {
		ctx := context.Background()
		disconnectedCount, err := agentRepo.MarkInactiveAgentsAsDisconnected(ctx, cfg.HealthTimeout)
		if err != nil {
			log.Printf("Error marking inactive agents as disconnected: %v", err)
		} else if disconnectedCount > 0 {
			log.Printf("Marked %d inactive agents as disconnected", disconnectedCount)
		}
	}
}

func JobMainenanceTask(cfg *config.JobConfig, jobRepo domain.JobRepository) {
	ticker := time.NewTicker(cfg.Maintenance)
	for range ticker.C {
		ctx := context.Background()

		// Release jobs that have been processing for too long
		releasedCount, _ := jobRepo.ReleaseStuckJobs(ctx, cfg.Timeout)
		if releasedCount > 0 {
			log.Printf("Released %d stuck jobs", releasedCount)
		}

		// Delete completed/failed old jobs
		deletedCount, _ := jobRepo.DeleteOldCompletedJobs(ctx, cfg.Retention)
		if deletedCount > 0 {
			log.Printf("Deleted %d old completed jobs", deletedCount)
		}
	}
}
