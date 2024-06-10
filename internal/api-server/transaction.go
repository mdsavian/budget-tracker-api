package apiserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type CreateTransactionInput struct {
	AccountID       uuid.UUID             `json:"account_id"`
	CreditCardID    uuid.UUID             `json:"credit_card_id"`
	CategoryID      uuid.UUID             `json:"category_id"`
	TransactionType types.TransactionType `json:"transaction_type"`
	Date            string                `json:"date"`
	Description     string                `json:"description"`
	Amount          float64               `json:"amount"`
	Paid            bool                  `json:"paid"`
	CostOfLiving    bool                  `json:"cost_of_living"`
}

func (s *APIServer) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	transactionInput := CreateTransactionInput{}
	if err := json.NewDecoder(r.Body).Decode(&transactionInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Println(transactionInput.Date)
	transactionDate, err := time.Parse("2006-01-02", transactionInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	transaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		AccountID:       transactionInput.AccountID,
		CategoryID:      transactionInput.CategoryID,
		TransactionType: transactionInput.TransactionType,
		Date:            transactionDate,
		Description:     transactionInput.Description,
		Amount:          transactionInput.Amount,
		Paid:            transactionInput.Paid,
		CostOfLiving:    transactionInput.CostOfLiving,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if transactionInput.CreditCardID != uuid.Nil {
		transaction.CreditCardID = &transactionInput.CreditCardID
	}

	if err := s.store.CreateTransaction(transaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	respondWithJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	transactions, err := s.store.GetTransaction()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, transactions)
}
