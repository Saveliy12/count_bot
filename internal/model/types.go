package model

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

type Actor struct {
	TelegramID int64
}

type UserProfile struct {
	TelegramID  int64
	DisplayName string
	Username    string
}

type CreateExpenseInput struct {
	Amount   decimal.Decimal
	Category Category
	SpentAt  time.Time
	Comment  string
}

type UpdateExpenseInput struct {
	Amount   *decimal.Decimal
	Category *Category
	SpentAt  *time.Time
	Comment  *string
}

// ExpenseFilter defines an optional inclusive date range and pagination.
// A zero From or To means that the bound is not set.
type ExpenseFilter struct {
	From     time.Time
	To       time.Time
	Category *Category
	Limit    int
	Offset   int
}

// FamilyExpenseFilter adds an optional member filter for the family panel.
type FamilyExpenseFilter struct {
	UserID   int64
	From     time.Time
	To       time.Time
	Category *Category
	Limit    int
	Offset   int
}

// ExpensePage is a single page of expenses plus a hint that another page exists.
type ExpensePage struct {
	Items   []Expense
	HasMore bool
}

// ExpenseSummary is an aggregate over a filtered set of expenses.
type ExpenseSummary struct {
	Total decimal.Decimal
	Count int64
}
