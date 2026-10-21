package expense

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

func (s *Service) DeleteExpense(ctx context.Context, actor model.Actor, id int64) error {
	expense, err := s.getOwnedExpense(ctx, actor, id)
	if err != nil {
		return err
	}

	return s.expenses.Delete(ctx, expense.ID)
}

func (s *Service) DeleteExpenseIfUnchanged(
	ctx context.Context,
	actor model.Actor,
	expense model.Expense,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if expense.ID <= 0 {
		return model.NewValidationError("id", "must be positive")
	}
	if expense.UserID != actor.TelegramID {
		return model.ErrNotFound
	}

	return s.expenses.DeleteIfUnchanged(ctx, expense)
}

func (s *Service) DeleteFamilyExpense(ctx context.Context, id int64) error {
	expense, err := s.getFamilyExpense(ctx, id)
	if err != nil {
		return err
	}

	return s.expenses.Delete(ctx, expense.ID)
}
