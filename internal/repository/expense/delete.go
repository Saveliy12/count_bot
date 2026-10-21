package expense

import (
	"context"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
)

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM expenses WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted expense count: %w", err)
	}
	if affected == 0 {
		return model.ErrNotFound
	}

	return nil
}

func (r *Repository) DeleteIfUnchanged(ctx context.Context, expense model.Expense) error {
	const query = `
		DELETE FROM expenses
		WHERE id = $1
		  AND user_id = $2
		  AND amount = $3::numeric
		  AND category = $4
		  AND spent_at = $5::date
		  AND COALESCE(comment, '') = $6
		  AND updated_at = $7
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		expense.ID,
		expense.UserID,
		expense.Amount.String(),
		string(expense.Category),
		expense.SpentAt,
		expense.Comment,
		expense.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("delete unchanged expense: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted expense count: %w", err)
	}
	if affected == 0 {
		return model.ErrConflict
	}

	return nil
}
