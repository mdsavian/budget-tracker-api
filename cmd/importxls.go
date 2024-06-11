package cmd

import (
	"log"
	"strconv"
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

		log.Println("adding row", *transaction)

		transactions = append(transactions, transaction)
	}

	log.Println(transactions, len(transactions))
	persistData(transactions, store)

}

func persistData(transactions []*Transaction, store *storage.PostgresStore) {
	var accounts []*types.Account

	for _, transaction := range transactions {
		var accountName string
		accountType := types.AccountType(transaction.Account)

		if accountType == types.AccountTypePersonal {
			accountName = "Bradesco"
		} else {
			accountName = "Empresa"
		}

		var account *types.Account
		if len(accounts) > 0 {
			for _, acc := range accounts {
				if acc.Name == accountName && acc.AccountType == accountType {
					account = acc
					break
				}
			}
		}

		if account == nil {
			account, err := store.GetUniqueAccount(accountName, accountType)
			if err != nil {
				log.Fatal("error getting unique account", accountName, accountType)
			}

			if account == nil {
				account, err = store.CreateAccount(&types.Account{
					ID:          uuid.Must(uuid.NewV7()),
					AccountType: accountType,
					Name:        accountName,
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				})
				if err != nil {
					log.Fatal("error creating account")
				}
			}

			accounts = append(accounts, account)
		}
	}

}
