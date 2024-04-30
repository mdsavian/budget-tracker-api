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

const COOKIE_NAME = "session_token"

type apiFunc func(http.ResponseWriter, *http.Request)

func (s *APIServer) validateSession(f apiFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(COOKIE_NAME)
		if err != nil {
			if err == http.ErrNoCookie {
				respondWithError(w, http.StatusUnauthorized, "cookie invalid")
				return
			}
			respondWithError(w, http.StatusBadRequest, "cookie invalid")
			return
		}

		sessionToken := cookie.Value
		sessionID, err := uuid.Parse(sessionToken)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error parsing session ID")
			return
		}

		session, err := s.store.GetSessionByID(sessionID)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		if session.IsExpired() {
			respondWithError(w, http.StatusUnauthorized, "session expired")
			return
		}
		f(w, r)

	})

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
		Name:    COOKIE_NAME,
		Value:   newSession.ID.String(),
		Expires: newSession.ExpiresAt,
	})

	respondWithJSON(w, http.StatusOK, "Login sucessfully")
}

func (s *APIServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		if err == http.ErrNoCookie {
			respondWithError(w, http.StatusUnauthorized, "cookie invalid")
			return
		}
		respondWithError(w, http.StatusBadRequest, "cookie invalid")
		return
	}

	sessionToken := cookie.Value
	sessionID, err := uuid.Parse(sessionToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error parsing session ID")
		return
	}

	if err := s.store.UpdateSession(sessionID, time.Now()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   "",
		Expires: time.Now()})

	respondWithJSON(w, http.StatusOK, "logout successfully")
}
