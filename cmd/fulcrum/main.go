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

	// Initialize repositories
	providerRepo := database.NewProviderRepository(db)

	// Initialize handlers
	providerHandler := api.NewProviderHandler(providerRepo)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Fulcrum Core API"))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Provider routes
		r.Mount("/providers", providerHandler.Routes())
	})

	// Start server
	log.Println("Server starting on port 3000...")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
