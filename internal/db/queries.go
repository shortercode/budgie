package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"budgie/internal/model"
)

// AccountBalance holds an account's name, type, and computed balance.
type AccountBalance struct {
	ID      int64
	Name    string
	Type    string
	Balance int64
}

// RecentTx holds a single transaction joined with its account name.
type RecentTx struct {
	Date        time.Time
	Account     string
	Description string
	Amount      int64
}

// TransactionRow holds a single transaction joined with its account name.
type TransactionRow struct {
	ID          int64
	AccountID   int64
	Date        time.Time
	Account     string
	Description string
	Amount      int64
	Category    string
}

// MonthlySummary holds income and expense totals for a given month.
type MonthlySummary struct {
	Year     int
	Month    time.Month
	Income   int64
	Expenses int64
}

var validAccountTypes = map[model.AccountType]bool{
	model.AccountTypeCurrent: true,
	model.AccountTypeSavings: true,
	model.AccountTypeCredit:  true,
	model.AccountTypeCash:    true,
}

// InsertAccount creates a new account and returns its ID.
func InsertAccount(db *sql.DB, name string, accountType model.AccountType) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("account name must not be empty")
	}
	if !validAccountTypes[accountType] {
		return 0, fmt.Errorf("invalid account type: %q", accountType)
	}

	result, err := db.Exec(
		`INSERT INTO accounts (name, type) VALUES (?, ?)`,
		name, string(accountType),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, fmt.Errorf("an account named %q already exists", name)
		}
		return 0, fmt.Errorf("inserting account: %w", err)
	}
	return result.LastInsertId()
}

// ErrAccountHasTransactions is returned when attempting to delete an account
// that still has transactions.
var ErrAccountHasTransactions = errors.New("account still has transactions")

// DeleteAccount removes an account by ID. It returns an error if the account
// has any transactions or if the account does not exist.
func DeleteAccount(db *sql.DB, accountID int64) error {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE account_id = ?`, accountID).Scan(&count)
	if err != nil {
		return fmt.Errorf("checking transactions: %w", err)
	}
	if count > 0 {
		return ErrAccountHasTransactions
	}

	result, err := db.Exec(`DELETE FROM accounts WHERE id = ?`, accountID)
	if err != nil {
		return fmt.Errorf("deleting account: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("account %d not found", accountID)
	}
	return nil
}

// ListAccounts returns all accounts ordered by name.
func ListAccounts(db *sql.DB) ([]model.Account, error) {
	rows, err := db.Query(`SELECT id, name, type FROM accounts ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	defer rows.Close()

	var out []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type); err != nil {
			return nil, fmt.Errorf("scanning account: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// InsertTransaction creates a new transaction and returns its ID.
func InsertTransaction(db *sql.DB, accountID int64, date time.Time, description string, amount int64, category string) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO transactions (account_id, date, description, amount, category) VALUES (?, ?, ?, ?, ?)`,
		accountID, date.Format("2006-01-02"), description, amount, category,
	)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return 0, fmt.Errorf("account does not exist")
		}
		return 0, fmt.Errorf("inserting transaction: %w", err)
	}
	return result.LastInsertId()
}

// DeleteTransaction removes a transaction by ID. Returns an error if it doesn't exist.
func DeleteTransaction(db *sql.DB, transactionID int64) error {
	result, err := db.Exec(`DELETE FROM transactions WHERE id = ?`, transactionID)
	if err != nil {
		return fmt.Errorf("deleting transaction: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("transaction %d not found", transactionID)
	}
	return nil
}

// UpdateTransaction updates an existing transaction by ID.
func UpdateTransaction(db *sql.DB, transactionID, accountID int64, date time.Time, description string, amount int64, category string) error {
	result, err := db.Exec(
		`UPDATE transactions SET account_id=?, date=?, description=?, amount=?, category=? WHERE id=?`,
		accountID, date.Format("2006-01-02"), description, amount, category, transactionID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return fmt.Errorf("account does not exist")
		}
		return fmt.Errorf("updating transaction: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("transaction %d not found", transactionID)
	}
	return nil
}

// AccountBalances returns every account with its computed balance.
func AccountBalances(db *sql.DB) ([]AccountBalance, error) {
	rows, err := db.Query(`
		SELECT a.id, a.name, a.type, COALESCE(SUM(t.amount), 0)
		FROM accounts a
		LEFT JOIN transactions t ON t.account_id = a.id
		GROUP BY a.id
		ORDER BY a.name
	`)
	if err != nil {
		return nil, fmt.Errorf("querying account balances: %w", err)
	}
	defer rows.Close()

	var out []AccountBalance
	for rows.Next() {
		var ab AccountBalance
		if err := rows.Scan(&ab.ID, &ab.Name, &ab.Type, &ab.Balance); err != nil {
			return nil, fmt.Errorf("scanning account balance: %w", err)
		}
		out = append(out, ab)
	}
	return out, rows.Err()
}

// RecentTransactions returns the last `limit` transactions with account names.
func RecentTransactions(db *sql.DB, limit int) ([]RecentTx, error) {
	rows, err := db.Query(`
		SELECT t.date, a.name, t.description, t.amount
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		ORDER BY t.date DESC, t.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent transactions: %w", err)
	}
	defer rows.Close()

	var out []RecentTx
	for rows.Next() {
		var tx RecentTx
		var dateStr string
		if err := rows.Scan(&dateStr, &tx.Account, &tx.Description, &tx.Amount); err != nil {
			return nil, fmt.Errorf("scanning transaction: %w", err)
		}
		tx.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing transaction date: %w", err)
		}
		out = append(out, tx)
	}
	return out, rows.Err()
}

