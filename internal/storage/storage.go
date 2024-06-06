package storage

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/mdsavian/budget-tracker-api/internal/types"
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

	if err := s.CreateSessionTable(); err != nil {
		return err
	}

	if err := s.CreateUserTable(); err != nil {
		return err
	}

	if err := s.CreateCategoryTable(); err != nil {
		return err
	}

	if err := s.CreateCreditCardTable(); err != nil {
		return err
	}

	return nil
}

//CreditCard

func (s *PostgresStore) CreateCreditCardTable() error {
	query := `create table if not exists "creditcard" (
				id UUID NOT NULL, 
				name varchar (60) NOT NULL, 
				archived boolean NOT NULL DEFAULT false, 
				closing_day int NOT NULL, 
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				CONSTRAINT uc_name UNIQUE(name),
				PRIMARY KEY ("id")
	)`
	_, err := s.db.Query(query)
	return err
}

func (s *PostgresStore) CreateCreditCard(creditCard *types.CreditCard) error {
	query := `insert into "creditcard" 
	(id, name, closing_day, created_at, updated_at)
	values ($1, $2, $3, $4, $5)`

	_, err := s.db.Query(query, creditCard.ID, creditCard.Name, creditCard.ClosingDay, creditCard.CreatedAt, creditCard.UpdatedAt)
	return err
}

func (s *PostgresStore) GetCreditCardByID(id uuid.UUID) (*types.CreditCard, error) {
	query := "select * from creditcard where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoCreditCard(rows)
	}

	return nil, fmt.Errorf("credit card %v not found", id)
}
func (s *PostgresStore) GetCreditCardByName(name string) (*types.CreditCard, error) {
	query := "select * from creditcard where name = $1"
	rows, err := s.db.Query(query, name)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoCreditCard(rows)
	}

	return nil, fmt.Errorf("credit card %v not found", name)
}

func (s *PostgresStore) GetCreditCard() ([]*types.CreditCard, error) {
	rows, err := s.db.Query("select * from creditcard")
	if err != nil {
		return nil, err
	}

	cards := []*types.CreditCard{}

	for rows.Next() {
		card, err := scanIntoCreditCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func scanIntoCreditCard(rows *sql.Rows) (*types.CreditCard, error) {
	card := &types.CreditCard{}
	err := rows.Scan(
		&card.ID,
		&card.Name,
		&card.ClosingDay,
		&card.Archived,
		&card.CreatedAt,
		&card.UpdatedAt)

	return card, err
}

func (s *PostgresStore) ArchiveCreditCard(creditCardID uuid.UUID) error {
	query := `UPDATE creditcard SET archived = $1 where id = $2`
	_, err := s.db.Query(query, true, creditCardID)

	return err
}

// Category
func (s *PostgresStore) CreateCategoryTable() error {
	query := `create table if not exists "category" (
				id UUID NOT NULL, 
				description varchar (60) NOT NULL, 
				archived boolean NOT NULL DEFAULT false, 
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				CONSTRAINT uc_description UNIQUE(description),
				PRIMARY KEY ("id")
	)`
	_, err := s.db.Query(query)
	return err
}

func (s *PostgresStore) CreateCategory(category *types.Category) error {
	query := `insert into "category" 
	(id, description, created_at, updated_at)
	values ($1, $2, $3, $4)`

	_, err := s.db.Query(query, category.ID, category.Description, category.CreatedAt, category.UpdatedAt)
	return err
}

func (s *PostgresStore) GetCategoryByDescription(description string) (*types.Category, error) {
	query := "select * from category where description = $1"
	rows, err := s.db.Query(query, description)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoCategory(rows)
	}

	return nil, fmt.Errorf("category %v not found", description)
}

func (s *PostgresStore) GetCategoryByID(id uuid.UUID) (*types.Category, error) {
	query := "select * from category where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoCategory(rows)
	}

	return nil, fmt.Errorf("category %v not found", id)
}

func (s *PostgresStore) GetCategory() ([]*types.Category, error) {
	rows, err := s.db.Query("select * from category")
	if err != nil {
		return nil, err
	}

	categories := []*types.Category{}

	for rows.Next() {
		category, err := scanIntoCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func scanIntoCategory(rows *sql.Rows) (*types.Category, error) {
	category := &types.Category{}
	err := rows.Scan(
		&category.ID,
		&category.Description,
		&category.Archived,
		&category.CreatedAt,
		&category.UpdatedAt)

	return category, err
}

func (s *PostgresStore) ArchiveCategory(categoryID uuid.UUID) error {
	query := `UPDATE category SET archived = $1 where id = $2`
	_, err := s.db.Query(query, true, categoryID)
	return err
}

// Session
func (s *PostgresStore) CreateSessionTable() error {
	query := `create table if not exists "session" (
				id UUID NOT NULL, 
				user_id UUID NOT NULL, 
				expires_at timestamptz NOT NULL,
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				PRIMARY KEY ("id"),
				CONSTRAINT "session_users" FOREIGN KEY ("user_id") REFERENCES "user" ("id")
				)`
	_, err := s.db.Query(query)
	return err
}

func (s *PostgresStore) CreateSession(session *types.Session) error {
	query := `insert into "session" 
	(id, user_id, expires_at, created_at, updated_at)
	values ($1, $2, $3, $4, $5)`

	_, err := s.db.Query(query, session.ID, session.UserId, session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	return err
}

func (s *PostgresStore) DeleteSession(sessionID uuid.UUID) error {
	query := `DELETE from session where id = $1`
	_, err := s.db.Query(query, sessionID)
	return err
}

func (s *PostgresStore) UpdateSession(sessionID uuid.UUID, expiresAt time.Time) error {
	query := `UPDATE session SET expires_at = $1 where id = $2`
	_, err := s.db.Query(query, expiresAt, sessionID)
	return err
}

func (s *PostgresStore) GetSessionByID(id uuid.UUID) (*types.Session, error) {
	query := "select * from session where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	session := &types.Session{}
	for rows.Next() {
		err := rows.Scan(
			&session.ID,
			&session.UserId,
			&session.ExpiresAt,
			&session.CreatedAt,
			&session.UpdatedAt)
		return session, err
	}
	return nil, fmt.Errorf("session %v not found", id)
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
	return err
}

func (s *PostgresStore) DeleteUser(id uuid.UUID) error {
	query := `delete from "user" where id = $1`

	_, err := s.db.Query(query, id)
	return err
}

func (s *PostgresStore) GetUserByEmail(email string) (*types.User, error) {
	query := `select * from "user" where email = $1`
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
