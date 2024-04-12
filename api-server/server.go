package apiserver

import (
	"log"
	"net/http"

	"github.com/mdsavian/budget-tracker-api/types"
)

type APIServer struct {
	listenAddr string
	store      types.Storage
}

func NewServer(listenAddr string, store types.Storage) *APIServer {
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

	//mux.HandleFunc("POST /login", s.handleLogin)

	log.Println("Server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, mux)
}
