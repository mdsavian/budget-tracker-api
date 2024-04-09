package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// create an env var for this
const jwtSecret = "test9999"

const ErrMethodNotAllowed = "method not allowed"

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

	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))

	router.HandleFunc("/user", makeHTTPHandleFunc(s.handleCreateUser))
	router.HandleFunc("/user/{id}", makeHTTPHandleFunc(s.handleUser))

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleAccountByID), s.store))

	log.Println("Server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, req)
}

// User
func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf(ErrMethodNotAllowed)
	}

	createNewUserInput := CreateNewUserInput{}

	if err := json.NewDecoder(r.Body).Decode(&createNewUserInput); err != nil {
		return err
	}

	user, err := NewUser(createNewUserInput)
	if err != nil {
		return err
	}

	if err := s.store.CreateUser(user); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, user)
}

func (s *APIServer) handleUser(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "DELETE":
		return s.handleDeleteUser(w, r)
	}

	return fmt.Errorf(ErrMethodNotAllowed)
}

func (s *APIServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) error {
	uUserID, err := getAndParseIDFromRequest(r)
	if err != nil {
		return err
	}

	if _, err := s.store.GetUserByID(uUserID); err != nil {
		return err
	}

	if err := s.store.DeleteUser(uUserID); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, "User deleted successfully")
}

// Account
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
	switch r.Method {
	case "GET":
		return s.handleGetAccountByID(w, r)
	case "DELETE":
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	uAccountID, err := getAndParseIDFromRequest(r)
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

	account := NewAccount(createNewAccountInput.Name, createNewAccountInput.AccountType)

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	tokenString, err := createJWT(account)
	if err != nil {
		return err
	}
	fmt.Println("token string: ", tokenString)
	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	uAccountID, err := getAndParseIDFromRequest(r)
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

func getAndParseIDFromRequest(r *http.Request) (uuid.UUID, error) {
	id := mux.Vars(r)["id"]
	uAccountId, err := uuid.Parse(id)
	if err != nil {
		return uAccountId, fmt.Errorf("error parsing id from request")
	}

	return uAccountId, nil
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtSecret), nil
	})

	return token, err
}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "Permission denied"})
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
	claims := JWTClaims{
		account.ID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
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
