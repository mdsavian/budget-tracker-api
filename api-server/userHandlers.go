package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/types"
	"golang.org/x/crypto/bcrypt"
)

const ErrMethodNotAllowed = "Method not allowed"

func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	createNewUserInput := types.CreateNewUserInput{}

	if err := json.NewDecoder(r.Body).Decode(&createNewUserInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := newUser(createNewUserInput)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.CreateUser(user); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (s *APIServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	uUserID, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := s.store.GetUserByID(uUserID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.DeleteUser(uUserID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, "User deleted successfully")
}

func newUser(input types.CreateNewUserInput) (*types.User, error) {
	encriptedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &types.User{
		ID:                uuid.Must(uuid.NewV7()),
		Name:              input.Name,
		EncryptedPassword: string(encriptedPassword),
		Email:             input.Email,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}, nil
}
