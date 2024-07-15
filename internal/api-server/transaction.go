package apiserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type CreateCreditCardExpenseInput struct {
	CreditCardID uuid.UUID `json:"credit_card_id"`
	AccountID    uuid.UUID `json:"account_id"`
	CategoryId   uuid.UUID `json:"category_id"`
	Amount       float32   `json:"amount"`
	Date         string    `json:"date"`
	Description  string    `json:"description"`
	Installments int       `json:"installments"`
	Fixed        bool      `json:"fixed"`
}

func (s *APIServer) handleCreateCreditCardExpense(w http.ResponseWriter, r *http.Request) {
	expenseInput := CreateCreditCardExpenseInput{}
	if err := json.NewDecoder(r.Body).Decode(&expenseInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if expenseInput.CreditCardID == uuid.Nil {
		respondWithError(w, http.StatusBadRequest, "credit card is required")
		return
	}

	if expenseInput.Installments > 0 && expenseInput.Fixed {
		respondWithError(w, http.StatusBadRequest, "installments and fixed cannot be used together")
		return
	}

	creditCard, err := s.store.GetCreditCardByID(expenseInput.CreditCardID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	creditCardExpenseDate, err := time.Parse("2006-01-02", expenseInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	expenseDayToAdd := creditCard.DueDay - creditCardExpenseDate.Day()
	if creditCardExpenseDate.Day() < creditCard.ClosingDay {
		creditCardExpenseDate = creditCardExpenseDate.AddDate(0, 0, expenseDayToAdd)
	} else {
		creditCardExpenseDate = creditCardExpenseDate.AddDate(0, 1, expenseDayToAdd)
	}

	if expenseInput.Fixed {
		creditCardRecurringTransaction, err := s.createRecurringCreditCardExpense(expenseInput, creditCardExpenseDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, creditCardRecurringTransaction)
		return
	}

	if expenseInput.Installments > 0 {
		firstInstallmentTransaction, err := s.createCreditCardExpenseInstallments(expenseInput, creditCardExpenseDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, firstInstallmentTransaction)
		return
	}

	transaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		CategoryID:      expenseInput.CategoryId,
		AccountID:       expenseInput.AccountID,
		CreditCardID:    &expenseInput.CreditCardID,
		TransactionType: types.TransactionTypeDebit,
		Amount:          expenseInput.Amount,
		Date:            creditCardExpenseDate,
		Description:     expenseInput.Description,
		Fulfilled:       false,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.store.CreateTransaction(transaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) createCreditCardExpenseInstallments(expenseInput CreateCreditCardExpenseInput, creditCardExpenseDate time.Time) (*types.Transaction, error) {
	amountPerInstallment := expenseInput.Amount / float32(expenseInput.Installments)
	var firstInstallmentTransaction *types.Transaction

	for i := 0; i < expenseInput.Installments; i++ {
		installmentDate := creditCardExpenseDate.AddDate(0, i, 0)

		installmentTransaction := &types.Transaction{
			ID:              uuid.Must(uuid.NewV7()),
			CategoryID:      expenseInput.CategoryId,
			AccountID:       expenseInput.AccountID,
			CreditCardID:    &expenseInput.CreditCardID,
			TransactionType: types.TransactionTypeDebit,
			Amount:          amountPerInstallment,
			Date:            installmentDate,
			Description:     expenseInput.Description + " (" + strconv.Itoa(i+1) + "/" + strconv.Itoa(expenseInput.Installments) + ")",
			Fulfilled:       false,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}

		if err := s.store.CreateTransaction(installmentTransaction); err != nil {
			return nil, err
		}

		if i == 0 {
			firstInstallmentTransaction = installmentTransaction
		}
	}
	return firstInstallmentTransaction, nil
}

func (s *APIServer) createRecurringCreditCardExpense(creditCardExpenseInput CreateCreditCardExpenseInput, creditCardExpenseDate time.Time) (*types.Transaction, error) {
	recurringTransactionID := uuid.Must(uuid.NewV7())

	creditCardRecurringTransaction := &types.Transaction{
		ID:                     uuid.Must(uuid.NewV7()),
		CategoryID:             creditCardExpenseInput.CategoryId,
		AccountID:              creditCardExpenseInput.AccountID,
		CreditCardID:           &creditCardExpenseInput.CreditCardID,
		RecurringTransactionID: &recurringTransactionID,
		TransactionType:        types.TransactionTypeDebit,
		Amount:                 creditCardExpenseInput.Amount,
		Date:                   creditCardExpenseDate,
		Description:            creditCardExpenseInput.Description,
		Fulfilled:              false,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}

	err := s.store.CreateRecurringTransaction(&types.RecurringTransaction{
		ID:              recurringTransactionID,
		AccountID:       creditCardExpenseInput.AccountID,
		CategoryID:      creditCardExpenseInput.CategoryId,
		CreditCardID:    &creditCardExpenseInput.CreditCardID,
		TransactionType: types.TransactionTypeDebit,
		Day:             creditCardExpenseDate.Day(),
		Description:     creditCardExpenseInput.Description,
		Amount:          creditCardExpenseInput.Amount,
		Archived:        false,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	if err := s.store.CreateTransaction(creditCardRecurringTransaction); err != nil {
		return nil, err
	}

	return creditCardRecurringTransaction, nil
}

func (s *APIServer) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	type CreateExpenseInput struct {
		CategoryId  uuid.UUID `json:"category_id"`
		AccountID   uuid.UUID `json:"account_id"`
		Amount      float32   `json:"amount"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
		Fulfilled   bool      `json:"fulfilled"`
		Fixed       bool      `json:"fixed"`
	}
	expenseInput := CreateExpenseInput{}
	if err := json.NewDecoder(r.Body).Decode(&expenseInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	expenseDate, err := time.Parse("2006-01-02", expenseInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	expenseTransaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		TransactionType: types.TransactionTypeDebit,
		Amount:          expenseInput.Amount,
		Date:            expenseDate,
		Description:     expenseInput.Description,
		CategoryID:      expenseInput.CategoryId,
		AccountID:       expenseInput.AccountID,
		Fulfilled:       expenseInput.Fulfilled,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if expenseInput.Fixed {
		recurringTransactionID := uuid.Must(uuid.NewV7())

		err := s.store.CreateRecurringTransaction(&types.RecurringTransaction{
			ID:              recurringTransactionID,
			AccountID:       expenseInput.AccountID,
			CategoryID:      expenseInput.CategoryId,
			TransactionType: types.TransactionTypeDebit,
			Day:             expenseDate.Day(),
			Description:     expenseInput.Description,
			Amount:          expenseInput.Amount,
			Archived:        false,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		expenseTransaction.RecurringTransactionID = &recurringTransactionID
	}

	if err := s.store.CreateTransaction(expenseTransaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	if expenseInput.Fulfilled {
		err = s.store.UpdateAccountBalance(expenseInput.AccountID, expenseInput.Amount, types.TransactionTypeDebit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
	}

	respondWithJSON(w, http.StatusOK, expenseTransaction)
}

func (s *APIServer) handleCreateIncome(w http.ResponseWriter, r *http.Request) {
	type CreateIncomeInput struct {
		Amount      float32   `json:"amount"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
		CategoryId  uuid.UUID `json:"category_id"`
		AccountID   uuid.UUID `json:"account_id"`
		Fulfilled   bool      `json:"fulfilled"`
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
		Fulfilled:       incomeInput.Fulfilled,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.store.CreateTransaction(incomeTransaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if incomeInput.Fulfilled {
		err = s.store.UpdateAccountBalance(incomeInput.AccountID, incomeInput.Amount, types.TransactionTypeCredit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
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
