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
		Transactions            []*types.TransactionView `json:"transactions"`
		TotalCredit             float64                  `json:"totalCredit"`
		TotalDebit              float64                  `json:"totalDebit"`
		TotalDebitUnpaid        float64                  `json:"totalDebitUnpaid"`
		TotalCreditUpcoming     float64                  `json:"totalCreditUpcoming"`
		TotalCreditCard         float64                  `json:"totalCreditCard"`
		TotalCreditCardUpcoming float64                  `json:"totalCreditCardUpcoming"`
		CategoryTotals          []CategoryTotal          `json:"categoryTotals"`
		Balance                 float64                  `json:"balance"`
		Accounts                []*types.Account         `json:"accounts"`
	}

	transactions, err := s.store.GetTransactionsWithRecurringByDate(startDateParsed, endDateParsed)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var totalCredit float64 = 0
	var totalDebit float64 = 0
	var totalDebitUnpaid float64 = 0
	var totalCreditUpcoming float64 = 0
	var totalCreditCardUpcoming float64 = 0
	var totalCreditCard float64 = 0
	var categoryMap = map[string]float64{}

	for _, transaction := range transactions {
		if transaction.TransactionType == types.TransactionTypeCredit {
			if !transaction.Fulfilled {
				totalCreditUpcoming += transaction.Amount
			} else {
				totalCredit += transaction.Amount
			}
		} else if transaction.TransactionType == types.TransactionTypeDebit {

			if !transaction.Fulfilled {
				totalDebitUnpaid += transaction.Amount
			} else {
				totalDebit += transaction.Amount
			}

			categoryMap[transaction.Category] += transaction.Amount
		}

		if transaction.CreditCardID != nil {
			if transaction.TransactionType == types.TransactionTypeDebit {
				totalCreditCard += transaction.Amount
			} else if transaction.TransactionType == types.TransactionTypeCredit {
				if transaction.Fulfilled {
					totalCreditCard -= transaction.Amount
				} else {
					totalCreditCardUpcoming += transaction.Amount
				}
			}
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

	balance := totalCredit + totalCreditUpcoming - (totalDebit + totalDebitUnpaid)

	dashboardInfo := DashboardInfo{
		Transactions:            transactions,
		TotalCredit:             totalCredit,
		TotalDebit:              totalDebit,
		TotalCreditCard:         totalCreditCard,
		TotalDebitUnpaid:        totalDebitUnpaid,
		TotalCreditUpcoming:     totalCreditUpcoming,
		TotalCreditCardUpcoming: totalCreditCardUpcoming,
		CategoryTotals:          categoryTotals,
		Balance:                 balance,
		Accounts:                accounts,
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
