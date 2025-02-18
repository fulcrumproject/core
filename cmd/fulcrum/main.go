package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"fulcrumproject.org/core/internal/api"
	"fulcrumproject.org/core/internal/database"
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

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(agentTypeRepo)
	serviceTypeHandler := api.NewServiceTypeHandler(serviceTypeRepo)
	providerHandler := api.NewProviderHandler(providerRepo, agentRepo)
	agentHandler := api.NewAgentHandler(agentRepo)
	serviceGroupHandler := api.NewServiceGroupHandler(serviceGroupRepo, serviceRepo)
	serviceHandler := api.NewServiceHandler(serviceRepo)
	metricTypeHandler := api.NewMetricTypeHandler(metricTypeRepo)
	metricEntryHandler := api.NewMetricEntryHandler(metricEntryRepo)

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
	})

	// Start server
	log.Println("Server starting on port 3000...")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
