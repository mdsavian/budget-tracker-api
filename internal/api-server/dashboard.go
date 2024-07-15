package apiserver

import (
	"net/http"
	"time"

	"github.com/mdsavian/budget-tracker-api/internal/types"
)

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
