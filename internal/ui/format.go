package ui

import (
	"fmt"
	"time"
)

// formatPence formats an integer amount in pence as a GBP string.
// e.g. 1250 → "£12.50", -4500 → "-£45.00"
func formatPence(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	pounds := amount / 100
	pence := amount % 100

	formatted := addThousandsSeparator(pounds)

	if negative {
		return fmt.Sprintf("-£%s.%02d", formatted, pence)
	}
	return fmt.Sprintf("£%s.%02d", formatted, pence)
}

// formatSignedPence formats with an explicit + or - prefix.
// e.g. 1250 → "+£12.50", -4500 → "-£45.00", 0 → "£0.00"
func formatSignedPence(amount int64) string {
	if amount > 0 {
		return "+" + formatPence(amount)
	}
	return formatPence(amount)
}

func addThousandsSeparator(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	// Work from the right, inserting commas every 3 digits.
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

// formatDate formats a time.Time as DD/MM/YYYY.
func formatDate(t time.Time) string {
	return t.Format("02/01/2006")
}
