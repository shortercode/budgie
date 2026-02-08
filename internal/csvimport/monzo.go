package csvimport

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// MonzoTransaction represents a single parsed row from a Monzo CSV export.
type MonzoTransaction struct {
	ExternalID  string
	Date        time.Time
	Description string // from Monzo "Name" column
	Amount      int64  // pence, signed
	Category    string
	Type        string // e.g. "card_payment"
	Notes       string
	LocalAmount string // e.g. "-6.25 EUR" or "" for domestic
	Emoji       string
	SourceDesc  string // Monzo "Description" column
}

var requiredColumns = []string{
	"Transaction ID",
	"Date",
	"Name",
	"Emoji",
	"Description",
	"Amount",
	"Category",
	"Type",
	"Notes and #tags",
	"Currency",
	"Local amount",
	"Local currency",
}

// ParseMonzoCSV reads a Monzo CSV export and returns parsed transactions.
// Rows with empty Transaction ID are skipped.
func ParseMonzoCSV(path string) ([]MonzoTransaction, error) {
	path = expandTilde(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	header := records[0]
	colIndex := make(map[string]int, len(header))
	for i, col := range header {
		colIndex[col] = i
	}

	for _, req := range requiredColumns {
		if _, ok := colIndex[req]; !ok {
			return nil, fmt.Errorf("missing required column: %q", req)
		}
	}

	var out []MonzoTransaction
	for i, row := range records[1:] {
		lineNum := i + 2 // 1-indexed, skip header

		extID := row[colIndex["Transaction ID"]]
		if extID == "" {
			continue
		}

		dateStr := row[colIndex["Date"]]
		date, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid date %q", lineNum, dateStr)
		}

		amountStr := row[colIndex["Amount"]]
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid amount %q", lineNum, amountStr)
		}
		amount := int64(math.Round(amountFloat * 100))

		tx := MonzoTransaction{
			ExternalID:  extID,
			Date:        date,
			Description: row[colIndex["Name"]],
			Amount:      amount,
			Category:    row[colIndex["Category"]],
			Type:        row[colIndex["Type"]],
			Notes:       row[colIndex["Notes and #tags"]],
			Emoji:       row[colIndex["Emoji"]],
			SourceDesc:  row[colIndex["Description"]],
		}

		currency := row[colIndex["Currency"]]
		localCurrency := row[colIndex["Local currency"]]
		if localCurrency != "" && localCurrency != currency {
			localAmountStr := row[colIndex["Local amount"]]
			localFloat, err := strconv.ParseFloat(localAmountStr, 64)
			if err == nil {
				tx.LocalAmount = fmt.Sprintf("%.2f %s", localFloat, localCurrency)
			}
		}

		out = append(out, tx)
	}

	return out, nil
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}
