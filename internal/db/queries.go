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
		return 0, errors.New("account name must not be empty")
	}
	if !validAccountTypes[accountType] {
		return 0, fmt.Errorf("invalid account type: %q", accountType)
	}

	result, err := db.Exec(
		`INSERT INTO accounts (name, type) VALUES (?, ?)`,
		name, string(accountType),
	)
	if err != nil {
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
		return nil, err
	}
	defer rows.Close()

	var out []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
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
		return nil, err
	}
	defer rows.Close()

	var out []AccountBalance
	for rows.Next() {
		var ab AccountBalance
		if err := rows.Scan(&ab.ID, &ab.Name, &ab.Type, &ab.Balance); err != nil {
			return nil, err
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
		return nil, err
	}
	defer rows.Close()

	var out []RecentTx
	for rows.Next() {
		var tx RecentTx
		var dateStr string
		if err := rows.Scan(&dateStr, &tx.Account, &tx.Description, &tx.Amount); err != nil {
			return nil, err
		}
		tx.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, err
		}
		out = append(out, tx)
	}
	return out, rows.Err()
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

	return ms, err
}
