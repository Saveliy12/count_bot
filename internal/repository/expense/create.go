package expense

import (
	"context"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) Create(ctx context.Context, expense model.Expense) (model.Expense, error) {
	const query = `
		INSERT INTO expenses (
			user_id,
			amount,
			category,
			spent_at,
			comment,
			created_at,
			updated_at
		)
		VALUES ($1, $2::numeric, $3, $4::date, NULLIF($5, ''), $6, $7)
		RETURNING ` + expenseColumns

	result, err := postgres.ScanExpense(r.db.QueryRowContext(
		ctx,
		query,
		expense.UserID,
		expense.Amount.String(),
		string(expense.Category),
		expense.SpentAt,
		expense.Comment,
		expense.CreatedAt,
		expense.UpdatedAt,
	))
	if err != nil {
		return model.Expense{}, fmt.Errorf("create expense: %w", err)
	}

	return result, nil
}
