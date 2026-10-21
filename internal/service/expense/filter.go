package expense

import (
	"time"

	"github.com/Saveliy12/expenses/internal/model"
)

func normalizeFilter(filter model.ExpenseFilter) (model.ExpenseFilter, error) {
	if filter.Limit == 0 {
		filter.Limit = model.DefaultPageSize
	}
	if filter.Limit < 0 {
		return model.ExpenseFilter{}, model.NewValidationError("limit", "must not be negative")
	}
	if filter.Limit > model.MaxPageSize {
		filter.Limit = model.MaxPageSize
	}
	if filter.Offset < 0 {
		return model.ExpenseFilter{}, model.NewValidationError("offset", "must not be negative")
	}
	if filter.Category != nil && !filter.Category.Valid() {
		return model.ExpenseFilter{}, model.NewValidationError("category", "unsupported category")
	}

	if !filter.From.IsZero() {
		filter.From = startOfDay(filter.From)
	}
	if !filter.To.IsZero() {
		filter.To = startOfDay(filter.To)
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		return model.ExpenseFilter{}, model.NewValidationError("period", "from must not be after to")
	}

	return filter, nil
}

func normalizeFamilyFilter(filter model.FamilyExpenseFilter) (model.FamilyExpenseFilter, error) {
	if filter.UserID < 0 {
		return model.FamilyExpenseFilter{}, model.NewValidationError("user_id", "must not be negative")
	}

	normalized, err := normalizeFilter(model.ExpenseFilter{
		From:     filter.From,
		To:       filter.To,
		Category: filter.Category,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	})
	if err != nil {
		return model.FamilyExpenseFilter{}, err
	}

	return model.FamilyExpenseFilter{
		UserID:   filter.UserID,
		From:     normalized.From,
		To:       normalized.To,
		Category: normalized.Category,
		Limit:    normalized.Limit,
		Offset:   normalized.Offset,
	}, nil
}

func startOfDay(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}

	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
