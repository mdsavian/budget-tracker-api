package apiserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/types"
	"github.com/samber/lo"
)

type CreateCreditCardExpenseInput struct {
	CreditCardID uuid.UUID `json:"creditCardId"`
	AccountID    uuid.UUID `json:"accountId"`
	CategoryId   uuid.UUID `json:"categoryId"`
	Amount       float32   `json:"amount"`
	Date         string    `json:"date"`
	Description  string    `json:"description"`
	Installments int32     `json:"installments"`
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

	for i := 0; i < int(expenseInput.Installments); i++ {
		installmentDate := creditCardExpenseDate.AddDate(0, i, 0)

		installmentTransaction := &types.Transaction{
			ID:              uuid.Must(uuid.NewV7()),
			CategoryID:      expenseInput.CategoryId,
			AccountID:       expenseInput.AccountID,
			CreditCardID:    &expenseInput.CreditCardID,
			TransactionType: types.TransactionTypeDebit,
			Amount:          amountPerInstallment,
			Date:            installmentDate,
			Description:     expenseInput.Description + " (" + strconv.Itoa(i+1) + "/" + strconv.Itoa(int(expenseInput.Installments)) + ")",
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
		CategoryId   uuid.UUID  `json:"categoryId"`
		CreditCardId *uuid.UUID `json:"creditCardId"`
		AccountID    uuid.UUID  `json:"accountId"`
		Amount       float32    `json:"amount"`
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

func (s *APIServer) handleEffectuateTransaction(w http.ResponseWriter, r *http.Request) {
	type EffectuateTransactionInput struct {
		TransactionID          uuid.UUID `json:"transactionId"`
		RecurringTransactionID uuid.UUID `json:"recurringTransactionId"`
		Amount                 float32   `json:"amount"`
	}

	effectuateTransactionInout := &EffectuateTransactionInput{}
	if err := json.NewDecoder(r.Body).Decode(&effectuateTransactionInout); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if effectuateTransactionInout.TransactionID == uuid.Nil && effectuateTransactionInout.RecurringTransactionID == uuid.Nil {
		respondWithError(w, http.StatusBadRequest, "transactionId or recurringTransactionId is required")
		return
	}

	transaction := &types.Transaction{}
	if effectuateTransactionInout.TransactionID != uuid.Nil {
		transaction, err := s.store.GetTransactionByID(effectuateTransactionInout.TransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		if transaction.Fulfilled {
			respondWithError(w, http.StatusBadRequest, "transaction already fulfilled")
			return
		}

		err = s.store.FulfillTransaction(effectuateTransactionInout.TransactionID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if effectuateTransactionInout.RecurringTransactionID != uuid.Nil {
		recurringTransaction, err := s.store.GetRecurringTransactionByID(effectuateTransactionInout.RecurringTransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		transaction = &types.Transaction{
			ID:                     uuid.Must(uuid.NewV7()),
			RecurringTransactionID: lo.ToPtr(recurringTransaction.ID),
			CategoryID:             recurringTransaction.CategoryID,
			AccountID:              recurringTransaction.AccountID,
			CreditCardID:           recurringTransaction.CreditCardID,
			TransactionType:        types.TransactionTypeDebit,
			Amount:                 effectuateTransactionInout.Amount,
			Date:                   time.Now().UTC(),
			Description:            recurringTransaction.Description,
			Fulfilled:              true,
			CreatedAt:              time.Now().UTC(),
			UpdatedAt:              time.Now().UTC(),
		}

		err = s.store.CreateTransaction(transaction)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	err := s.store.UpdateAccountBalance(transaction.AccountID, transaction.Amount, transaction.TransactionType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	type UpdateTransactionInput struct {
		TransactionID              *uuid.UUID `json:"transactionId"`
		AccountID                  uuid.UUID  `json:"accountId"`
		CreditCardID               *uuid.UUID `json:"creditCardId"`
		CategoryID                 uuid.UUID  `json:"categoryId"`
		RecurringTransactionID     *uuid.UUID `json:"recurringTransactionId"`
		Date                       string     `json:"date"`
		Description                string     `json:"description"`
		Amount                     float32    `json:"amount"`
		UpdateRecurringTransaction bool       `json:"updateRecurringTransaction"`
	}

	updateInput := UpdateTransactionInput{}
	if err := json.NewDecoder(r.Body).Decode(&updateInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if updateInput.TransactionID == nil && updateInput.RecurringTransactionID == nil {
		respondWithJSON(w, http.StatusBadRequest, "transactionId or recurringTransactionId is required")
		return
	}

	transactionDate, err := time.Parse("2006-01-02", updateInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	transaction := &types.Transaction{
		AccountID:              updateInput.AccountID,
		CreditCardID:           updateInput.CreditCardID,
		CategoryID:             updateInput.CategoryID,
		Amount:                 updateInput.Amount,
		Description:            updateInput.Description,
		Date:                   transactionDate,
		RecurringTransactionID: updateInput.RecurringTransactionID,
	}

	if updateInput.TransactionID != nil && *updateInput.TransactionID != uuid.Nil {
		transactionFromDb, err := s.store.GetTransactionByID(*updateInput.TransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if transactionFromDb == nil {
			respondWithError(w, http.StatusBadRequest, "transaction not found")
			return
		}

		err = s.store.UpdateTransaction(*updateInput.TransactionID, transaction)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else if !updateInput.UpdateRecurringTransaction {
		// if dont update recurring and dont have transaction ID we need to create a new transaction
		transaction.ID = uuid.Must(uuid.NewV7())
		transaction.TransactionType = types.TransactionTypeDebit
		err := s.store.CreateTransaction(transaction)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	if updateInput.RecurringTransactionID != nil && *updateInput.RecurringTransactionID != uuid.Nil && updateInput.UpdateRecurringTransaction {
		recurringTransaction, err := s.store.GetRecurringTransactionByID(*updateInput.RecurringTransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if recurringTransaction == nil {
			respondWithError(w, http.StatusBadRequest, "recurring transaction not found")
			return
		}

		recurringTransactionToUpdate := &types.RecurringTransaction{
			AccountID:    updateInput.AccountID,
			CreditCardID: updateInput.CreditCardID,
			CategoryID:   updateInput.CategoryID,
			Amount:       updateInput.Amount,
			Description:  updateInput.Description,
			Day:          transactionDate.Day(),
		}

		if err := s.store.UpdateRecurringTransaction(*updateInput.RecurringTransactionID, recurringTransactionToUpdate); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	respondWithJSON(w, http.StatusOK, "Transaction updated")
}
