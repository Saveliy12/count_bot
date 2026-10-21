package expense

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

func (r *Repository) SummaryByUser(
	ctx context.Context,
	userID int64,
	filter model.ExpenseFilter,
) (model.ExpenseSummary, error) {
	query := `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
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

	var (
		count    int64
		totalRaw string
	)
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count, &totalRaw); err != nil {
		return model.ExpenseSummary{}, fmt.Errorf("summarize expenses: %w", err)
	}

	total, err := decimal.NewFromString(totalRaw)
	if err != nil {
		return model.ExpenseSummary{}, fmt.Errorf("parse expense total: %w", err)
	}

	return model.ExpenseSummary{Total: total, Count: count}, nil
}

func (r *Repository) SummaryByFamily(
	ctx context.Context,
	familyID int64,
	filter model.FamilyExpenseFilter,
) (model.ExpenseSummary, error) {
	query := `
		SELECT COUNT(*), COALESCE(SUM(e.amount), 0)
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

	var (
		count    int64
		totalRaw string
	)
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count, &totalRaw); err != nil {
		return model.ExpenseSummary{}, fmt.Errorf("summarize family expenses: %w", err)
	}

	total, err := decimal.NewFromString(totalRaw)
	if err != nil {
		return model.ExpenseSummary{}, fmt.Errorf("parse family expense total: %w", err)
	}

	return model.ExpenseSummary{Total: total, Count: count}, nil
}
