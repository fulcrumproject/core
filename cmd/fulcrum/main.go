package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"fulcrumproject.org/core/internal/api"
	"fulcrumproject.org/core/internal/database"
	"fulcrumproject.org/core/internal/service"
)

func main() {
	// Initialize database
	dbConfig := database.NewConfigFromEnv()
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Seed with basic data if needed
	if err := database.Seed(db); err != nil {
		log.Fatalf("Failed to seed the database: %v", err)
	}

	// Initialize repositories
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

	// Initialize services
	serviceOps := service.NewServiceOperationService(serviceRepo, jobRepo)

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(agentTypeRepo)
	serviceTypeHandler := api.NewServiceTypeHandler(serviceTypeRepo)
	providerHandler := api.NewProviderHandler(providerRepo, agentRepo)
	agentHandler := api.NewAgentHandler(agentRepo)
	serviceGroupHandler := api.NewServiceGroupHandler(serviceGroupRepo, serviceRepo)

	// Use ServiceOperationService for service operations that require job creation
	serviceHandler := api.NewServiceHandler(serviceOps)

	// Job handler for agent communication
	jobHandler := api.NewJobHandler(jobRepo, agentRepo)

	metricTypeHandler := api.NewMetricTypeHandler(metricTypeRepo)
	metricEntryHandler := api.NewMetricEntryHandler(metricEntryRepo)
	auditEntryHandler := api.NewAuditEntryHandler(auditEntryRepo)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/providers", providerHandler.Routes())
		r.Mount("/agent-types", agentTypeHandler.Routes())
		r.Mount("/service-types", serviceTypeHandler.Routes())
		r.Mount("/agents", agentHandler.Routes())
		r.Mount("/service-groups", serviceGroupHandler.Routes())
		r.Mount("/services", serviceHandler.Routes())
		r.Mount("/metric-types", metricTypeHandler.Routes())
		r.Mount("/metric-entries", metricEntryHandler.Routes())
		r.Mount("/audit-entries", auditEntryHandler.Routes())
		r.Mount("/jobs", jobHandler.Routes())
	})

	// Setup background job maintenance worker
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			ctx := context.Background()

			// Release jobs that have been processing for more than 30 minutes
			releasedCount, _ := jobRepo.ReleaseStuckJobs(ctx, 30)
			if releasedCount > 0 {
				log.Printf("Released %d stuck jobs", releasedCount)
			}

			// Delete completed/failed jobs older than 7 days
			deletedCount, _ := jobRepo.DeleteOldCompletedJobs(ctx, 7)
			if deletedCount > 0 {
				log.Printf("Deleted %d old completed jobs", deletedCount)
			}
		}
	}()

	// Setup background worker to mark inactive agents as disconnected
	go func() {
		inactiveTime := 5 * time.Minute
		ticker := time.NewTicker(inactiveTime)
		for range ticker.C {
			ctx := context.Background()
			disconnectedCount, err := agentRepo.MarkInactiveAgentsAsDisconnected(ctx, inactiveTime)
			if err != nil {
				log.Printf("Error marking inactive agents as disconnected: %v", err)
			} else if disconnectedCount > 0 {
				log.Printf("Marked %d inactive agents as disconnected", disconnectedCount)
			}
		}
	}()

	// Start server
	log.Println("Server starting on port 3000...")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
