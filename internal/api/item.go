package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
)

type createItemRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Properties   map[string]interface{} `json:"properties"`
	JsonProperty domain.JsonProperty    `json:"jsonProperty"`
}

type updateItemRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Properties   map[string]interface{} `json:"properties"`
	JsonProperty domain.JsonProperty    `json:"jsonProperty"`
}

type itemResponse struct {
	ID           uint                   `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Properties   map[string]interface{} `json:"properties"`
	JsonProperty domain.JsonProperty    `json:"jsonProperty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

func newItemResponse(item *domain.Item) *itemResponse {
	return &itemResponse{
		ID:           item.ID,
		Name:         item.Name,
		Description:  item.Description,
		Properties:   item.Properties,
		JsonProperty: item.JsonProperty,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func HandleListItems(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := repo.List()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to list items")
			return
		}

		var responses []*itemResponse
		for _, item := range items {
			responses = append(responses, newItemResponse(&item))
		}
		respondWithJSON(w, http.StatusOK, responses)
	}
}

func HandleCreateItem(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		item := &domain.Item{
			Name:         req.Name,
			Description:  req.Description,
			Properties:   req.Properties,
			JsonProperty: req.JsonProperty,
		}

		if err := repo.Create(item); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create item")
			return
		}

		respondWithJSON(w, http.StatusCreated, newItemResponse(item))
	}
}

func HandleGetItem(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid item ID")
			return
		}

		item, err := repo.GetByID(uint(id))
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to get item")
			return
		}
		if item == nil {
			respondWithError(w, http.StatusNotFound, "Item not found")
			return
		}

		respondWithJSON(w, http.StatusOK, newItemResponse(item))
	}
}

func HandleUpdateItem(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid item ID")
			return
		}

		var req updateItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		existingItem, err := repo.GetByID(uint(id))
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to get item")
			return
		}
		if existingItem == nil {
			respondWithError(w, http.StatusNotFound, "Item not found")
			return
		}

		existingItem.Name = req.Name
		existingItem.Description = req.Description
		existingItem.Properties = req.Properties
		existingItem.JsonProperty = req.JsonProperty

		if err := repo.Update(existingItem); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update item")
			return
		}

		respondWithJSON(w, http.StatusOK, newItemResponse(existingItem))
	}
}

func HandleDeleteItem(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid item ID")
			return
		}

		if err := repo.Delete(uint(id)); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete item")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
