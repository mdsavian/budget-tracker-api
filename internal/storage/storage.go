package storage

import (
	"database/sql"
	"fmt"
	"log"
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
	if err := s.createAccountTable(); err != nil {
		return err
	}

	if err := s.createUserTable(); err != nil {
		return err
	}

	if err := s.createSessionTable(); err != nil {
		return err
	}

	if err := s.CreateCategoryTable(); err != nil {
		return err
	}

	if err := s.createCreditCardTable(); err != nil {
		return err
	}

	if err := s.createTransactionTable(); err != nil {
		return err
	}

	return nil
}

// Transaction
func (s *PostgresStore) createTransactionTable() error {
	query := `create table if not exists "transaction" (
		id UUID NOT NULL, 
		account_id UUID NOT NULL,
		creditcard_id UUID NULL,
		category_id UUID NOT NULL,
		transaction_type varchar (100) NOT NULL,
		date timestamptz NOT NULL, 
		description varchar(500) NOT NULL,
		amount numeric NOT NULL,
		paid boolean NOT NULL DEFAULT false,
		cost_of_living boolean NOT NULL DEFAULT false,		
		created_at timestamptz NOT NULL, 
		updated_at timestamptz NOT NULL, 

		PRIMARY KEY ("id"),
		CONSTRAINT "transaction_account" FOREIGN KEY ("account_id") REFERENCES "account" ("id"),
		CONSTRAINT "transaction_card" FOREIGN KEY ("creditcard_id") REFERENCES "creditcard" ("id"),
		CONSTRAINT "transaction_category" FOREIGN KEY ("category_id") REFERENCES "category" ("id")
	)`
	_, err := s.db.Query(query)
	return err
}

func (s *PostgresStore) CreateTransaction(transaction *types.Transaction) error {
	query := `insert into "transaction" 
	(id, account_id, creditcard_id, category_id, transaction_type, date, description, 
		amount, paid, cost_of_living, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := s.db.Query(query,
		transaction.ID,
		transaction.AccountID,
		transaction.CreditCardID,
		transaction.CategoryID,
		transaction.TransactionType,
		transaction.Date,
		transaction.Description,
		transaction.Amount,
		transaction.Paid,
		transaction.CostOfLiving,
		transaction.CreatedAt,
		transaction.UpdatedAt)

	return err
}

func (s *PostgresStore) GetTransaction() ([]*types.Transaction, error) {
	rows, err := s.db.Query("select * from transaction")
	if err != nil {
		return nil, err
	}

	transactions := []*types.Transaction{}

	for rows.Next() {
		transaction, err := scanIntoTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func scanIntoTransaction(rows *sql.Rows) (*types.Transaction, error) {
	transaction := &types.Transaction{}
	err := rows.Scan(
		&transaction.ID,
		&transaction.AccountID,
		&transaction.CreditCardID,
		&transaction.CategoryID,
		&transaction.TransactionType,
		&transaction.Date,
		&transaction.Description,
		&transaction.Amount,
		&transaction.Paid,
		&transaction.CostOfLiving,
		&transaction.CreatedAt,
		&transaction.UpdatedAt)

	return transaction, err
}

// CreditCard
func (s *PostgresStore) createCreditCardTable() error {
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
	row := s.db.QueryRow(query, description)

	category := &types.Category{}
	err := row.Scan(
		&category.ID,
		&category.Description,
		&category.Archived,
		&category.CreatedAt,
		&category.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return category, nil
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
func (s *PostgresStore) createSessionTable() error {
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
func (s *PostgresStore) createUserTable() error {
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
func (s *PostgresStore) createAccountTable() error {
	query := `create table if not exists account (
				id UUID primary key NOT NULL, 
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				balance numeric NOT NULL DEFAULT 0, 
				name varchar (200) NOT NULL, 
				account_type varchar (50) NOT NULL,
				CONSTRAINT "uq_name_type" UNIQUE(name, account_type)
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

func (s *PostgresStore) GetUniqueAccount(name string, accountType types.AccountType) (*types.Account, error) {
	new := s.db.QueryRow(`select * from account where name= $1 and account_type = $2`, name, accountType.String())

	account := &types.Account{}
	err := new.Scan(
		&account.ID,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Balance,
		&account.Name,
		&account.AccountType)

	if err != nil {
		return nil, err
	}

	return account, nil
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
	log.Println("errorrrr", err)
	return account, err
}
