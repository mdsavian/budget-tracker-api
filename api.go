package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

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
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleAccountByID)))
	log.Println("Server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {

	switch r.Method {
	case "GET":
		return s.handleGetAccounts(w, r)
	case "POST":
		return s.handleCreateAccount(w, r)
	}

	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleAccountByID(w http.ResponseWriter, r *http.Request) error {

	if r.Method == "GET" {
		return s.handleGetAccountByID(w, r)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {

	uAccountID, err := getAndParseAccountID(r)
	if err != nil {
		return err

	}

	account, err := s.store.GetAccountByID(uAccountID)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createNewAccountInput := CreateNewAccountInput{}

	if err := json.NewDecoder(r.Body).Decode(&createNewAccountInput); err != nil {
		return err
	}

	account := NewAccount(createNewAccountInput.Name, AccountType(createNewAccountInput.AccountType))

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	uAccountID, err := getAndParseAccountID(r)
	if err != nil {
		return err
	}

	if _, err := s.store.GetAccountByID(uAccountID); err != nil {
		return err
	}

	if err := s.store.DeleteAccount(uAccountID); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, "Account deleted successfully")
}

func (s *APIServer) handleGetAccounts(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func getAndParseAccountID(r *http.Request) (uuid.UUID, error) {
	id := mux.Vars(r)["id"]
	uAccountId, err := uuid.Parse(id)
	if err != nil {
		return uAccountId, fmt.Errorf("error parsing account id")
	}

	return uAccountId, nil

}

func validateJWT(tokenString string) (*jwt.Token, error) {
	// TODO create an env var for this
	const jwtSecret = "test9999"

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtSecret), nil
	})

	return token, err
}

func withJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Calling JWT auth middleware")
		handlerFunc(w, r)
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type ApiError struct {
	Error string `json:"error"`
}

/*
* Declared this type and the makeHttpHandleFunc function because we need to deal with the error
* the core func HandlerFunc doesn't support the error so we need to convert it
* to be able to use inside the mux
 */
type apiFunc func(http.ResponseWriter, *http.Request) error

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}
