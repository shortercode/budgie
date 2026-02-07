package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func Open() (*sql.DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}
	dbPath := filepath.Join(dir, "budgie.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return db, nil
}

func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "budgie"), nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL CHECK(type IN ('current','savings','credit','cash'))
		);

		CREATE TABLE IF NOT EXISTS transactions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id  INTEGER NOT NULL REFERENCES accounts(id),
			date        TEXT    NOT NULL,
			description TEXT    NOT NULL DEFAULT '',
			amount      INTEGER NOT NULL,
			category    TEXT    NOT NULL DEFAULT ''
		);

		CREATE INDEX IF NOT EXISTS idx_transactions_account
			ON transactions(account_id);
		CREATE INDEX IF NOT EXISTS idx_transactions_date
			ON transactions(date);
	`)
	if err != nil {
		return err
	}

	// Migrate accounts table from old "checking" CHECK constraint to "current".
	// CREATE TABLE IF NOT EXISTS won't update an existing table's constraints,
	// so we must detect the old schema and recreate the table.
	var tableSql string
	err = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='accounts'`).Scan(&tableSql)
	if err != nil {
		return err
	}
	if strings.Contains(tableSql, "'checking'") {
		_, err = db.Exec(`
			PRAGMA foreign_keys = OFF;

			CREATE TABLE accounts_new (
				id   INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				type TEXT NOT NULL CHECK(type IN ('current','savings','credit','cash'))
			);

			INSERT INTO accounts_new (id, name, type)
				SELECT id, name, CASE WHEN type = 'checking' THEN 'current' ELSE type END
				FROM accounts;

			DROP TABLE accounts;
			ALTER TABLE accounts_new RENAME TO accounts;

			PRAGMA foreign_keys = ON;
		`)
		if err != nil {
			return fmt.Errorf("migrating accounts table: %w", err)
		}
	}

	return nil
}
