package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"budgie/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

// testDB returns a migrated in-memory SQLite database.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	if err := migrate(db); err != nil {
		t.Fatalf("migrating test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// insertTestAccount is a helper that inserts an account directly via SQL.
func insertTestAccount(t *testing.T, db *sql.DB, name string, acctType string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO accounts (name, type) VALUES (?, ?)`, name, acctType)
	if err != nil {
		t.Fatalf("inserting test account: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

// insertTestTransaction is a helper that inserts a transaction directly via SQL.
func insertTestTransaction(t *testing.T, db *sql.DB, accountID int64, date string, desc string, amount int64) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO transactions (account_id, date, description, amount, category) VALUES (?, ?, ?, ?, '')`,
		accountID, date, desc, amount,
	)
	if err != nil {
		t.Fatalf("inserting test transaction: %v", err)
	}
}

// --- InsertAccount tests ---

func TestInsertAccount_Valid(t *testing.T) {
	db := testDB(t)
	id, err := InsertAccount(db, "Main Account", model.AccountTypeCurrent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestInsertAccount_EmptyName(t *testing.T) {
	db := testDB(t)
	_, err := InsertAccount(db, "", model.AccountTypeCurrent)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestInsertAccount_WhitespaceName(t *testing.T) {
	db := testDB(t)
	_, err := InsertAccount(db, "   ", model.AccountTypeSavings)
	if err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestInsertAccount_InvalidType(t *testing.T) {
	db := testDB(t)
	_, err := InsertAccount(db, "Test", model.AccountType("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid account type")
	}
}

func TestInsertAccount_DuplicateName(t *testing.T) {
	db := testDB(t)
	_, err := InsertAccount(db, "Savings", model.AccountTypeSavings)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}
	_, err = InsertAccount(db, "Savings", model.AccountTypeSavings)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

// --- DeleteAccount tests ---

func TestDeleteAccount_Valid(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "ToDelete", "current")
	if err := DeleteAccount(db, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAccount_HasTransactions(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "WithTx", "current")
	insertTestTransaction(t, db, id, "2026-01-15", "Salary", 150000)

	err := DeleteAccount(db, id)
	if !errors.Is(err, ErrAccountHasTransactions) {
		t.Fatalf("expected ErrAccountHasTransactions, got: %v", err)
	}
}

func TestDeleteAccount_NotFound(t *testing.T) {
	db := testDB(t)
	err := DeleteAccount(db, 9999)
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

// --- ListAccounts tests ---

func TestListAccounts_Empty(t *testing.T) {
	db := testDB(t)
	accounts, err := ListAccounts(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestListAccounts_OrderedByName(t *testing.T) {
	db := testDB(t)
	insertTestAccount(t, db, "Zebra", "cash")
	insertTestAccount(t, db, "Alpha", "current")
	insertTestAccount(t, db, "Middle", "savings")

	accounts, err := ListAccounts(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(accounts))
	}
	if accounts[0].Name != "Alpha" || accounts[1].Name != "Middle" || accounts[2].Name != "Zebra" {
		t.Fatalf("accounts not ordered by name: %v, %v, %v", accounts[0].Name, accounts[1].Name, accounts[2].Name)
	}
}

func TestListAccounts_CorrectFields(t *testing.T) {
	db := testDB(t)
	insertTestAccount(t, db, "Checking", "current")

	accounts, err := ListAccounts(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	a := accounts[0]
	if a.Name != "Checking" {
		t.Errorf("expected name 'Checking', got %q", a.Name)
	}
	if a.Type != model.AccountTypeCurrent {
		t.Errorf("expected type 'current', got %q", a.Type)
	}
	if a.ID <= 0 {
		t.Errorf("expected positive ID, got %d", a.ID)
	}
}

// --- InsertTransaction tests ---

func TestInsertTransaction_Valid(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	id, err := InsertTransaction(db, acctID, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "Salary", 150000, "income")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestInsertTransaction_EmptyDescriptionAndCategory(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	_, err := InsertTransaction(db, acctID, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "", 5000, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertTransaction_InvalidAccount(t *testing.T) {
	db := testDB(t)
	_, err := InsertTransaction(db, 9999, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "Test", 1000, "")
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestInsertTransaction_NegativeAmount(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	id, err := InsertTransaction(db, acctID, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "Rent", -80000, "housing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestInsertTransaction_StoresDateCorrectly(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	InsertTransaction(db, acctID, date, "Test", 1000, "")

	txs, err := RecentTransactions(db, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Date != date {
		t.Errorf("expected date %v, got %v", date, txs[0].Date)
	}
}

// --- DeleteTransaction tests ---

func TestDeleteTransaction_Valid(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, acctID, "2026-01-15", "ToDelete", 1000)

	// Get the transaction ID
	var txID int64
	db.QueryRow(`SELECT id FROM transactions LIMIT 1`).Scan(&txID)

	if err := DeleteTransaction(db, txID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	txs, _ := RecentTransactions(db, 10)
	if len(txs) != 0 {
		t.Errorf("expected 0 transactions after delete, got %d", len(txs))
	}
}

func TestDeleteTransaction_NotFound(t *testing.T) {
	db := testDB(t)
	err := DeleteTransaction(db, 9999)
	if err == nil {
		t.Fatal("expected error for non-existent transaction")
	}
}

// --- UpdateTransaction tests ---

func TestUpdateTransaction_Valid(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	acctID2 := insertTestAccount(t, db, "Savings", "savings")
	insertTestTransaction(t, db, acctID, "2026-01-15", "Salary", 150000)

	// Get the transaction ID
	var txID int64
	db.QueryRow(`SELECT id FROM transactions LIMIT 1`).Scan(&txID)

	newDate := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	err := UpdateTransaction(db, txID, acctID2, newDate, "Updated Salary", 200000, "income")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify via AllTransactions
	txs, _, err := AllTransactions(db, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}
	tx := txs[0]
	if tx.Account != "Savings" {
		t.Errorf("expected account 'Savings', got %q", tx.Account)
	}
	if tx.Description != "Updated Salary" {
		t.Errorf("expected description 'Updated Salary', got %q", tx.Description)
	}
	if tx.Amount != 200000 {
		t.Errorf("expected amount 200000, got %d", tx.Amount)
	}
	if tx.Category != "income" {
		t.Errorf("expected category 'income', got %q", tx.Category)
	}
	if tx.Date != newDate {
		t.Errorf("expected date %v, got %v", newDate, tx.Date)
	}
}

func TestUpdateTransaction_NotFound(t *testing.T) {
	db := testDB(t)
	err := UpdateTransaction(db, 9999, 1, time.Now(), "Test", 1000, "")
	if err == nil {
		t.Fatal("expected error for non-existent transaction")
	}
}

func TestUpdateTransaction_InvalidAccount(t *testing.T) {
	db := testDB(t)
	acctID := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, acctID, "2026-01-15", "Salary", 150000)

	var txID int64
	db.QueryRow(`SELECT id FROM transactions LIMIT 1`).Scan(&txID)

	err := UpdateTransaction(db, txID, 9999, time.Now(), "Test", 1000, "")
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

// --- AccountBalances tests ---

func TestAccountBalances_Empty(t *testing.T) {
	db := testDB(t)
	balances, err := AccountBalances(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 0 {
		t.Fatalf("expected 0 balances, got %d", len(balances))
	}
}

func TestAccountBalances_ZeroBalance(t *testing.T) {
	db := testDB(t)
	insertTestAccount(t, db, "Empty", "savings")

	balances, err := AccountBalances(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(balances))
	}
	if balances[0].Balance != 0 {
		t.Errorf("expected zero balance, got %d", balances[0].Balance)
	}
}

func TestAccountBalances_ComputesCorrectly(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-01-10", "Salary", 200000)
	insertTestTransaction(t, db, id, "2026-01-15", "Rent", -80000)
	insertTestTransaction(t, db, id, "2026-01-20", "Groceries", -5000)

	balances, err := AccountBalances(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(balances))
	}
	expected := int64(200000 - 80000 - 5000)
	if balances[0].Balance != expected {
		t.Errorf("expected balance %d, got %d", expected, balances[0].Balance)
	}
}

// --- RecentTransactions tests ---

func TestRecentTransactions_Empty(t *testing.T) {
	db := testDB(t)
	txs, err := RecentTransactions(db, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txs))
	}
}

func TestRecentTransactions_ReverseChronological(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-01-10", "First", 1000)
	insertTestTransaction(t, db, id, "2026-01-20", "Second", 2000)
	insertTestTransaction(t, db, id, "2026-01-15", "Middle", 3000)

	txs, err := RecentTransactions(db, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(txs))
	}
	if txs[0].Description != "Second" || txs[1].Description != "Middle" || txs[2].Description != "First" {
		t.Errorf("transactions not in reverse chronological order: %v, %v, %v",
			txs[0].Description, txs[1].Description, txs[2].Description)
	}
}

func TestRecentTransactions_RespectsLimit(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-01-10", "One", 1000)
	insertTestTransaction(t, db, id, "2026-01-11", "Two", 2000)
	insertTestTransaction(t, db, id, "2026-01-12", "Three", 3000)

	txs, err := RecentTransactions(db, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}
}

func TestRecentTransactions_IncludesAccountName(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Savings Pot", "savings")
	insertTestTransaction(t, db, id, "2026-01-10", "Interest", 500)

	txs, err := RecentTransactions(db, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Account != "Savings Pot" {
		t.Errorf("expected account 'Savings Pot', got %q", txs[0].Account)
	}
}

// --- AllTransactions tests ---

func TestAllTransactions_Empty(t *testing.T) {
	db := testDB(t)
	txs, total, err := AllTransactions(db, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txs))
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
}

func TestAllTransactions_ReverseChronological(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-01-10", "First", 1000)
	insertTestTransaction(t, db, id, "2026-01-20", "Second", 2000)
	insertTestTransaction(t, db, id, "2026-01-15", "Middle", 3000)

	txs, total, err := AllTransactions(db, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(txs) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(txs))
	}
	if txs[0].Description != "Second" || txs[1].Description != "Middle" || txs[2].Description != "First" {
		t.Errorf("transactions not in reverse chronological order: %v, %v, %v",
			txs[0].Description, txs[1].Description, txs[2].Description)
	}
}

func TestAllTransactions_Pagination(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-01-10", "One", 1000)
	insertTestTransaction(t, db, id, "2026-01-11", "Two", 2000)
	insertTestTransaction(t, db, id, "2026-01-12", "Three", 3000)
	insertTestTransaction(t, db, id, "2026-01-13", "Four", 4000)
	insertTestTransaction(t, db, id, "2026-01-14", "Five", 5000)

	// First page
	txs, total, err := AllTransactions(db, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}
	if txs[0].Description != "Five" || txs[1].Description != "Four" {
		t.Errorf("unexpected first page: %v, %v", txs[0].Description, txs[1].Description)
	}

	// Second page
	txs, total, err = AllTransactions(db, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}
	if txs[0].Description != "Three" || txs[1].Description != "Two" {
		t.Errorf("unexpected second page: %v, %v", txs[0].Description, txs[1].Description)
	}
}

func TestAllTransactions_IncludesAccountName(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Savings Pot", "savings")
	insertTestTransaction(t, db, id, "2026-01-10", "Interest", 500)

	txs, _, err := AllTransactions(db, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Account != "Savings Pot" {
		t.Errorf("expected account 'Savings Pot', got %q", txs[0].Account)
	}
}

func TestAllTransactions_IncludesCategory(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	_, err := db.Exec(
		`INSERT INTO transactions (account_id, date, description, amount, category) VALUES (?, ?, ?, ?, ?)`,
		id, "2026-01-10", "Salary", 200000, "income",
	)
	if err != nil {
		t.Fatalf("inserting test transaction: %v", err)
	}

	txs, _, err := AllTransactions(db, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Category != "income" {
		t.Errorf("expected category 'income', got %q", txs[0].Category)
	}
}

// --- MonthSummary tests ---

func TestMonthSummary_Empty(t *testing.T) {
	db := testDB(t)
	ms, err := MonthSummary(db, 2026, time.January)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.Income != 0 || ms.Expenses != 0 {
		t.Errorf("expected zeroes, got income=%d expenses=%d", ms.Income, ms.Expenses)
	}
}

func TestMonthSummary_SeparatesIncomeAndExpenses(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-02-05", "Salary", 300000)
	insertTestTransaction(t, db, id, "2026-02-10", "Rent", -100000)
	insertTestTransaction(t, db, id, "2026-02-15", "Food", -25000)

	ms, err := MonthSummary(db, 2026, time.February)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.Income != 300000 {
		t.Errorf("expected income 300000, got %d", ms.Income)
	}
	if ms.Expenses != -125000 {
		t.Errorf("expected expenses -125000, got %d", ms.Expenses)
	}
}

func TestMonthSummary_OnlyIncludesSpecifiedMonth(t *testing.T) {
	db := testDB(t)
	id := insertTestAccount(t, db, "Main", "current")
	insertTestTransaction(t, db, id, "2026-01-15", "Jan salary", 200000)
	insertTestTransaction(t, db, id, "2026-02-15", "Feb salary", 300000)
	insertTestTransaction(t, db, id, "2026-03-15", "Mar salary", 250000)

	ms, err := MonthSummary(db, 2026, time.February)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.Income != 300000 {
		t.Errorf("expected income 300000 (Feb only), got %d", ms.Income)
	}
}
