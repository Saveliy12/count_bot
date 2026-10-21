package model

import (
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

const MaxCommentLength = 500

var maxExpenseAmount = decimal.RequireFromString("9999999999.99")

type Expense struct {
	ID        int64
	UserID    int64
	Amount    decimal.Decimal
	Category  Category
	SpentAt   time.Time
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ValidateExpense(expense Expense) error {
	if expense.UserID <= 0 {
		return NewValidationError("user_id", "must be positive")
	}
	if !expense.Amount.IsPositive() {
		return NewValidationError("amount", "must be positive")
	}
	if expense.Amount.GreaterThan(maxExpenseAmount) {
		return NewValidationError("amount", "is too large")
	}
	if !expense.Amount.Equal(expense.Amount.Round(2)) {
		return NewValidationError("amount", "must have at most two decimal places")
	}
	if !expense.Category.Valid() {
		return NewValidationError("category", "unsupported category")
	}
	if expense.SpentAt.IsZero() {
		return NewValidationError("spent_at", "must not be empty")
	}
	if utf8.RuneCountInString(expense.Comment) > MaxCommentLength {
		return NewValidationError("comment", "is too long")
	}

	return nil
}
