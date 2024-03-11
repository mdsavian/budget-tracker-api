package main

import "github.com/google/uuid"

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(uuid.UUID) error
	UpdateAccount(uuid.UUID) error
	GetAccountByID(uuid.UUID) (*Account, error)
}