// AllTransactions returns a page of transactions with account names, ordered
// by date descending then ID descending. It also returns the total count.
func AllTransactions(db *sql.DB, offset, limit int) ([]TransactionRow, int, error) {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting transactions: %w", err)
	}

	rows, err := db.Query(`
		SELECT t.id, t.account_id, t.date, a.name, t.description, t.amount, t.category
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		ORDER BY t.date DESC, t.id DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying transactions: %w", err)
	}
	defer rows.Close()

	var out []TransactionRow
	for rows.Next() {
		var tx TransactionRow
		var dateStr string
		if err := rows.Scan(&tx.ID, &tx.AccountID, &dateStr, &tx.Account, &tx.Description, &tx.Amount, &tx.Category); err != nil {
			return nil, 0, fmt.Errorf("scanning transaction: %w", err)
		}
		tx.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, 0, fmt.Errorf("parsing transaction date: %w", err)
		}
		out = append(out, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating transactions: %w", err)
	}
	return out, total, nil
}

// ImportTransaction holds the data needed to bulk-insert an imported transaction.
type ImportTransaction struct {
	AccountID     int64
	Date          time.Time
	Description   string
	Amount        int64
	Category      string
	ExternalID    string
	Notes         string
	Type          string
	LocalAmount   *int64
	LocalCurrency string
}

// ExistingExternalIDs returns a set of all non-NULL external_id values.
func ExistingExternalIDs(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT external_id FROM transactions WHERE external_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("querying external IDs: %w", err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning external ID: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// BulkInsertTransactions inserts multiple transactions in a single DB transaction.
// It returns the number of rows inserted.
func BulkInsertTransactions(db *sql.DB, txns []ImportTransaction) (int, error) {
	if len(txns) == 0 {
		return 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO transactions (account_id, date, description, amount, category, external_id, notes, type, local_amount, local_currency)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing insert: %w", err)
	}
	defer stmt.Close()

	for _, t := range txns {
		_, err := stmt.Exec(
			t.AccountID,
			t.Date.Format("2006-01-02"),
			t.Description,
			t.Amount,
			t.Category,
			t.ExternalID,
			t.Notes,
			t.Type,
			t.LocalAmount,
			t.LocalCurrency,
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return 0, fmt.Errorf("duplicate transaction ID %q already exists", t.ExternalID)
			}
			if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
				return 0, fmt.Errorf("account does not exist")
			}
			return 0, fmt.Errorf("inserting transaction: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing transaction: %w", err)
	}
	return len(txns), nil
}

// MonthSummary returns income and expense totals for the given year/month.
func MonthSummary(db *sql.DB, year int, month time.Month) (MonthlySummary, error) {
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	ms := MonthlySummary{Year: year, Month: month}

	err := db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END), 0)
		FROM transactions
		WHERE date >= ? AND date < ?
	`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&ms.Income, &ms.Expenses)
	if err != nil {
		return ms, fmt.Errorf("querying month summary: %w", err)
	}

	return ms, nil
}
