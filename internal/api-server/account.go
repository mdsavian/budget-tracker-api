package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) {
	uAccountID, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	account, err := s.store.GetAccountByID(uAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	createNewAccountInput := types.CreateNewAccountInput{}
	if err := json.NewDecoder(r.Body).Decode(&createNewAccountInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	account := newAccount(createNewAccountInput)
	if err := s.store.CreateAccount(account); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	uAccountID, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := s.store.GetAccountByID(uAccountID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.DeleteAccount(uAccountID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, "Account deleted successfully")
}

func (s *APIServer) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, accounts)
}

func newAccount(input types.CreateNewAccountInput) *types.Account {
	return &types.Account{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        input.Name,
		Balance:     0,
		AccountType: input.AccountType,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}
