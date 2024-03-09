package main

import "github.com/gofrs/uuid"

type AccountType int

const (
	Business = 0
	Personal = 1
)

func (at AccountType) String() string {
	switch at {
	case Business:
		return "Conta PJ"
	case Personal:
		return "Conta PF"
	}
	return "Invalid Account type"
}

type Account struct {
	ID          uuid.UUID
	Name        string
	Balance     int64
	AccountType AccountType
}

func NewAccount(name string, accountType AccountType) *Account {
	return &Account{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Balance:     0,
		AccountType: accountType}
}
