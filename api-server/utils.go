package apiserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

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

func getAndParseIDFromRequest(r *http.Request) (uuid.UUID, error) {
	id := r.PathValue("id")
	uAccountId, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error parsing id from request")
	}

	return uAccountId, nil
}
