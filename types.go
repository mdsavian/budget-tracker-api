package main

import "github.com/google/uuid"

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
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Balance     int64       `json:"balance"`
	AccountType AccountType `json:"accountType"`
}

func NewAccount(name string, accountType AccountType) *Account {
	return &Account{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Balance:     0,
		AccountType: accountType}
}
