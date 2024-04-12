package types

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(uuid.UUID) error
	GetAccountByID(uuid.UUID) (*Account, error)
	GetAccounts() ([]*Account, error)
	CreateUser(*User) error
	DeleteUser(uuid.UUID) error
	GetUserByID(uuid.UUID) (*User, error)
	GetUserByEmail(string) (*User, error)
}

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
type CreateNewUserInput struct {
	Name     string
	Email    string
	Password string
}

// TODO move this
func NewUser(input CreateNewUserInput) (*User, error) {
	encriptedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:                uuid.Must(uuid.NewV7()),
		Name:              input.Name,
		EncryptedPassword: string(encriptedPassword),
		Email:             input.Email,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}, nil
}

type AccountType string

const (
	AccountTypeBusiness AccountType = "Conta PJ"
	AccountTypePersonal AccountType = "Conta PF"
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

// TODO move this
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
