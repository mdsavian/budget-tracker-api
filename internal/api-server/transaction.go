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

type CreateCreditCardDebitInput struct {
	CreditCardID uuid.UUID `json:"creditCardId"`
	AccountID    uuid.UUID `json:"accountId"`
	CategoryId   uuid.UUID `json:"categoryId"`
	Amount       float32   `json:"amount"`
	Date         string    `json:"date"`
	Description  string    `json:"description"`
	Installments int32     `json:"installments"`
	Fixed        bool      `json:"fixed"`
}

func (s *APIServer) handleCreateCreditCardDebit(w http.ResponseWriter, r *http.Request) {
	debitInput := CreateCreditCardDebitInput{}
	if err := json.NewDecoder(r.Body).Decode(&debitInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if debitInput.CreditCardID == uuid.Nil {
		respondWithError(w, http.StatusBadRequest, "credit card is required")
		return
	}

	if debitInput.Installments > 0 && debitInput.Fixed {
		respondWithError(w, http.StatusBadRequest, "installments and fixed cannot be used together")
		return
	}

	creditCard, err := s.store.GetCreditCardByID(debitInput.CreditCardID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	creditCardDebitDate, err := time.Parse("2006-01-02", debitInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	debitDayToAdd := creditCard.DueDay - creditCardDebitDate.Day()
	if creditCardDebitDate.Day() < creditCard.ClosingDay {
		creditCardDebitDate = creditCardDebitDate.AddDate(0, 0, debitDayToAdd)
	} else {
		creditCardDebitDate = creditCardDebitDate.AddDate(0, 1, debitDayToAdd)
	}

	if debitInput.Fixed {
		creditCardRecurringTransaction, err := s.createRecurringCreditCardDebit(debitInput, creditCardDebitDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, creditCardRecurringTransaction)
		return
	}

	if debitInput.Installments > 0 {
		firstInstallmentTransaction, err := s.createCreditCardDebitInstallments(debitInput, creditCardDebitDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, firstInstallmentTransaction)
		return
	}

	transaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		CategoryID:      debitInput.CategoryId,
		AccountID:       debitInput.AccountID,
		CreditCardID:    &debitInput.CreditCardID,
		TransactionType: types.TransactionTypeDebit,
		Amount:          debitInput.Amount,
		Date:            creditCardDebitDate,
		Description:     debitInput.Description,
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

func (s *APIServer) createCreditCardDebitInstallments(debitInput CreateCreditCardDebitInput, creditCardDebitDate time.Time) (*types.Transaction, error) {
	amountPerInstallment := debitInput.Amount / float32(debitInput.Installments)
	var firstInstallmentTransaction *types.Transaction

	for i := 0; i < int(debitInput.Installments); i++ {
		installmentDate := creditCardDebitDate.AddDate(0, i, 0)

		installmentTransaction := &types.Transaction{
			ID:              uuid.Must(uuid.NewV7()),
			CategoryID:      debitInput.CategoryId,
			AccountID:       debitInput.AccountID,
			CreditCardID:    &debitInput.CreditCardID,
			TransactionType: types.TransactionTypeDebit,
			Amount:          amountPerInstallment,
			Date:            installmentDate,
			Description:     debitInput.Description + " (" + strconv.Itoa(i+1) + "/" + strconv.Itoa(int(debitInput.Installments)) + ")",
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

func (s *APIServer) createRecurringCreditCardDebit(creditCardDebitInput CreateCreditCardDebitInput, creditCardDebitDate time.Time) (*types.Transaction, error) {
	recurringTransactionID := uuid.Must(uuid.NewV7())

	creditCardRecurringTransaction := &types.Transaction{
		ID:                     uuid.Must(uuid.NewV7()),
		CategoryID:             creditCardDebitInput.CategoryId,
		AccountID:              creditCardDebitInput.AccountID,
		CreditCardID:           &creditCardDebitInput.CreditCardID,
		RecurringTransactionID: &recurringTransactionID,
		TransactionType:        types.TransactionTypeDebit,
		Amount:                 creditCardDebitInput.Amount,
		Date:                   creditCardDebitDate,
		Description:            creditCardDebitInput.Description,
		Fulfilled:              false,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}

	err := s.store.CreateRecurringTransaction(&types.RecurringTransaction{
		ID:              recurringTransactionID,
		AccountID:       creditCardDebitInput.AccountID,
		CategoryID:      creditCardDebitInput.CategoryId,
		CreditCardID:    &creditCardDebitInput.CreditCardID,
		TransactionType: types.TransactionTypeDebit,
		Day:             creditCardDebitDate.Day(),
		Description:     creditCardDebitInput.Description,
		Amount:          creditCardDebitInput.Amount,
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

func (s *APIServer) handleCreateDebit(w http.ResponseWriter, r *http.Request) {
	type CreateDebitInput struct {
		CategoryId  uuid.UUID `json:"categoryId"`
		AccountID   uuid.UUID `json:"accountId"`
		Amount      float32   `json:"amount"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
		Fulfilled   bool      `json:"fulfilled"`
		Fixed       bool      `json:"fixed"`
	}

	debitInput := CreateDebitInput{}
	if err := json.NewDecoder(r.Body).Decode(&debitInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	debitDate, err := time.Parse("2006-01-02", debitInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	debitTransaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		TransactionType: types.TransactionTypeDebit,
		Amount:          debitInput.Amount,
		Date:            debitDate,
		Description:     debitInput.Description,
		CategoryID:      debitInput.CategoryId,
		AccountID:       debitInput.AccountID,
		Fulfilled:       debitInput.Fulfilled,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if debitInput.Fixed {
		recurringTransactionID := uuid.Must(uuid.NewV7())

		err := s.store.CreateRecurringTransaction(&types.RecurringTransaction{
			ID:              recurringTransactionID,
			AccountID:       debitInput.AccountID,
			CategoryID:      debitInput.CategoryId,
			TransactionType: types.TransactionTypeDebit,
			Day:             debitDate.Day(),
			Description:     debitInput.Description,
			Amount:          debitInput.Amount,
			Archived:        false,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		debitTransaction.RecurringTransactionID = &recurringTransactionID
	}

	if err := s.store.CreateTransaction(debitTransaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	if debitInput.Fulfilled {
		err = s.store.UpdateAccountBalance(debitInput.AccountID, debitInput.Amount, types.TransactionTypeDebit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
	}

	respondWithJSON(w, http.StatusOK, debitTransaction)
}

func (s *APIServer) handleCreateCredit(w http.ResponseWriter, r *http.Request) {
	type CreateCreditInput struct {
		Amount      float32   `json:"amount"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
		CategoryId  uuid.UUID `json:"categoryId"`
		AccountID   uuid.UUID `json:"accountId"`
		Fulfilled   bool      `json:"fulfilled"`
	}

	creditInput := CreateCreditInput{}
	if err := json.NewDecoder(r.Body).Decode(&creditInput); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	creditDate, err := time.Parse("2006-01-02", creditInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	creditTransaction := &types.Transaction{
		ID:              uuid.Must(uuid.NewV7()),
		TransactionType: types.TransactionTypeCredit,
		Amount:          creditInput.Amount,
		Date:            creditDate,
		Description:     creditInput.Description,
		CategoryID:      creditInput.CategoryId,
		AccountID:       creditInput.AccountID,
		Fulfilled:       creditInput.Fulfilled,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.store.CreateTransaction(creditTransaction); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if creditInput.Fulfilled {
		err = s.store.UpdateAccountBalance(creditInput.AccountID, creditInput.Amount, types.TransactionTypeCredit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondWithJSON(w, http.StatusOK, creditTransaction)
}

func (s *APIServer) handleEffectuateTransaction(w http.ResponseWriter, r *http.Request) {
	type EffectuateTransactionInput struct {
		TransactionID          uuid.UUID `json:"transactionId"`
		RecurringTransactionID uuid.UUID `json:"recurringTransactionId"`
		Amount                 float32   `json:"amount"`
		Date                   string    `json:"date"`
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
	var err error
	if effectuateTransactionInout.TransactionID != uuid.Nil {
		transaction, err = s.store.GetTransactionByID(effectuateTransactionInout.TransactionID)
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

		date, err := time.Parse("2006-01-02", effectuateTransactionInout.Date)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		transaction = &types.Transaction{
			ID:                     uuid.Must(uuid.NewV7()),
			AccountID:              recurringTransaction.AccountID,
			CreditCardID:           recurringTransaction.CreditCardID,
			CategoryID:             recurringTransaction.CategoryID,
			RecurringTransactionID: lo.ToPtr(recurringTransaction.ID),
			TransactionType:        types.TransactionTypeDebit,
			EffectuatedDate:        lo.ToPtr(time.Now().UTC()),
			Date:                   date,
			Description:            recurringTransaction.Description,
			Amount:                 effectuateTransactionInout.Amount,
			Fulfilled:              true,
			CreatedAt:              time.Time{},
			UpdatedAt:              time.Now().UTC(),
		}

		err = s.store.CreateTransaction(transaction)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	err = s.store.UpdateAccountBalance(transaction.AccountID, transaction.Amount, transaction.TransactionType)
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
		CreditCardID               *string    `json:"creditCardId"`
		CategoryID                 uuid.UUID  `json:"categoryId"`
		RecurringTransactionID     *string    `json:"recurringTransactionId"`
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

	var uRecurringTransactionID *uuid.UUID
	var uCreditCardID *uuid.UUID
	var err error

	transactionDate, err := time.Parse("2006-01-02", updateInput.Date)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if updateInput.RecurringTransactionID != nil && *updateInput.RecurringTransactionID != "" {
		parsedRecurringTransactionID, err := uuid.Parse(*updateInput.RecurringTransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		uRecurringTransactionID = &parsedRecurringTransactionID

	}

	if updateInput.CreditCardID != nil && *updateInput.CreditCardID != "" {
		parsedCreditCardId, err := uuid.Parse(*updateInput.CreditCardID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		uCreditCardID = &parsedCreditCardId
	}

	transaction := &types.Transaction{
		AccountID:              updateInput.AccountID,
		CreditCardID:           uCreditCardID,
		CategoryID:             updateInput.CategoryID,
		Amount:                 updateInput.Amount,
		Description:            updateInput.Description,
		Date:                   transactionDate,
		RecurringTransactionID: uRecurringTransactionID,
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

	if uRecurringTransactionID != nil && *uRecurringTransactionID != uuid.Nil && updateInput.UpdateRecurringTransaction {
		recurringTransaction, err := s.store.GetRecurringTransactionByID(*uRecurringTransactionID)
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
			CreditCardID: uCreditCardID,
			CategoryID:   updateInput.CategoryID,
			Amount:       updateInput.Amount,
			Description:  updateInput.Description,
			Day:          transactionDate.Day(),
		}

		if err := s.store.UpdateRecurringTransaction(*uRecurringTransactionID, recurringTransactionToUpdate); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	respondWithJSON(w, http.StatusOK, "Transaction updated")
}

func (s *APIServer) handleGetTransactionByID(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	transactionId := queryValues.Get("id")
	isRecurringQueryParam := queryValues.Get("isRecurring")
	transactionDate := queryValues.Get("date")

	if transactionId == "" {
		respondWithError(w, http.StatusBadRequest, "id is required")
		return
	}

	isRecurring, err := strconv.ParseBool(isRecurringQueryParam)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var parsedTransactionDate time.Time
	if transactionDate != "" {
		parsedTransactionDate, err = time.Parse("02/01/2006", transactionDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		parsedTransactionDate = time.Now().UTC()
	}

	uTransactionID, err := uuid.Parse(transactionId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if isRecurring {
		recurringTransaction, err := s.store.GetRecurringTransactionByID(uTransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		transactionFormatted := &types.Transaction{
			AccountID:              recurringTransaction.AccountID,
			CreditCardID:           recurringTransaction.CreditCardID,
			CategoryID:             recurringTransaction.CategoryID,
			RecurringTransactionID: &recurringTransaction.ID,
			TransactionType:        recurringTransaction.TransactionType,
			Date:                   parsedTransactionDate,
			Description:            recurringTransaction.Description,
			Amount:                 recurringTransaction.Amount,
			Fulfilled:              false,
		}

		respondWithJSON(w, http.StatusOK, transactionFormatted)
		return
	}

	transaction, err := s.store.GetTransactionByID(uTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID, err := getAndParseIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = s.store.DeleteTransaction(transactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, "Transaction deleted")
}
