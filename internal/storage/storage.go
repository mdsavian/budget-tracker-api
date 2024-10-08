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

	if err := s.createRecurringTransactionTable(); err != nil {
		return err
	}

	if err := s.createTransactionTable(); err != nil {
		return err
	}

	return nil
}

// Recurring transaction
func (s *PostgresStore) createRecurringTransactionTable() error {
	query := `create table if not exists "recurring_transaction" (
		id UUID NOT NULL, 
		account_id UUID NOT NULL,
		creditcard_id UUID NULL,
		category_id UUID NOT NULL,

		transaction_type varchar (100) NOT NULL,
		day numeric NOT NULL, 
		description varchar(200) NOT NULL,
		amount numeric NOT NULL,
		archived boolean NOT NULL DEFAULT false, 

		created_at timestamptz NOT NULL, 
		updated_at timestamptz NOT NULL, 

		PRIMARY KEY ("id"),
		CONSTRAINT "recurring_transaction_account" FOREIGN KEY ("account_id") REFERENCES "account" ("id"),
		CONSTRAINT "recurring_transaction_card" FOREIGN KEY ("creditcard_id") REFERENCES "credit_card" ("id"),
		CONSTRAINT "recurring_transaction_category" FOREIGN KEY ("category_id") REFERENCES "category" ("id")
	)`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
func (s *PostgresStore) CreateRecurringTransaction(recurringTransaction *types.RecurringTransaction) error {
	query := `insert into "recurring_transaction" 
		(id, account_id, creditcard_id, category_id, transaction_type, day, description, 
			amount, archived, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	conn, err := s.db.Query(query,
		recurringTransaction.ID,
		recurringTransaction.AccountID,
		recurringTransaction.CreditCardID,
		recurringTransaction.CategoryID,
		recurringTransaction.TransactionType,
		recurringTransaction.Day,
		recurringTransaction.Description,
		recurringTransaction.Amount,
		recurringTransaction.Archived,
		recurringTransaction.CreatedAt,
		recurringTransaction.UpdatedAt)
	if err != nil {
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) ArchiveRecurringTransaction(recurringTransactionID uuid.UUID) error {
	query := `UPDATE recurring_transaction SET archived = $1 where id = $2`

	conn, err := s.db.Query(query, true, recurringTransactionID)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) UpdateRecurringTransaction(recurringTransactionID uuid.UUID, update *types.RecurringTransaction) error {
	query := `UPDATE recurring_transaction SET 
		account_id = COALESCE($1, account_id),
		creditcard_id = $2,
		category_id = COALESCE($3, category_id),
		day = COALESCE($4, day),
		description = COALESCE($5, description),
		amount = COALESCE($6, amount),
		updated_at = $7
		WHERE id = $8`

	_, err := s.db.Exec(query,
		update.AccountID,
		update.CreditCardID,
		update.CategoryID,
		update.Day,
		update.Description,
		update.Amount,
		time.Now().UTC(),
		recurringTransactionID)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) GetRecurringTransactionByID(id uuid.UUID) (*types.RecurringTransaction, error) {
	query := "select * from recurring_transaction where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		defer rows.Close()
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		return scanIntoRecurringTransaction(rows)
	}

	return nil, fmt.Errorf("recurring transaction %v not found", id)
}

func scanIntoRecurringTransaction(rows *sql.Rows) (*types.RecurringTransaction, error) {
	recurringTransaction := &types.RecurringTransaction{}
	err := rows.Scan(
		&recurringTransaction.ID,
		&recurringTransaction.AccountID,
		&recurringTransaction.CreditCardID,
		&recurringTransaction.CategoryID,
		&recurringTransaction.TransactionType,
		&recurringTransaction.Day,
		&recurringTransaction.Description,
		&recurringTransaction.Amount,
		&recurringTransaction.Archived,
		&recurringTransaction.CreatedAt,
		&recurringTransaction.UpdatedAt)

	return recurringTransaction, err

}

func (s *PostgresStore) createTransactionTable() error {
	query := `create table if not exists "transaction" (
		id UUID NOT NULL, 
		account_id UUID NOT NULL,
		creditcard_id UUID NULL,
		category_id UUID NOT NULL,
		recurring_transaction_id UUID NULL,

		transaction_type varchar (100) NOT NULL,
		"date" date NOT NULL, 
		description varchar(200) NOT NULL,
		amount numeric NOT NULL,
		fulfilled boolean NOT NULL DEFAULT false,
		created_at timestamptz NOT NULL, 
		updated_at timestamptz NOT NULL, 

		PRIMARY KEY ("id"),
		CONSTRAINT "transaction_account" FOREIGN KEY ("account_id") REFERENCES "account" ("id"),
		CONSTRAINT "transaction_card" FOREIGN KEY ("creditcard_id") REFERENCES "credit_card" ("id"),
		CONSTRAINT "transaction_category" FOREIGN KEY ("category_id") REFERENCES "category" ("id"),
		CONSTRAINT "transaction_recurring" FOREIGN KEY ("recurring_transaction_id") REFERENCES "recurring_transaction" ("id")
	);
		ALTER TABLE "transaction" ADD COLUMN IF NOT EXISTS "effectuated_date" date;
		ALTER TABLE "transaction" ADD COLUMN IF NOT EXISTS "archived" boolean NOT NULL DEFAULT false;`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) CreateTransaction(transaction *types.Transaction) error {
	query := `insert into "transaction" 
	(id, account_id, creditcard_id, category_id, recurring_transaction_id, transaction_type, date,effectuated_date, description, 
		amount, fulfilled, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	conn, err := s.db.Query(query,
		transaction.ID,
		transaction.AccountID,
		transaction.CreditCardID,
		transaction.CategoryID,
		transaction.RecurringTransactionID,
		transaction.TransactionType,
		transaction.Date,
		transaction.EffectuatedDate,
		transaction.Description,
		transaction.Amount,
		transaction.Fulfilled,
		time.Now(),
		time.Now())
	if err != nil {
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) DeleteTransaction(transacionID uuid.UUID) error {
	query := `UPDATE "transaction" SET archived = true, updated_at = $1 WHERE id = $2`
	_, err := s.db.Exec(query, time.Now().UTC(), transacionID)
	return err
}

func (s *PostgresStore) FulfillTransaction(transactionID uuid.UUID) error {
	query := `UPDATE "transaction" 
		SET fulfilled = $1,
		effectuated_date = $3,
		updated_at = $2 
		where id = $4`
	_, err := s.db.Exec(query, true, time.Now().UTC(), time.Now().UTC(), transactionID)
	return err
}

func (s *PostgresStore) UpdateTransaction(transactionID uuid.UUID, update *types.Transaction) error {
	query := `UPDATE "transaction" SET 
		account_id = COALESCE($1, account_id),
		creditcard_id = $2,
		category_id = COALESCE($3, category_id),
		effectuated_date = COALESCE($4, date),
		description = COALESCE($5, description),
		amount = COALESCE($6, amount),
		fulfilled = COALESCE($7, fulfilled),
		updated_at = $8
		WHERE id = $9`

	conn, err := s.db.Query(query,
		update.AccountID,
		update.CreditCardID,
		update.CategoryID,
		update.Date,
		update.Description,
		update.Amount,
		update.Fulfilled,
		time.Now().UTC(),
		transactionID)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) GetTransactionByID(id uuid.UUID) (*types.Transaction, error) {
	query := "select * from transaction where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		defer rows.Close()
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		return scanIntoTransaction(rows)
	}

	return nil, fmt.Errorf("transaction %v not found", id)
}

func (s *PostgresStore) GetTransactionsWithRecurringByDate(startDate, endDate time.Time) ([]*types.TransactionView, error) {
	query := `
	WITH RECURRING_DATES AS (
		SELECT 
			id AS recurring_transaction_id,
			account_id,
			creditcard_id,
			category_id,
			transaction_type,
			description,
			amount,
			generate_series(
				DATE_TRUNC('month', $1::date) + (day - 1) * INTERVAL '1 day',
				DATE_TRUNC('month', $2::date) + (day - 1) * INTERVAL '1 day',
				'1 month'
			)::date AS occurrence_date
		FROM 
			recurring_transaction
		WHERE 
			archived = false 			
	)
	SELECT 
		t.id, 
		t.account_id, 
		a."name" AS Account,
		t.creditcard_id,
		c."name" AS CreditCard,
		t.category_id,
		c2.description AS Category,
		t.recurring_transaction_id,
		t.transaction_type,
		t.date, 
		t.effectuated_date,
		t.description, 
		t.amount, 
		t.fulfilled
	FROM 
		transaction t
	LEFT JOIN 
		credit_card c ON c.id = t.creditcard_id 
	LEFT JOIN 
		category c2 ON c2.id = t.category_id 
	LEFT JOIN 
		account a ON a.id = t.account_id
	WHERE 
		((t.effectuated_date IS NOT NULL AND t.effectuated_date BETWEEN $1 AND $2)
		OR (t.date BETWEEN $1 AND $2))
		AND t.archived = false
	UNION ALL
	SELECT 
		NULL AS id,
		r.account_id, 
		a."name" AS Account,
		r.creditcard_id,
		c."name" AS CreditCard,
		r.category_id,
		c2.description AS Category,
		r.recurring_transaction_id,
		r.transaction_type,
		r.occurrence_date AS date,
		NULL as effectuated_date,
		r.description, 
		r.amount, 
		false AS fulfilled
	FROM 
		RECURRING_DATES r
	LEFT JOIN 
		transaction t 
	ON 
		t.date = r.occurrence_date 
		AND t.recurring_transaction_id = r.recurring_transaction_id
		AND t.archived = false
	LEFT JOIN 
		credit_card c ON c.id = r.creditcard_id 
	LEFT JOIN 
		category c2 ON c2.id = r.category_id 
	LEFT JOIN 
		account a ON a.id = r.account_id
	WHERE 
		t.id IS NULL
	ORDER BY 
		date desc;`

	rows, err := s.db.Query(query, startDate, endDate)
	if err != nil {
		defer rows.Close()
		return nil, err
	}
	defer rows.Close()

	transactions := []*types.TransactionView{}

	for rows.Next() {
		transaction, err := scanIntoTransactionView(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func scanIntoTransactionView(rows *sql.Rows) (*types.TransactionView, error) {
	transaction := &types.TransactionView{}
	err := rows.Scan(
		&transaction.ID,
		&transaction.AccountID,
		&transaction.Account,
		&transaction.CreditCardID,
		&transaction.CreditCard,
		&transaction.CategoryID,
		&transaction.Category,
		&transaction.RecurringTransactionID,
		&transaction.TransactionType,
		&transaction.Date,
		&transaction.EffectuatedDate,
		&transaction.Description,
		&transaction.Amount,
		&transaction.Fulfilled,
	)
	return transaction, err
}

func scanIntoTransaction(rows *sql.Rows) (*types.Transaction, error) {
	transaction := &types.Transaction{}
	err := rows.Scan(
		&transaction.ID,
		&transaction.AccountID,
		&transaction.CreditCardID,
		&transaction.CategoryID,
		&transaction.RecurringTransactionID,
		&transaction.TransactionType,
		&transaction.Date,
		&transaction.Description,
		&transaction.Amount,
		&transaction.Fulfilled,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
		&transaction.EffectuatedDate,
		&transaction.Archived)

	return transaction, err
}

// CreditCard
func (s *PostgresStore) createCreditCardTable() error {
	query := `create table if not exists "credit_card" (
				id UUID NOT NULL, 
				name varchar (60) NOT NULL, 
				archived boolean NOT NULL DEFAULT false, 
				due_day int NOT NULL, 
				closing_day int NOT NULL, 
				created_at timestamptz NOT NULL, 
				updated_at timestamptz NOT NULL, 
				CONSTRAINT uc_name UNIQUE(name),
				PRIMARY KEY ("id")
	)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateCreditCard(creditCard *types.CreditCard) error {
	query := `insert into "credit_card" 
	(id, name, due_day, closing_day, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	conn, err := s.db.Query(query, creditCard.ID, creditCard.Name, creditCard.DueDay, creditCard.ClosingDay, creditCard.CreatedAt, creditCard.UpdatedAt)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) GetCreditCardByID(id uuid.UUID) (*types.CreditCard, error) {
	query := "select * from credit_card where id = $1"
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
	query := "select * from credit_card where name = $1"
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
	rows, err := s.db.Query("select * from credit_card c order by c.name")
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
		&card.Archived,
		&card.DueDay,
		&card.ClosingDay,
		&card.CreatedAt,
		&card.UpdatedAt)

	return card, err
}

func (s *PostgresStore) ArchiveCreditCard(creditCardID uuid.UUID) error {
	query := `UPDATE credit_card SET archived = $1 where id = $2`
	conn, err := s.db.Query(query, true, creditCardID)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
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
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) CreateCategory(category *types.Category) error {
	query := `insert into "category" 
	(id, description, created_at, updated_at)
	values ($1, $2, $3, $4)`

	conn, err := s.db.Query(query, category.ID, category.Description, category.CreatedAt, category.UpdatedAt)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
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
	rows, err := s.db.Query("select * from category c order by c.description")
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
	conn, err := s.db.Query(query, true, categoryID)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
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
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) CreateSession(session *types.Session) error {
	query := `insert into "session" 
	(id, user_id, expires_at, created_at, updated_at)
	values ($1, $2, $3, $4, $5)`

	conn, err := s.db.Query(query, session.ID, session.UserId, session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) DeleteSession(sessionID uuid.UUID) error {
	query := `DELETE from session where id = $1`
	conn, err := s.db.Query(query, sessionID)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) UpdateSession(sessionID uuid.UUID, expiresAt time.Time) error {
	query := `UPDATE session SET expires_at = $1 where id = $2`
	_, err := s.db.Exec(query, expiresAt, sessionID)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) GetSessionByID(id uuid.UUID) (*types.Session, error) {
	query := "select * from session where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		defer rows.Close()
		return nil, err
	}

	defer rows.Close()
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
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) CreateUser(user *types.User) error {
	query := `insert into "user" 
	(id, name, email, password, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	conn, err := s.db.Query(query, user.ID, user.Name, user.Email, user.EncryptedPassword, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) DeleteUser(id uuid.UUID) error {
	query := `delete from "user" where id = $1`

	conn, err := s.db.Query(query, id)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
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
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) UpdateAccountBalance(accountID uuid.UUID, amount float32, transactionType types.TransactionType) error {
	var balance float32
	var newBalance float32

	query := "select balance from account where id = $1"
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		defer rows.Close()
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&balance)
		if err != nil {
			return err
		}
	}

	if transactionType == types.TransactionTypeCredit {
		newBalance = balance + amount
	} else {
		newBalance = balance - amount
	}

	query = `update account set balance = $1 where id = $2`
	conn, err := s.db.Query(query, newBalance, accountID)
	if err != nil {
		defer conn.Close()
		return err
	}
	defer conn.Close()

	return nil
}

func (s *PostgresStore) CreateAccount(acc *types.Account) error {
	query := `insert into account 
	(id, name, account_type, balance, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6)`

	conn, err := s.db.Query(query, acc.ID, acc.Name, acc.AccountType, acc.Balance, acc.CreatedAt, acc.UpdatedAt)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil
}

func (s *PostgresStore) DeleteAccount(id uuid.UUID) error {
	query := "delete from account where id = $1"

	conn, err := s.db.Query(query, id)
	if err != nil {
		defer conn.Close()
		return err
	}

	defer conn.Close()
	return nil

}

func (s *PostgresStore) GetAccountByID(id uuid.UUID) (*types.Account, error) {
	query := "select * from account where id = $1"
	rows, err := s.db.Query(query, id)
	if err != nil {
		defer rows.Close()
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
	rows, err := s.db.Query("select * from account a order by a.name")
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
