package apiserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type CreateNewCreditCardInput struct {
	Name       string `json:"name"`
	ClosingDay int8   `json:"closingDay"`
}

func (s *APIServer) handleCreateCreditCard(w http.ResponseWriter, r *http.Request) {
	cardInput := CreateNewCreditCardInput{}

	if err := json.NewDecoder(r.Body).Decode(&cardInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	creditCard := &types.CreditCard{
		ID:         uuid.Must(uuid.NewV7()),
		Name:       cardInput.Name,
		ClosingDay: cardInput.ClosingDay,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := s.store.CreateCreditCard(creditCard); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	respondWithJSON(w, http.StatusOK, creditCard)
}

func (s *APIServer) handleGetCreditCard(w http.ResponseWriter, r *http.Request) {
	nameInputFilter := r.URL.Query().Get("name")
	if nameInputFilter != "" {
		creditCard, err := s.store.GetCreditCardByName(nameInputFilter)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, creditCard)
		return
	}

	cards, err := s.store.GetCreditCard()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, cards)
}

func (s *APIServer) handleGetCreditCardById(w http.ResponseWriter, r *http.Request) {
	id, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	creditCard, err := s.store.GetCreditCardByID(id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, creditCard)
}

func (s *APIServer) handleArchiveCreditCard(w http.ResponseWriter, r *http.Request) {
	id, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	if _, err := s.store.GetCreditCardByID(id); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = s.store.ArchiveCreditCard(id)
	log.Println(err)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, "CreditCard archived successfully")
}
