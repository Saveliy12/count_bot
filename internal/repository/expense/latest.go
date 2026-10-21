package expense

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) LatestByUser(ctx context.Context, userID int64) (model.Expense, error) {
	return r.LatestByUserFiltered(ctx, userID, model.ExpenseFilter{})
}

func (r *Repository) LatestByUserFiltered(
	ctx context.Context,
	userID int64,
	filter model.ExpenseFilter,
) (model.Expense, error) {
	query := `
		SELECT ` + expenseColumns + `
		FROM expenses
		WHERE user_id = $1
	`
	args := []any{userID}

	if !filter.From.IsZero() {
		args = append(args, filter.From)
		query += fmt.Sprintf(" AND spent_at >= $%d::date", len(args))
	}
	if !filter.To.IsZero() {
		args = append(args, filter.To)
		query += fmt.Sprintf(" AND spent_at <= $%d::date", len(args))
	}
	if filter.Category != nil {
		args = append(args, string(*filter.Category))
		query += fmt.Sprintf(" AND category = $%d", len(args))
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT 1"

	expense, err := postgres.ScanExpense(r.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Expense{}, model.ErrNotFound
	}
	if err != nil {
		return model.Expense{}, fmt.Errorf("get latest filtered expense: %w", err)
	}

	return expense, nil
}
