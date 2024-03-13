package main

import "github.com/google/uuid"

type AccountType string

const (
	Business = "Conta PJ"
	Personal = "Conta PF"
)

func (at AccountType) String() string {
	return string(at)
}

type Account struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Balance     int64       `json:"balance"`
	AccountType AccountType `json:"account_type"`
}

func NewAccount(name string, accountType AccountType) *Account {
	return &Account{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Balance:     0,
		AccountType: accountType}
}
