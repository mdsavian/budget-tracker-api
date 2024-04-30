package apiserver

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type Storage interface {
	CreateAccount(*types.Account) error
	DeleteAccount(uuid.UUID) error
	GetAccountByID(uuid.UUID) (*types.Account, error)
	GetAccounts() ([]*types.Account, error)
	CreateUser(*types.User) error
	DeleteUser(uuid.UUID) error
	GetUserByID(uuid.UUID) (*types.User, error)
	GetUserByEmail(string) (*types.User, error)
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

	mux.HandleFunc("POST /user", s.validateSession(s.handleCreateUser))
	mux.HandleFunc("DELETE /user/{id}", s.handleDeleteUser)

	mux.HandleFunc("POST /account", s.handleCreateAccount)
	mux.HandleFunc("GET /account", s.handleGetAccounts)
	mux.HandleFunc("GET /account/{id}", s.handleGetAccountByID)
	mux.HandleFunc("DELETE /account/{id}", s.handleDeleteAccount)

	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)

	log.Println("Server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, mux)
}
