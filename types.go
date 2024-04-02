package main

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	EncryptedPassword string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewUser(name, email, password string) (*User, error) {
	encriptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:                uuid.Must(uuid.NewV7()),
		Name:              name,
		EncryptedPassword: string(encriptedPassword),
		Email:             email,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}, nil
}

type AccountType string

const (
	Business = "Conta PJ"
	Personal = "Conta PF"
)

func (at AccountType) String() string {
	return string(at)
}

type CreateNewAccountInput struct {
	Name        string      `json:"name"`
	AccountType AccountType `json:"account_type"`
}
type Account struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Balance     int64       `json:"balance"`
	AccountType AccountType `json:"account_type"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func NewAccount(name string, accountType AccountType) *Account {
	return &Account{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Balance:     0,
		AccountType: accountType,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}
