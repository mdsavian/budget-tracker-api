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
				name varchar (200) NOT NULL, 
				account_type varchar (50) NOT null
				)`

	_, err := s.db.Query(query)
	return err

}

func (s *PostgresStore) CreateAccount(*Account) error {
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
