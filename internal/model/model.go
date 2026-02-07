package model

import "time"

type Account struct {
	ID   int64
	Name string
	Type AccountType
}

type AccountType string

const (
	AccountTypeCurrent AccountType = "current"
	AccountTypeSavings  AccountType = "savings"
	AccountTypeCredit   AccountType = "credit"
	AccountTypeCash     AccountType = "cash"
)

type Transaction struct {
	ID          int64
	AccountID   int64
	Date        time.Time
	Description string
	Amount      int64 // stored in cents to avoid floating point issues
	Category    string
}
