package main

import (
	"log"
	"net/http"

	"fulcrumproject.org/core/internal/api"
	"fulcrumproject.org/core/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize database
	dbConfig := database.NewConfigFromEnv()
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repository
	itemRepo := database.NewItemRepository(db)

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the API"))
	})

	// Items endpoints
	r.Route("/items", func(r chi.Router) {
		r.Get("/", api.HandleListItems(itemRepo))
		r.Post("/", api.HandleCreateItem(itemRepo))
		r.Get("/{id}", api.HandleGetItem(itemRepo))
		r.Put("/{id}", api.HandleUpdateItem(itemRepo))
		r.Delete("/{id}", api.HandleDeleteItem(itemRepo))
	})

	log.Println("Server starting on port 3000...")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
