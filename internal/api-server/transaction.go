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

func (s *APIServer) handleGetDashboardInfo(w http.ResponseWriter, r *http.Request) {
	type CategoryTotal struct {
		Name  string  `json:"name"`
		Total float64 `json:"total"`
	}

	type ObjetoRetorno struct {
		Transactions    []*types.TransactionView `json:"transactions"`
		TotalCredit     float64                  `json:"totalCredit"`
		TotalDebit      float64                  `json:"totalDebit"`
		TotalCreditCard float64                  `json:"totalCreditCard"`
		CategoryTotals  []CategoryTotal          `json:"categoryTotals"`
		Accounts        []*types.Account         `json:"accounts"`
	}

	now := time.Now().AddDate(0, -1, 0)
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastDay := firstDay.AddDate(0, 1, -1)

	transactions, err := s.store.GetTransactionsByDate(firstDay, lastDay)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var totalCredit float64 = 0
	var totalDebit float64 = 0
	var totalCreditCard float64 = 0
	var categoryMap = map[string]float64{}

	for _, transaction := range transactions {
		log.Println(transaction.TransactionType)
		if transaction.TransactionType == types.TransactionTypeCredit {
			totalCredit += transaction.Amount
		} else if transaction.TransactionType == types.TransactionTypeDebit {
			totalDebit += transaction.Amount
			categoryMap[transaction.Category] += transaction.Amount
		}

		if transaction.CreditCardID != nil {
			totalCreditCard += transaction.Amount
		}
	}

	accounts, err := s.store.GetAccounts()
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	var categoryTotals []CategoryTotal
	for category, total := range categoryMap {
		categoryTotals = append(categoryTotals, CategoryTotal{Name: category, Total: total})
	}

	xx := ObjetoRetorno{
		Transactions:    transactions,
		TotalCredit:     totalCredit,
		TotalDebit:      totalDebit,
		TotalCreditCard: totalCreditCard,
		CategoryTotals:  categoryTotals,
		Accounts:        accounts,
	}

	log.Println(categoryTotals, totalCredit, totalDebit, totalCreditCard)

	respondWithJSON(w, http.StatusOK, xx)

	// group by credit card
	// saldo atual conta pf e conta pj

}
