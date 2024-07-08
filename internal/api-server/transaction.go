package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

func (s *APIServer) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) handleCreateIncome(w http.ResponseWriter, r *http.Request) {
	type CreateIncomeInput struct {
		Amount      float64   `json:"amount"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
		CategoryId  uuid.UUID `json:"category_id"`
		AccountID   uuid.UUID `json:"account_id"`
	}

	incomeInput := CreateIncomeInput{}
	if err := json.NewDecoder(r.Body).Decode(&incomeInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	incomeDate, err := time.Parse("2006-01-02", incomeInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	incomeTransaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		TransactionType: types.TransactionTypeCredit,
		Amount:          incomeInput.Amount,
		Date:            incomeDate,
		Description:     incomeInput.Description,
		CategoryID:      incomeInput.CategoryId,
		AccountID:       incomeInput.AccountID,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.store.CreateTransaction(incomeTransaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	respondWithJSON(w, http.StatusOK, incomeTransaction)
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

	type DashboardInfo struct {
		Transactions    []*types.TransactionView `json:"transactions"`
		TotalCredit     float64                  `json:"totalCredit"`
		TotalDebit      float64                  `json:"totalDebit"`
		TotalCreditCard float64                  `json:"totalCreditCard"`
		CategoryTotals  []CategoryTotal          `json:"categoryTotals"`
		Accounts        []*types.Account         `json:"accounts"`
	}

	now := time.Now().AddDate(0, -3, 0)
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

	dashboardInfo := DashboardInfo{
		Transactions:    transactions,
		TotalCredit:     totalCredit,
		TotalDebit:      totalDebit,
		TotalCreditCard: totalCreditCard,
		CategoryTotals:  categoryTotals,
		Accounts:        accounts,
	}

	respondWithJSON(w, http.StatusOK, dashboardInfo)
}
