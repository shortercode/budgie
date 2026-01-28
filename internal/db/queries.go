package db

import (
	"database/sql"
	"time"
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
