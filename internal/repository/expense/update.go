package expense

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) Update(ctx context.Context, expense model.Expense) (model.Expense, error) {
	const query = `
		UPDATE expenses
		SET amount = $2::numeric,
		    category = $3,
		    spent_at = $4::date,
		    comment = NULLIF($5, ''),
		    updated_at = $6
		WHERE id = $1 AND user_id = $7
		RETURNING ` + expenseColumns

	result, err := postgres.ScanExpense(r.db.QueryRowContext(
		ctx,
		query,
		expense.ID,
		expense.Amount.String(),
		string(expense.Category),
		expense.SpentAt,
		expense.Comment,
		expense.UpdatedAt,
		expense.UserID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Expense{}, model.ErrNotFound
	}
	if err != nil {
		return model.Expense{}, fmt.Errorf("update expense: %w", err)
	}

	return result, nil
}
