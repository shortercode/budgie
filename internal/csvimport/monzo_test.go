package csvimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "monzo.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp CSV: %v", err)
	}
	return path
}

const validHeader = "Transaction ID,Date,Time,Type,Name,Emoji,Category,Amount,Currency,Local amount,Local currency,Notes and #tags,Address,Receipt,Description,Category split,Money Out,Money In\n"

func TestParseMonzoCSV_ValidFile(t *testing.T) {
	csv := validHeader +
		"tx_001,07/02/2026,14:30:00,card_payment,Tesco,,groceries,-15.50,GBP,-15.50,GBP,weekly shop,,,,,-15.50,\n"

	path := writeTempCSV(t, csv)
	txns, err := ParseMonzoCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}

	tx := txns[0]
	if tx.ExternalID != "tx_001" {
		t.Errorf("expected external ID 'tx_001', got %q", tx.ExternalID)
	}
	if tx.Date.Day() != 7 || tx.Date.Month() != 2 || tx.Date.Year() != 2026 {
		t.Errorf("unexpected date: %v", tx.Date)
	}
	if tx.Description != "Tesco" {
		t.Errorf("expected description 'Tesco', got %q", tx.Description)
	}
	if tx.Amount != -1550 {
		t.Errorf("expected amount -1550, got %d", tx.Amount)
	}
	if tx.Category != "groceries" {
		t.Errorf("expected category 'groceries', got %q", tx.Category)
	}
	if tx.Type != "card_payment" {
		t.Errorf("expected type 'card_payment', got %q", tx.Type)
	}
	if tx.Notes != "weekly shop" {
		t.Errorf("expected notes 'weekly shop', got %q", tx.Notes)
	}
	if tx.LocalAmount != nil {
		t.Errorf("expected nil LocalAmount for GBP transaction, got %d", *tx.LocalAmount)
	}
	if tx.LocalCurrency != "" {
		t.Errorf("expected empty LocalCurrency for GBP transaction, got %q", tx.LocalCurrency)
	}
}

func TestParseMonzoCSV_FileNotFound(t *testing.T) {
	_, err := ParseMonzoCSV("/nonexistent/path/monzo.csv")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "could not open file") {
		t.Errorf("expected 'could not open file' error, got: %v", err)
	}
}

func TestParseMonzoCSV_InvalidHeader(t *testing.T) {
	csv := "ID,Date,Amount\n" +
		"1,07/02/2026,10.00\n"

	path := writeTempCSV(t, csv)
	_, err := ParseMonzoCSV(path)
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
	if !strings.Contains(err.Error(), "missing required column") {
		t.Errorf("expected 'missing required column' error, got: %v", err)
	}
}

func TestParseMonzoCSV_EmptyFile(t *testing.T) {
	path := writeTempCSV(t, validHeader)
	txns, err := ParseMonzoCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txns))
	}
}

func TestParseMonzoCSV_InvalidAmount(t *testing.T) {
	csv := validHeader +
		"tx_001,07/02/2026,14:30:00,card_payment,Tesco,,groceries,abc,GBP,,GBP,,,,,,,\n"

	path := writeTempCSV(t, csv)
	_, err := ParseMonzoCSV(path)
	if err == nil {
		t.Fatal("expected error for invalid amount")
	}
	if !strings.Contains(err.Error(), "invalid amount") {
		t.Errorf("expected 'invalid amount' error, got: %v", err)
	}
}

func TestParseMonzoCSV_ForeignCurrency(t *testing.T) {
	csv := validHeader +
		"tx_002,07/02/2026,14:30:00,card_payment,Cafe,,eating_out,-5.00,GBP,-6.25,EUR,,,,,,-5.00,\n"

	path := writeTempCSV(t, csv)
	txns, err := ParseMonzoCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}

	tx := txns[0]
	if tx.LocalAmount == nil {
		t.Fatal("expected non-nil LocalAmount for foreign currency transaction")
	}
	if *tx.LocalAmount != -625 {
		t.Errorf("expected local amount -625, got %d", *tx.LocalAmount)
	}
	if tx.LocalCurrency != "EUR" {
		t.Errorf("expected local currency 'EUR', got %q", tx.LocalCurrency)
	}
}

func TestParseMonzoCSV_SkipsEmptyExternalID(t *testing.T) {
	csv := validHeader +
		"tx_001,07/02/2026,14:30:00,card_payment,Tesco,,groceries,-15.50,GBP,-15.50,GBP,,,,,,-15.50,\n" +
		",07/02/2026,14:30:00,card_payment,Empty,,misc,-1.00,GBP,-1.00,GBP,,,,,,-1.00,\n"

	path := writeTempCSV(t, csv)
	txns, err := ParseMonzoCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction (skipping empty ID), got %d", len(txns))
	}
	if txns[0].ExternalID != "tx_001" {
		t.Errorf("expected external ID 'tx_001', got %q", txns[0].ExternalID)
	}
}
