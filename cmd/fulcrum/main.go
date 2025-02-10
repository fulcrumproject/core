package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Item represents a simple resource
type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
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
		r.Get("/", listItems)
		r.Post("/", createItem)
		r.Get("/{id}", getItem)
	})

	// Start server
	println("Server starting on port 3000...")
	http.ListenAndServe(":3000", r)
}

func listItems(w http.ResponseWriter, r *http.Request) {
	items := []Item{
		{ID: "1", Name: "Item 1"},
		{ID: "2", Name: "Item 2"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func createItem(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// In a real application, you would save the item to a database
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func getItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// In a real application, you would fetch the item from a database
	item := Item{ID: id, Name: "Sample Item"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}
