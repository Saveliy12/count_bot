package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestValidateExpense(t *testing.T) {
	valid := Expense{
		UserID:   42,
		Amount:   decimal.NewFromInt(350),
		Category: CategoryFood,
		SpentAt:  time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Comment:  "кофе",
	}

	tests := []struct {
		name      string
		mutate    func(expense *Expense)
		wantField string
	}{
		{name: "valid"},
		{
			name: "missing user",
			mutate: func(expense *Expense) {
				expense.UserID = 0
			},
			wantField: "user_id",
		},
		{
			name: "zero amount",
			mutate: func(expense *Expense) {
				expense.Amount = decimal.Zero
			},
			wantField: "amount",
		},
		{
			name: "negative amount",
			mutate: func(expense *Expense) {
				expense.Amount = decimal.NewFromInt(-1)
			},
			wantField: "amount",
		},
		{
			name: "amount with three decimal places",
			mutate: func(expense *Expense) {
				expense.Amount = decimal.RequireFromString("10.999")
			},
			wantField: "amount",
		},
		{
			name: "amount is too large",
			mutate: func(expense *Expense) {
				expense.Amount = decimal.RequireFromString("10000000000.00")
			},
			wantField: "amount",
		},
		{
			name: "invalid category",
			mutate: func(expense *Expense) {
				expense.Category = "crypto"
			},
			wantField: "category",
		},
		{
			name: "missing date",
			mutate: func(expense *Expense) {
				expense.SpentAt = time.Time{}
			},
			wantField: "spent_at",
		},
		{
			name: "comment is too long",
			mutate: func(expense *Expense) {
				expense.Comment = strings.Repeat("я", MaxCommentLength+1)
			},
			wantField: "comment",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expense := valid
			if test.mutate != nil {
				test.mutate(&expense)
			}

			err := ValidateExpense(expense)
			if test.wantField == "" {
				if err != nil {
					t.Fatalf("validate expense: %v", err)
				}
				return
			}

			assertValidationField(t, err, test.wantField)
		})
	}
}

func assertValidationField(t *testing.T, err error, wantField string) {
	t.Helper()

	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Field != wantField {
		t.Fatalf("expected field %q, got %q", wantField, validationErr.Field)
	}
}
