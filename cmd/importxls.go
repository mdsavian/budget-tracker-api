package cmd

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/storage"
	"github.com/mdsavian/budget-tracker-api/internal/types"
	"github.com/thedatashed/xlsxreader"
)

type Transaction struct {
	CreditCard      bool      `json:"creditCard"`
	TransactionType string    `json:"transaction_type"`
	Account         string    `json:"account"`
	Date            time.Time `json:"date"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	Amount          float64   `json:"amount"`
	Paid            bool      `json:"paid"`
}

func ImportData(path string, store *storage.PostgresStore) {

	// Create an instance of the reader by opening a target fileP
	xl, _ := xlsxreader.OpenFile(path)

	// Ensure the file reader is closed once utilised
	defer xl.Close()

	var transactions []*Transaction
	// Iterate on the rows of data
	for row := range xl.ReadRows(xl.Sheets[2]) {
		// ignore headers
		if row.Index == 1 {
			continue
		}

		cells := row.Cells

		if cells[0].Column != "A" {
			log.Println("ignoring row = ", row.Index)
			continue
		}

		dateString := cells[3].Value
		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			log.Fatal("error parsing date ", dateString)
		}

		var amount float64
		if cells[6].Type == xlsxreader.TypeNumerical {
			amountString := cells[6].Value

			amount, err = strconv.ParseFloat(amountString, 64)
			if err != nil {
				log.Fatal("error parsing amount ", amountString)
			}
		}

		transaction := &Transaction{
			CreditCard:      cells[0].Value == "Sim",
			TransactionType: cells[1].Value,
			Account:         cells[2].Value,
			Date:            date,
			Description:     cells[4].Value,
			Category:        cells[5].Value,
			Amount:          amount,
			Paid:            cells[7].Value == "Sim",
		}

		transactions = append(transactions, transaction)
	}

	persistData(transactions, store)

}

func persistData(transactions []*Transaction, store *storage.PostgresStore) {
	accounts, _ := store.GetAccounts()
	categories, _ := store.GetCategory()

	creditCard := getOrCreateCreditCard("Itaú", store)

	for _, transaction := range transactions {
		var account *types.Account
		accountName, accountType := mapTransactionAccount(transaction.Account)
		// search first on array avoiding calling the db for each transaction
		if len(accounts) > 0 {
			for _, acc := range accounts {
				if acc.Name == accountName && acc.AccountType == accountType {
					account = acc
					break
				}
			}
		}
		if account == nil {
			account = getOrCreateAccountOnDB(accountName, accountType, store)
			accounts = append(accounts, account)
		}

		var category *types.Category
		if len(categories) > 0 {
			for _, ctg := range categories {
				if ctg.Description == transaction.Category {
					category = ctg
					break
				}
			}
		}
		if category == nil {
			category = getOrCreateCategory(strings.ToLower(transaction.Category), store)
			categories = append(categories, category)
		}

		newTransaction := &types.Transaction{
			ID:              uuid.Must(uuid.NewV7()),
			AccountID:       account.ID,
			CategoryID:      category.ID,
			TransactionType: types.TransactionType(transaction.TransactionType),
			Date:            transaction.Date,
			Description:     transaction.Description,
			Amount:          transaction.Amount,
			Paid:            transaction.Paid,
			CostOfLiving:    false,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}

		if transaction.CreditCard {
			newTransaction.CreditCardID = &creditCard.ID
		}

		store.CreateTransaction(newTransaction)
	}
}

func getOrCreateCategory(description string, store *storage.PostgresStore) *types.Category {
	category, err := store.GetCategoryByDescription(description)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal("error searching for category ", description, err)
	}

	if category != nil {
		return category
	}

	newCategory := &types.Category{
		ID:          uuid.Must(uuid.NewV7()),
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	err = store.CreateCategory(newCategory)
	if err != nil {
		log.Fatal("error creating category ", description)
	}

	return newCategory
}

func getOrCreateCreditCard(name string, store *storage.PostgresStore) *types.CreditCard {
	creditCard, _ := store.GetCreditCardByName(name)
	if creditCard == nil {
		creditCard = &types.CreditCard{
			ID:         uuid.Must(uuid.NewV7()),
			Name:       name,
			ClosingDay: 10,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		err := store.CreateCreditCard(creditCard)
		if err != nil {
			log.Fatal("error creating credit card ")
		}
	}
	return creditCard
}

func mapTransactionAccount(transactionAccount string) (string, types.AccountType) {
	var accountName string
	accountType := types.AccountType(transactionAccount)

	if accountType == types.AccountTypePersonal {
		accountName = "Bradesco"
	} else {
		accountName = "Empresa"
	}

	return accountName, accountType
}

func getOrCreateAccountOnDB(accountName string, accountType types.AccountType, store *storage.PostgresStore) *types.Account {
	account, err := store.GetUniqueAccount(accountName, accountType)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal("error getting unique account", accountName, accountType)
	}

	if account != nil {
		return account
	}

	newAccount := &types.Account{
		ID:          uuid.Must(uuid.NewV7()),
		AccountType: accountType,
		Name:        accountName,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	err = store.CreateAccount(newAccount)
	if err != nil {
		log.Fatal("error creating account")
	}

	return newAccount

}
