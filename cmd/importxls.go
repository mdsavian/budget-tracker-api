package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mdsavian/budget-tracker-api/internal/storage"
	"github.com/mdsavian/budget-tracker-api/internal/types"
	"github.com/thedatashed/xlsxreader"
)

type Transaction struct {
	CreditCard      bool
	TransactionType string
	Account         string
	Date            time.Time
	Description     string
	Category        string
	Amount          float32
	FulFilled       bool
}

func ImportData(path string, store *storage.PostgresStore) {

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".xlsx" {
			fmt.Println("Importing file name:", info.Name())

			readXlsx(path, store)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
	}

}

func readXlsx(path string, store *storage.PostgresStore) {
	// Create an instance of the reader by opening a target fileP
	xl, _ := xlsxreader.OpenFile(path)

	// Ensure the file reader is closed once utilised
	defer xl.Close()
	var lastValidDate time.Time

	var transactions []*Transaction
	// Iterate on the rows of data
	for row := range xl.ReadRows(xl.Sheets[2]) {
		// ignore headers
		if row.Index == 1 {
			continue
		}
		log.Println("reading row = ", row.Index)

		cells := row.Cells

		if cells[0].Column != "A" || len(cells) < 7 {
			log.Println("ignoring row = ", row.Index)
			continue
		}

		dateString := cells[3].Value
		notFoundDate := false
		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			notFoundDate = true
			date = time.Date(lastValidDate.Year(), lastValidDate.Month(), 1, 0, 0, 0, 0, time.UTC)
			log.Print("not found date, new date: ", date)
		}
		lastValidDate = date

		index := 6
		if notFoundDate {
			index = 5
		}

		var amount float64 = 0
		if cells[index].Type == xlsxreader.TypeNumerical {
			amountString := cells[index].Value
			amount, err = strconv.ParseFloat(amountString, 32)
			if err != nil {
				log.Fatal("error parsing amount ", amountString)
			}
		}

		if amount == 0 || amount < 0 {
			log.Println("amount is <= 0, ignoring row = ", row.Index)
			continue
		}

		index = 4
		if notFoundDate {
			index = 3
		}

		if cells[index+3].Value != "Sim" {
			log.Println("not fulfilled, ignoring row = ", row.Index)
			continue
		}

		transaction := &Transaction{
			CreditCard:      cells[0].Value == "Sim",
			TransactionType: cells[1].Value,
			Account:         cells[2].Value,
			Date:            date,
			Description:     cells[index].Value,
			Category:        cells[index+1].Value,
			Amount:          float32(amount),
			FulFilled:       cells[index+3].Value == "Sim",
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
			Fulfilled:       transaction.FulFilled,
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
			DueDay:     16,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		err := store.CreateCreditCard(creditCard)
		if err != nil {
			log.Fatal("error creating credit card ", err)
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
