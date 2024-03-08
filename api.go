package main

import (
	"net/http"
)

type APIServer struct {
	listenAddr string
}

func NewApiServer(listenAddr string) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
	}
}

func (s *APIServer) Start() {

}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, req *http.Request) error {

	return nil
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, req *http.Request) error {

	return nil
}
func (s *APIServer) handleGetAccount(w http.ResponseWriter, req *http.Request) error {

	return nil
}
