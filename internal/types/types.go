package types

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Session struct {
	ID        uuid.UUID
	UserId    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s Session) IsExpired() bool {
	return s.ExpiresAt.Before(time.Now().UTC())
}

type User struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	EncryptedPassword string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (u *User) ValidPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password)) == nil
}

type AccountType string

const (
	AccountTypeBusiness AccountType = "Conta PJ"
	AccountTypePersonal AccountType = "Conta PF"
)

func (at AccountType) String() string {
	return string(at)
}

type Account struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Balance     float32     `json:"balance"`
	AccountType AccountType `json:"account_type"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Category struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreditCard struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Archived   bool      `json:"archived"`
	DueDay     int       `json:"dueDay"`
	ClosingDay int       `json:"closingDay"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type TransactionType string

const (
	TransactionTypeCredit TransactionType = "Credit"
	TransactionTypeDebit  TransactionType = "Debit"
)

func (at TransactionType) String() string {
	return string(at)
}

type Transaction struct {
	ID                     uuid.UUID  `json:"id"`
	AccountID              uuid.UUID  `json:"accountId"`
	CreditCardID           *uuid.UUID `json:"creditCardId"`
	CategoryID             uuid.UUID  `json:"categoryId"`
	RecurringTransactionID *uuid.UUID `json:"recurringTransactionId"`

	TransactionType TransactionType `json:"transactionType"`
	Date            time.Time       `json:"date"`
	Description     string          `json:"description"`
	Amount          float32         `json:"amount"`
	Fulfilled       bool            `json:"paid"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TransactionView struct {
	ID                     uuid.UUID       `json:"id"`
	AccountID              uuid.UUID       `json:"accountId"`
	Account                string          `json:"account"`
	CreditCardID           *uuid.UUID      `json:"creditCardId"`
	CreditCard             *string         `json:"creditCard"`
	CategoryID             uuid.UUID       `json:"categoryId"`
	Category               string          `json:"category"`
	RecurringTransactionID *uuid.UUID      `json:"recurringTransactionId"`
	TransactionType        TransactionType `json:"transactionType"`
	Date                   time.Time       `json:"date"`
	Description            string          `json:"description"`
	Amount                 float64         `json:"amount"`
	Fulfilled              bool            `json:"paid"`
}

type RecurringTransaction struct {
	ID           uuid.UUID  `json:"id"`
	AccountID    uuid.UUID  `json:"accountId"`
	CreditCardID *uuid.UUID `json:"creditCardId"`
	CategoryID   uuid.UUID  `json:"categoryId"`

	TransactionType TransactionType `json:"transactionType"`
	Day             int             `json:"day"`
	Description     string          `json:"description"`
	Amount          float32         `json:"amount"`
	Archived        bool            `json:"archived"`
	CreatedAt       time.Time       `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}
