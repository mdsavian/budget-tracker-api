package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type CreateNewCategoryInput struct {
	Description string `json:"description"`
}

func (s *APIServer) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	categoryInput := CreateNewCategoryInput{}

	if err := json.NewDecoder(r.Body).Decode(&categoryInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	category := &types.Category{
		ID:          uuid.Must(uuid.NewV7()),
		Description: categoryInput.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.store.CreateCategory(category); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	respondWithJSON(w, http.StatusOK, category)
}

func (s *APIServer) handleGetCategory(w http.ResponseWriter, r *http.Request) {
	categories, err := s.store.GetCategory()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, categories)
}

func (s *APIServer) handleGetCategoryByDescription(w http.ResponseWriter, r *http.Request) {
	description := r.PathValue("description")

	category, err := s.store.GetCategoryByDescription(description)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, category)
}

func (s *APIServer) handleArchiveCategory(w http.ResponseWriter, r *http.Request) {
	id, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	err = s.store.ArchiveCategory(id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, "Category archived successfully")
}
