package expense

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

func (s *Service) GetExpense(
	ctx context.Context,
	actor model.Actor,
	id int64,
) (model.Expense, error) {
	return s.getOwnedExpense(ctx, actor, id)
}

func (s *Service) GetFamilyExpense(ctx context.Context, id int64) (model.Expense, error) {
	return s.getFamilyExpense(ctx, id)
}

func (s *Service) getOwnedExpense(
	ctx context.Context,
	actor model.Actor,
	id int64,
) (model.Expense, error) {
	if err := validateActor(actor); err != nil {
		return model.Expense{}, err
	}
	if id <= 0 {
		return model.Expense{}, model.NewValidationError("id", "must be positive")
	}

	expense, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return model.Expense{}, err
	}
	if expense.UserID != actor.TelegramID {
		return model.Expense{}, model.ErrNotFound
	}

	return expense, nil
}

func (s *Service) getFamilyExpense(ctx context.Context, id int64) (model.Expense, error) {
	if id <= 0 {
		return model.Expense{}, model.NewValidationError("id", "must be positive")
	}

	family, err := s.families.GetDefault(ctx)
	if err != nil {
		return model.Expense{}, err
	}

	expense, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return model.Expense{}, err
	}

	user, err := s.users.GetByTelegramID(ctx, expense.UserID)
	if err != nil {
		return model.Expense{}, err
	}
	if user.FamilyID != family.ID {
		return model.Expense{}, model.ErrNotFound
	}

	return expense, nil
}
