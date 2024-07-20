package apiserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
)

type CreateCreditCardExpenseInput struct {
	CreditCardID uuid.UUID `json:"creditCardId"`
	AccountID    uuid.UUID `json:"accountId"`
	CategoryId   uuid.UUID `json:"categoryId"`
	Amount       string    `json:"amount"`
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

	fAmount, err := strconv.ParseFloat(*&expenseInput.Amount, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	amount := float32(fAmount)

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
		Amount:          amount,
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

	fAmount, err := strconv.ParseFloat(*&expenseInput.Amount, 32)
	if err != nil {
		return nil, err
	}
	amount := float32(fAmount)

	amountPerInstallment := amount / float32(expenseInput.Installments)
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

	fAmount, err := strconv.ParseFloat(*&creditCardExpenseInput.Amount, 32)
	if err != nil {
		return nil, err
	}
	amount := float32(fAmount)

	recurringTransactionID := uuid.Must(uuid.NewV7())

	creditCardRecurringTransaction := &types.Transaction{
		ID:                     uuid.Must(uuid.NewV7()),
		CategoryID:             creditCardExpenseInput.CategoryId,
		AccountID:              creditCardExpenseInput.AccountID,
		CreditCardID:           &creditCardExpenseInput.CreditCardID,
		RecurringTransactionID: &recurringTransactionID,
		TransactionType:        types.TransactionTypeDebit,
		Amount:                 amount,
		Date:                   creditCardExpenseDate,
		Description:            creditCardExpenseInput.Description,
		Fulfilled:              false,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}

	err = s.store.CreateRecurringTransaction(&types.RecurringTransaction{
		ID:              recurringTransactionID,
		AccountID:       creditCardExpenseInput.AccountID,
		CategoryID:      creditCardExpenseInput.CategoryId,
		CreditCardID:    &creditCardExpenseInput.CreditCardID,
		TransactionType: types.TransactionTypeDebit,
		Day:             creditCardExpenseDate.Day(),
		Description:     creditCardExpenseInput.Description,
		Amount:          amount,
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
		CategoryId   uuid.UUID  `json:"categoryId"`
		CreditCardId *uuid.UUID `json:"creditCardId"`
		AccountID    uuid.UUID  `json:"accountId"`
		Amount       string     `json:"amount"`
		Date         string     `json:"date"`
		Description  string     `json:"description"`
		Fulfilled    bool       `json:"fulfilled"`
		Fixed        bool       `json:"fixed"`
	}
	expenseInput := CreateExpenseInput{}
	if err := json.NewDecoder(r.Body).Decode(&expenseInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if expenseInput.CreditCardId != nil {
		s.handleCreateCreditCardExpense(w, r)
		return

	}
	fAmount, err := strconv.ParseFloat(expenseInput.Amount, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	amount := float32(fAmount)

	expenseDate, err := time.Parse("2006-01-02", expenseInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	expenseTransaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		TransactionType: types.TransactionTypeDebit,
		Amount:          amount,
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
			Amount:          amount,
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
		err = s.store.UpdateAccountBalance(expenseInput.AccountID, amount, types.TransactionTypeDebit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
	}

	respondWithJSON(w, http.StatusOK, expenseTransaction)
}

func (s *APIServer) handleCreateIncome(w http.ResponseWriter, r *http.Request) {
	type CreateIncomeInput struct {
		Amount      string    `json:"amount"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
		CategoryId  uuid.UUID `json:"categoryId"`
		AccountID   uuid.UUID `json:"accountId"`
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
	fAmount, err := strconv.ParseFloat(*&incomeInput.Amount, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	amount := float32(fAmount)

	incomeTransaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		TransactionType: types.TransactionTypeCredit,
		Amount:          amount,
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
		err = s.store.UpdateAccountBalance(incomeInput.AccountID, amount, types.TransactionTypeCredit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondWithJSON(w, http.StatusOK, incomeTransaction)
}

func (s *APIServer) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	type UpdateTransactionInput struct {
		TransactionID uuid.UUID  `json:"transactionId"`
		AccountID     *uuid.UUID `json:"accountId"`
		CreditCardID  *uuid.UUID `json:"creditCardId"`
		CategoryID    *uuid.UUID `json:"categoryId"`

		Date                       *string `json:"date"`
		Description                *string `json:"description"`
		Amount                     *string `json:"amount"`
		Fulfilled                  *bool   `json:"fulfilled"`
		UpdateRecurringTransaction *bool   `json:"updateRecurringTransaction"`
	}

	updateInput := UpdateTransactionInput{}
	if err := json.NewDecoder(r.Body).Decode(&updateInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	fAmount, err := strconv.ParseFloat(*updateInput.Amount, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	amount := float32(fAmount)

	transaction, err := s.store.GetTransactionByID(updateInput.TransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if updateInput.AccountID != nil {
		transaction.AccountID = *updateInput.AccountID
	}
	if updateInput.CreditCardID != nil {
		transaction.CreditCardID = updateInput.CreditCardID
	}
	if updateInput.CategoryID != nil {
		transaction.CategoryID = *updateInput.CategoryID
	}
	if updateInput.Date != nil {
		date, err := time.Parse("2006-01-02", *updateInput.Date)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		transaction.Date = date
	}
	if updateInput.Description != nil {
		transaction.Description = *updateInput.Description
	}
	if updateInput.Amount != nil {
		transaction.Amount = amount
	}
	if updateInput.Fulfilled != nil {
		transaction.Fulfilled = *updateInput.Fulfilled
	}

	if err := s.store.UpdateTransaction(updateInput.TransactionID, transaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if updateInput.UpdateRecurringTransaction != nil && *updateInput.UpdateRecurringTransaction {
		if transaction.RecurringTransactionID != nil {
			recurringTransaction, err := s.store.GetRecurringTransactionByID(*transaction.RecurringTransactionID)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, err.Error())
				return
			}

			recurringTransaction.AccountID = transaction.AccountID
			recurringTransaction.CategoryID = transaction.CategoryID
			recurringTransaction.CreditCardID = transaction.CreditCardID
			recurringTransaction.Day = transaction.Date.Day()
			recurringTransaction.Description = transaction.Description
			recurringTransaction.Amount = transaction.Amount

			log.Println("entrei 12121221")
			if err := s.store.UpdateRecurringTransaction(*transaction.RecurringTransactionID, recurringTransaction); err != nil {
				respondWithError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	respondWithJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) handleEffectuateTransaction(w http.ResponseWriter, r *http.Request) {
	uTransactionID, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	transaction, err := s.store.GetTransactionByID(uTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if transaction.Fulfilled {
		respondWithError(w, http.StatusBadRequest, "transaction already fulfilled")
		return
	}

	transaction.Fulfilled = true
	err = s.store.UpdateTransaction(uTransactionID, transaction)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = s.store.UpdateAccountBalance(transaction.AccountID, transaction.Amount, transaction.TransactionType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, transaction)
}
