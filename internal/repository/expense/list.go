package expense

import (
	"context"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) ListByUser(
	ctx context.Context,
	userID int64,
	filter model.ExpenseFilter,
) (model.ExpensePage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultPageSize
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

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

	args = append(args, limit+1, offset)
	query += fmt.Sprintf(" ORDER BY spent_at DESC, id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return model.ExpensePage{}, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		expense, err := postgres.ScanExpense(rows)
		if err != nil {
			return model.ExpensePage{}, fmt.Errorf("scan expense: %w", err)
		}
		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return model.ExpensePage{}, fmt.Errorf("iterate expenses: %w", err)
	}

	return expensePage(expenses, limit), nil
}

func (r *Repository) ListByFamily(
	ctx context.Context,
	familyID int64,
	filter model.FamilyExpenseFilter,
) (model.ExpensePage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultPageSize
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT ` + expenseColumnsWithAlias + `
		FROM expenses e
		JOIN users u ON u.telegram_id = e.user_id
		WHERE u.family_id = $1
	`
	args := []any{familyID}

	if filter.UserID > 0 {
		args = append(args, filter.UserID)
		query += fmt.Sprintf(" AND e.user_id = $%d", len(args))
	}
	if !filter.From.IsZero() {
		args = append(args, filter.From)
		query += fmt.Sprintf(" AND e.spent_at >= $%d::date", len(args))
	}
	if !filter.To.IsZero() {
		args = append(args, filter.To)
		query += fmt.Sprintf(" AND e.spent_at <= $%d::date", len(args))
	}
	if filter.Category != nil {
		args = append(args, string(*filter.Category))
		query += fmt.Sprintf(" AND e.category = $%d", len(args))
	}

	args = append(args, limit+1, offset)
	query += fmt.Sprintf(
		" ORDER BY e.spent_at DESC, e.id DESC LIMIT $%d OFFSET $%d",
		len(args)-1,
		len(args),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return model.ExpensePage{}, fmt.Errorf("list family expenses: %w", err)
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		expense, err := postgres.ScanExpense(rows)
		if err != nil {
			return model.ExpensePage{}, fmt.Errorf("scan family expense: %w", err)
		}
		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return model.ExpensePage{}, fmt.Errorf("iterate family expenses: %w", err)
	}

	return expensePage(expenses, limit), nil
}

// expensePage trims the probe row that is selected to detect the next page.
func expensePage(expenses []model.Expense, limit int) model.ExpensePage {
	page := model.ExpensePage{Items: expenses}
	if len(expenses) > limit {
		page.Items = expenses[:limit]
		page.HasMore = true
	}

	return page
}
