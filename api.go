package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const ErrMethodNotAllowed = "Method not allowed"

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewApiServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /user", s.handleCreateUser)
	mux.HandleFunc("DELETE /user/{id}", s.handleDeleteUser)

	mux.HandleFunc("POST /account", s.handleCreateAccount)
	mux.HandleFunc("GET /account", s.handleGetAccounts)
	mux.HandleFunc("GET /account/{id}", s.handleGetAccountByID)
	mux.HandleFunc("DELETE /account/{id}", s.handleDeleteAccount)

	mux.HandleFunc("POST /login", s.handleLogin)

	log.Println("Server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, mux)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err := s.store.GetUserByEmail(req.Email)
	// TODO check if the error is user not found otherwise return only the error
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	respondWithJSON(w, http.StatusOK, req)
}

// User
func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	createNewUserInput := CreateNewUserInput{}

	if err := json.NewDecoder(r.Body).Decode(&createNewUserInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := NewUser(createNewUserInput)
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

// Account
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
	createNewAccountInput := CreateNewAccountInput{}

	if err := json.NewDecoder(r.Body).Decode(&createNewAccountInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	account := NewAccount(createNewAccountInput.Name, createNewAccountInput.AccountType)

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

func getAndParseIDFromRequest(r *http.Request) (uuid.UUID, error) {
	id := r.PathValue("id")
	uAccountId, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error parsing id from request")
	}

	return uAccountId, nil
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtSecret), nil
	})

	return token, err
}

func permissionDenied(w http.ResponseWriter) {
	respondWithError(w, http.StatusUnauthorized, "Permission deined")
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Calling JWT auth middleware")

		// TODO change the name of the header -- might convert to cookies
		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)
		if err != nil {
			permissionDenied(w)
			return
		}
		if !token.Valid {
			permissionDenied(w)
			return
		}

		accountId, err := getAndParseIDFromRequest(r)
		if err != nil {
			permissionDenied(w)
			return
		}
		account, err := s.GetAccountByID(accountId)
		if err != nil {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		claimAccountID, ok := claims["accountID"].(string)
		if !ok {
			permissionDenied(w)
			return
		}
		uClaimAccountID, err := uuid.Parse(claimAccountID)
		if err != nil {
			permissionDenied(w)
			return
		}

		if account.ID != uClaimAccountID {
			permissionDenied(w)
			return
		}

		handlerFunc(w, r)
	}
}

type JWTClaims struct {
	accountID uuid.UUID
	jwt.RegisteredClaims
}

func createJWT(account *Account) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	claims := JWTClaims{
		account.ID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	if statusCode > 499 {
		log.Println("Respond with 5XX error:", message)
	}

	respondWithJSON(w, statusCode, errorResponse{
		Error: message,
	})
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", payload)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)

}
