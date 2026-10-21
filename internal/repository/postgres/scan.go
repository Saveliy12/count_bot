package postgres

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func ScanUser(row rowScanner) (model.User, error) {
	var user model.User

	if err := row.Scan(
		&user.TelegramID,
		&user.FamilyID,
		&user.DisplayName,
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return model.User{}, err
	}

	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()

	return user, nil
}

func ScanFamily(row rowScanner) (model.Family, error) {
	var family model.Family

	if err := row.Scan(
		&family.ID,
		&family.Name,
		&family.IsDefault,
		&family.CreatedAt,
	); err != nil {
		return model.Family{}, err
	}

	family.CreatedAt = family.CreatedAt.UTC()

	return family, nil
}

func ScanExpense(row rowScanner) (model.Expense, error) {
	var (
		expense  model.Expense
		amount   string
		category string
	)

	if err := row.Scan(
		&expense.ID,
		&expense.UserID,
		&amount,
		&category,
		&expense.SpentAt,
		&expense.Comment,
		&expense.CreatedAt,
		&expense.UpdatedAt,
	); err != nil {
		return model.Expense{}, err
	}

	parsedAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return model.Expense{}, fmt.Errorf("parse expense amount: %w", err)
	}

	expense.Amount = parsedAmount
	expense.Category = model.Category(category)
	expense.SpentAt = dateOnly(expense.SpentAt)
	expense.CreatedAt = expense.CreatedAt.UTC()
	expense.UpdatedAt = expense.UpdatedAt.UTC()

	return expense, nil
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
