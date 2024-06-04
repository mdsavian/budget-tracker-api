package apiserver

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type Storage interface {
	// CreditCard
	CreateCreditCard(*types.CreditCard) error
	GetCreditCard() ([]*types.CreditCard, error)
	GetCreditCardByName(string) (*types.CreditCard, error)
	GetCreditCardByID(uuid.UUID) (*types.CreditCard, error)
	ArchiveCreditCard(uuid.UUID) error

	// Category
	CreateCategory(*types.Category) error
	GetCategory() ([]*types.Category, error)
	GetCategoryByDescription(string) (*types.Category, error)
	GetCategoryByID(uuid.UUID) (*types.Category, error)
	ArchiveCategory(uuid.UUID) error

	// Account
	CreateAccount(*types.Account) error
	DeleteAccount(uuid.UUID) error
	GetAccountByID(uuid.UUID) (*types.Account, error)
	GetAccounts() ([]*types.Account, error)

	// User
	CreateUser(*types.User) error
	DeleteUser(uuid.UUID) error
	GetUserByID(uuid.UUID) (*types.User, error)
	GetUserByEmail(string) (*types.User, error)

	// Session
	CreateSession(*types.Session) error
	DeleteSession(uuid.UUID) error
	UpdateSession(uuid.UUID, time.Time) error
	GetSessionByID(uuid.UUID) (*types.Session, error)
}

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /creditcard", s.validateSession(s.handleCreateCreditCard))
	mux.HandleFunc("GET /creditcard", s.validateSession(s.handleGetCreditCard))
	mux.HandleFunc("GET /creditcard/{id}", s.validateSession(s.handleGetCreditCardById))
	mux.HandleFunc("PUT /creditcard/archive/{id}", s.validateSession(s.handleArchiveCreditCard))

	mux.HandleFunc("POST /category", s.validateSession(s.handleCreateCategory))
	mux.HandleFunc("GET /category", s.validateSession(s.handleGetCategory))
	mux.HandleFunc("GET /category/{description}", s.validateSession(s.handleGetCategoryByDescription))
	mux.HandleFunc("PUT /category/archive/{id}", s.validateSession(s.handleArchiveCategory))

	mux.HandleFunc("POST /user", s.validateSession(s.handleCreateUser))
	mux.HandleFunc("DELETE /user/{id}", s.validateSession(s.handleDeleteUser))

	mux.HandleFunc("POST /account", s.validateSession(s.handleCreateAccount))
	mux.HandleFunc("GET /account", s.validateSession(s.handleGetAccounts))
	mux.HandleFunc("GET /account/{id}", s.validateSession(s.handleGetAccountByID))
	mux.HandleFunc("DELETE /account/{id}", s.validateSession(s.handleDeleteAccount))

	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)

	log.Println("Server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, mux)
}
