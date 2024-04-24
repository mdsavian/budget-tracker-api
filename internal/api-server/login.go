package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type CredentialsInput struct {
	Email    string
	Password string
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	loginInput := CredentialsInput{}

	if err := json.NewDecoder(r.Body).Decode(&loginInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := s.store.GetUserByEmail(loginInput.Email)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	if !user.ValidPassword(loginInput.Password) {
		respondWithError(w, http.StatusUnauthorized, "password invalid")
		return
	}

	newSession := &types.Session{
		ID:        uuid.New(),
		UserId:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 2),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.CreateSession(newSession); err != nil {
		respondWithError(w, http.StatusInternalServerError, "fail creating session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSession.ID.String(),
		Expires: newSession.ExpiresAt,
	})

	respondWithJSON(w, http.StatusOK, "Login sucessfully")
}
