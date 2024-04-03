package main

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(uuid.UUID) error
	UpdateAccount(uuid.UUID) error
	GetAccountByID(uuid.UUID) (*Account, error)
	GetAccounts() ([]*Account, error)
	CreateUser(*User) error
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {

	connStr := "user=marlon dbname=budgettracker password=marlon port=5433 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Init() error {
	return s.createTables()
}

func (s *PostgresStore) createTables() error {
	if err := s.CreateAccountTable(); err != nil {
		return err
	}

	if err := s.CreateUserTable(); err != nil {
		return err
	}

	return nil
}

// User
func (s *PostgresStore) CreateUserTable() error {
	query := `create table if not exists "user" (
				id UUID primary key NOT NULL, 
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				name varchar (200) NOT NULL, 
				email varchar (200) NOT NULL, 
				password varchar NOT NULL
				)`
	_, err := s.db.Query(query)
	return err
}

func (s *PostgresStore) CreateUser(user *User) error {
	query := `insert into "user" 
	(id, name, email, password, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.Query(query, user.ID, user.Name, user.Email, user.EncryptedPassword, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

// Account
func (s *PostgresStore) CreateAccountTable() error {
	query := `create table if not exists account (
				id UUID primary key NOT NULL, 
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				balance bigint NOT NULL, 
				name varchar (200) NOT NULL, 
				account_type varchar (50) NOT null
				)`
	_, err := s.db.Query(query)
	return err
}

func (s *PostgresStore) CreateAccount(acc *Account) error {
	query := `insert into account 
	(id, name, account_type, balance, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.Query(query, acc.ID, acc.Name, acc.AccountType, acc.Balance, acc.CreatedAt, acc.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) UpdateAccount(id uuid.UUID) error {
	return nil
}

func (s *PostgresStore) DeleteAccount(id uuid.UUID) error {
	query := "delete from account where id = $1"

	_, err := s.db.Query(query, id)
	return err
}

func (s *PostgresStore) GetAccountByID(id uuid.UUID) (*Account, error) {
	query := "select * from account where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}
	return nil, fmt.Errorf("account %s not found", id)

}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("select * from account")
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}

	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := &Account{}
	err := rows.Scan(
		&account.ID,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Balance,
		&account.Name,
		&account.AccountType)
	return account, err
}
