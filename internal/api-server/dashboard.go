package apiserver

import (
	"net/http"
	"time"

	"github.com/mdsavian/budget-tracker-api/internal/types"
)

func (s *APIServer) handleGetDashboardInfo(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	startDate := queryValues.Get("startDate")
	endDate := queryValues.Get("endDate")

	if startDate == "" || endDate == "" {
		respondWithError(w, http.StatusBadRequest, "startDate and endDate are required")
		return
	}

	startDateParsed, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "startDate is not a valid date")
		return
	}
	endDateParsed, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "endDate is not a valid date")
		return
	}

	type CategoryTotal struct {
		Name  string  `json:"name"`
		Total float64 `json:"total"`
	}

	type DashboardInfo struct {
		Transactions     []*types.TransactionView `json:"transactions"`
		TotalCredit      float64                  `json:"totalCredit"`
		TotalDebit       float64                  `json:"totalDebit"`
		TotalDebitUnpaid float64                  `json:"totalDebitUnpaid"`
		TotalCreditCard  float64                  `json:"totalCreditCard"`
		CategoryTotals   []CategoryTotal          `json:"categoryTotals"`
		Accounts         []*types.Account         `json:"accounts"`
	}

	transactions, err := s.store.GetTransactionsWithRecurringByDate(startDateParsed, endDateParsed)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var totalCredit float64 = 0
	var totalDebit float64 = 0
	var totalDebitUnpaid float64 = 0

	var totalCreditCard float64 = 0
	var categoryMap = map[string]float64{}

	for _, transaction := range transactions {
		if transaction.TransactionType == types.TransactionTypeCredit {
			totalCredit += transaction.Amount
		} else if transaction.TransactionType == types.TransactionTypeDebit {
			totalDebit += transaction.Amount
			if !transaction.Fulfilled {
				totalDebitUnpaid += transaction.Amount
			}
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
		Transactions:     transactions,
		TotalCredit:      totalCredit,
		TotalDebit:       totalDebit,
		TotalCreditCard:  totalCreditCard,
		TotalDebitUnpaid: totalDebitUnpaid,
		CategoryTotals:   categoryTotals,
		Accounts:         accounts,
	}

	respondWithJSON(w, http.StatusOK, dashboardInfo)
}

func (s *APIServer) handleGetTransactionsByDate(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	startDate := queryValues.Get("startDate")
	endDate := queryValues.Get("endDate")

	if startDate == "" || endDate == "" {
		respondWithError(w, http.StatusBadRequest, "startDate and endDate are required")
		return
	}

	startDateParsed, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "startDate is not a valid date")
		return
	}
	endDateParsed, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "endDate is not a valid date")
		return
	}

	type TransactionInfo struct {
		Transactions []*types.TransactionView `json:"transactions"`
	}

	transactions, err := s.store.GetTransactionsWithRecurringByDate(startDateParsed, endDateParsed)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	transactionInfo := TransactionInfo{
		Transactions: transactions,
	}

	respondWithJSON(w, http.StatusOK, transactionInfo)
}
