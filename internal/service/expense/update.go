package expense

import (
	"context"
	"strings"

	"github.com/Saveliy12/expenses/internal/model"
)

func (s *Service) UpdateExpense(
	ctx context.Context,
	actor model.Actor,
	id int64,
	patch model.UpdateExpenseInput,
) (model.Expense, error) {
	if isEmptyPatch(patch) {
		return model.Expense{}, model.NewValidationError("patch", "at least one field is required")
	}

	expense, err := s.getOwnedExpense(ctx, actor, id)
	if err != nil {
		return model.Expense{}, err
	}

	return s.updateExpense(ctx, expense, patch)
}

func (s *Service) UpdateFamilyExpense(
	ctx context.Context,
	id int64,
	patch model.UpdateExpenseInput,
) (model.Expense, error) {
	if isEmptyPatch(patch) {
		return model.Expense{}, model.NewValidationError("patch", "at least one field is required")
	}

	expense, err := s.getFamilyExpense(ctx, id)
	if err != nil {
		return model.Expense{}, err
	}

	return s.updateExpense(ctx, expense, patch)
}

func (s *Service) updateExpense(
	ctx context.Context,
	expense model.Expense,
	patch model.UpdateExpenseInput,
) (model.Expense, error) {
	if patch.Amount != nil {
		expense.Amount = *patch.Amount
	}
	if patch.Category != nil {
		expense.Category = *patch.Category
	}
	if patch.SpentAt != nil {
		expense.SpentAt = startOfDay(*patch.SpentAt)
	}
	if patch.Comment != nil {
		expense.Comment = strings.TrimSpace(*patch.Comment)
	}
	expense.UpdatedAt = s.now().UTC()

	if err := model.ValidateExpense(expense); err != nil {
		return model.Expense{}, err
	}

	return s.expenses.Update(ctx, expense)
}
