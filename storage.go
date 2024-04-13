package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/mdsavian/budget-tracker-api/types"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	postgresPort := os.Getenv("DB_PORT")
	postgresUser := os.Getenv("DB_USER")
	postgresPass := os.Getenv("DB_PASS")
	postgresDbName := os.Getenv("DB_NAME")
	dbSSL := os.Getenv("DB_SSL")

	connStr := fmt.Sprintf("user=%s dbname=%s password=%s port=%s sslmode=%s", postgresUser, postgresDbName, postgresPass, postgresPort, dbSSL)
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

func (s *PostgresStore) CreateUser(user *types.User) error {
	query := `insert into "user" 
	(id, name, email, password, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.Query(query, user.ID, user.Name, user.Email, user.EncryptedPassword, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) DeleteUser(id uuid.UUID) error {
	query := `delete from "user" where id = $1`

	_, err := s.db.Query(query, id)
	return err
}

func (s *PostgresStore) GetUserByEmail(email string) (*types.User, error) {
	query := "select * from user where email = $1"
	rows, err := s.db.Query(query, email)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoUser(rows)
	}

	return nil, fmt.Errorf("user with email %s not found", email)
}

func (s *PostgresStore) GetUserByID(id uuid.UUID) (*types.User, error) {
	query := `select * from "user" where id = $1`
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoUser(rows)
	}

	return nil, fmt.Errorf("user with id %v not found", id)
}

func scanIntoUser(rows *sql.Rows) (*types.User, error) {
	user := &types.User{}
	err := rows.Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Name,
		&user.Email,
		&user.EncryptedPassword)

	return user, err
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

func (s *PostgresStore) CreateAccount(acc *types.Account) error {
	query := `insert into account 
	(id, name, account_type, balance, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.Query(query, acc.ID, acc.Name, acc.AccountType, acc.Balance, acc.CreatedAt, acc.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) DeleteAccount(id uuid.UUID) error {
	query := "delete from account where id = $1"

	_, err := s.db.Query(query, id)
	return err
}

func (s *PostgresStore) GetAccountByID(id uuid.UUID) (*types.Account, error) {
	query := "select * from account where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}
	return nil, fmt.Errorf("account %v not found", id)

}

func (s *PostgresStore) GetAccounts() ([]*types.Account, error) {
	rows, err := s.db.Query("select * from account")
	if err != nil {
		return nil, err
	}

	accounts := []*types.Account{}

	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func scanIntoAccount(rows *sql.Rows) (*types.Account, error) {
	account := &types.Account{}
	err := rows.Scan(
		&account.ID,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Balance,
		&account.Name,
		&account.AccountType)
	return account, err
}
