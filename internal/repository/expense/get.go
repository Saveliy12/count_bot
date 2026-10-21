package expense

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) GetByID(ctx context.Context, id int64) (model.Expense, error) {
	const query = `
		SELECT ` + expenseColumns + `
		FROM expenses
		WHERE id = $1
	`

	expense, err := postgres.ScanExpense(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Expense{}, model.ErrNotFound
	}
	if err != nil {
		return model.Expense{}, fmt.Errorf("get expense by ID: %w", err)
	}

	return expense, nil
}
