package main

import (
	"database/sql"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(uuid.UUID) error
	UpdateAccount(uuid.UUID) error
	GetAccountByID(uuid.UUID) (*Account, error)
	GetAccounts() ([]*Account, error)
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
	return s.CreateAccountTable()
}

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

	resp, err := s.db.Query(query, acc.ID, acc.Name, acc.AccountType, acc.Balance, acc.CreatedAt, acc.UpdatedAt)
	if err != nil {
		return err
	}

	return nil

}

func (s *PostgresStore) UpdateAccount(id uuid.UUID) error {
	return nil
}

func (s *PostgresStore) DeleteAccount(id uuid.UUID) error {
	return nil
}

func (s *PostgresStore) GetAccountByID(id uuid.UUID) (*Account, error) {
	return nil, nil
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("select * from account")
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}

	for rows.Next() {
		account := &Account{}
		err := rows.Scan(
			&account.ID,
			&account.CreatedAt,
			&account.UpdatedAt,
			&account.Balance,
			&account.Name,
			&account.AccountType)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}
