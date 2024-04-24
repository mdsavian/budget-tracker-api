package types

import (
	"time"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Session struct {
	ID        uuid.UUID
	UserId    string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
type User struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	EncryptedPassword string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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
